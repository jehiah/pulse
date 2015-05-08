package main

//This is command and control tool

import (
	"compress/gzip"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"flag"
	"github.com/abh/geoip"
	"github.com/miekg/dns"
	"github.com/turbobytes/pulse/utils"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"sync"
	"time"
)

//type Resolver int
var gia *geoip.GeoIP
var session *mgo.Session

//AgentInfo is what we store in db...
type AgentInfo struct {
	Name           string
	City           string
	State          string
	Country        string
	SerialNumber   *big.Int
	LocalResolvers []string
}

func (agent *AgentInfo) GetBSON() (interface{}, error) {
	return bson.D{
		{"Name", agent.Name},
		{"City", agent.City},
		{"State", agent.State},
		{"Country", agent.Country},
		{"LocalResolvers", strings.Join(agent.LocalResolvers, ",")},
		{"_id", agent.SerialNumber.String()},
	}, nil
}

func (agent *AgentInfo) SetBSON(raw bson.Raw) error {
	data := make(map[string]string)
	err := raw.Unmarshal(data)
	if err != nil {
		return err
	}
	agent.Name = data["Name"]
	agent.City = data["City"]
	agent.State = data["State"]
	agent.Country = data["Country"]
	agent.LocalResolvers = strings.Split(data["LocalResolvers"], ",")
	agent.SerialNumber = new(big.Int)
	agent.SerialNumber.SetString(data["_id"], 10)
	return nil
}

type Worker struct {
	Client    *rpc.Client
	IP        string
	Geo       string   //TODO: Make richer
	Resolvers []string //List of resolvers this worker supports
	Name      string
	ASN       *string
	ASName    *string
	State     string
	Country   string
	City      string
	Serial    *big.Int
}

func getasn(ip string) (*string, *string) {
	asntmp, _ := gia.GetName(ip)
	if asntmp != "" {
		splitted := strings.SplitN(asntmp, " ", 2)
		if len(splitted) == 2 {
			return &splitted[0], &splitted[1]
		}
	}
	return nil, nil
}

func populatedata(w *Worker) {
	c := session.DB("dnsdist").C("agents")
	agent := new(AgentInfo)
	err := c.Find(bson.M{"_id": w.Serial.String()}).One(agent)
	if err == mgo.ErrNotFound {
		agent.Name = w.Name
		agent.SerialNumber = w.Serial
		err1 := c.Insert(agent)
		if err1 != nil {
			log.Fatal(err1)
		}
	} else if err != nil {
		log.Fatal(err)
	}
	w.Name = agent.Name
	w.City = agent.City
	w.State = agent.State
	w.Country = agent.Country
	w.Resolvers = agent.LocalResolvers
}

func NewWorker(conn net.Conn) *Worker {
	w := &Worker{}
	w.Client = rpc.NewClient(conn)
	w.IP = strings.Split(conn.RemoteAddr().String(), ":")[0]
	//TODO: Authenticate and fetch capabilities
	tlsconn, ok := conn.(*tls.Conn)
	if !ok {
		log.Println("Not TLS Conn")
	} else {
		w.ASN, w.ASName = getasn(w.IP)

		err := pingworker(w) //Ping in begining to make sure we can talk and trigger handshake
		if err == nil {
			state := tlsconn.ConnectionState()
			if len(state.PeerCertificates) > 0 {
				w.Name = state.PeerCertificates[0].Subject.CommonName
				serial := state.PeerCertificates[0].SerialNumber
				log.Println(serial)
				w.Serial = serial
				log.Println(w)
				populatedata(w)
				log.Println(w)
				return w
			}
		}
	}
	return nil
}

type Tracker struct {
	workers    map[string]*Worker
	workerlock *sync.RWMutex
}

func NewTracker() *Tracker {
	t := &Tracker{}
	t.workerlock = &sync.RWMutex{}
	t.workers = make(map[string]*Worker)
	go t.Pinger()
	return t
}

func (tracker *Tracker) Register(conn net.Conn) {
	worker := NewWorker(conn)
	if worker != nil {
		tracker.workerlock.Lock()
		tracker.workers[conn.RemoteAddr().String()] = worker
		tracker.workerlock.Unlock()
	}
}

func (tracker *Tracker) UnRegister(worker *Worker) {
	tracker.workerlock.Lock()
	defer tracker.workerlock.Unlock()
	//Copy all except this one
	for k, w := range tracker.workers {
		if worker == w {
			delete(tracker.workers, k)
		}
	}
	//tracker.workers = newworkers
}

func pingworker(worker *Worker) error {
	var reply bool
	err := worker.Client.Call("Pinger.Ping", true, &reply)
	if err == rpc.ErrShutdown {
		go tracker.UnRegister(worker) //Async cause of locking
		log.Println("Unregistering from tracker")
	} else if err != nil {
		log.Println("pinger", err)
	}
	return err
}

func (tracker *Tracker) SendPings() {
	tracker.workerlock.RLock()
	defer tracker.workerlock.RUnlock()
	for _, worker := range tracker.workers {
		go pingworker(worker)
	}

}

func (tracker *Tracker) Pinger() {
	for {
		time.Sleep(time.Second * 20)
		tracker.SendPings()
	}
}

func addresolvers(args pulse.DNSRequest, resolvers []string) {

}

func (tracker *Tracker) Runner(req *pulse.CombinedRequest) []*pulse.CombinedResult {
	tracker.workerlock.RLock()
	defer tracker.workerlock.RUnlock()
	results := make([]*pulse.CombinedResult, 0)
	n := len(tracker.workers)
	rchan := make(chan *pulse.CombinedResult, n)
	var originalargs pulse.DNSRequest
	if req.Type == pulse.TypeDNS {
		args, ok := req.Args.(pulse.DNSRequest)
		if ok {
			originalargs = args
		}
	}
	for ip, worker := range tracker.workers {
		go func(worker *Worker, ip string) {
			log.Println(ip, worker)
			var reply *pulse.CombinedResult
			//TODO: Implement timeout
			//If CombinedRequest is of type TypeDNS and taget is not specified... then insert defaults for worker...
			if req.Type == pulse.TypeDNS {
				args, ok := req.Args.(pulse.DNSRequest)
				if ok {
					if len(originalargs.Targets) == 0 {
						args.Targets = []string{"8.8.8.8:53", "208.67.222.222:53"}
						for _, resolver := range worker.Resolvers {
							if resolver != "" {
								args.Targets = append(args.Targets, resolver+":53")
							}
						}
						req.Args = args
					}
				}
			}
			call := worker.Client.Go("Resolver.Combined", req, &reply, nil)
			select {
			case replyCall := <-call.Done:
				log.Println(ip)
				if replyCall.Error == rpc.ErrShutdown {
					go tracker.UnRegister(worker) //Async cause of locking
					log.Println("Unregistering from tracker")
					rchan <- nil
				} else if replyCall.Error != nil {
					log.Println(replyCall.Error)
					rchan <- nil
				} else {
					//reply.Name += " (" + strings.Split(ip, ":")[0] + ")"
					iponly := strings.Split(ip, ":")[0]
					splitted := strings.Split(iponly, ".")
					splitted[3] = "0"
					reply.Agent = strings.Join(splitted, ".")
					reply.Name = worker.Name //Insert in this workers Common Name here
					reply.ASN = worker.ASN
					reply.ASName = worker.ASName
					reply.City = worker.City
					reply.State = worker.State
					reply.Country = worker.Country
					reply.Id = worker.Serial
					//log.Println(reply.Name)
					rchan <- reply
				}
				return
			case <-time.After(time.Minute):
				go tracker.UnRegister(worker) //Nuke the turtle...
				rchan <- nil
				return
			}
		}(worker, ip)
	}

	for i := 0; i < n; i++ {
		log.Println(i, "of", n)
		reply := <-rchan
		if reply != nil {
			log.Println(reply.Name)
			results = append(results, reply)
		}
	}
	return results
}

var tracker *Tracker

//https://gist.github.com/the42/1956518

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func makeGzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Origin"), "https://my.turbobytes.com") {
			w.Header().Set("Access-Control-Allow-Origin", "https://my.turbobytes.com")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		}
		if strings.Contains(r.Header.Get("Origin"), "http://127.0.0.1:8000") {
			w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:8000")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		}
		if r.Method == "OPTIONS" {
			return
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func runcurl(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(data))
	req := &pulse.CurlRequest{}
	err = json.Unmarshal(data, req)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(req)
	creq := &pulse.CombinedRequest{
		Type:        pulse.TypeCurl,
		Args:        req,
		RequestedAt: time.Now(),
	}
	results := tracker.Runner(creq)
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Println(err)
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func runmtr(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(data))
	req := pulse.MtrRequest{}
	err = json.Unmarshal(data, &req)
	if err != nil {
		log.Println(err)
		return
	}
	creq := &pulse.CombinedRequest{
		Type:        pulse.TypeMTR,
		Args:        req,
		RequestedAt: time.Now(),
	}
	log.Println(req)
	results := tracker.Runner(creq)
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Println(err)
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func runtest(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(data))
	req := pulse.DNSRequest{}
	err = json.Unmarshal(data, &req)
	if err != nil {
		log.Println(err)
		return
	}
	if !strings.HasSuffix(req.Host, ".") {
		//Make FQDN
		req.Host = req.Host + "."
	}
	if req.Targets != nil {
		if len(req.Targets) > 0 {
			for i, t := range req.Targets {
				req.Targets[i] = t + ":53"
			}
		}
	}
	creq := &pulse.CombinedRequest{
		Type:        pulse.TypeDNS,
		Args:        req,
		RequestedAt: time.Now(),
	}
	log.Println(req)
	results := tracker.Runner(creq)

	//newresult := make(&pulse.CombinedResult, len(results))

	for i, res := range results {
		result, _ := res.Result.(pulse.DNSResult)
		for j, item := range result.Results {
			item.ASN, item.ASName = getasn(item.Server)
			msg := &dns.Msg{}
			msg.Unpack(item.Raw)
			item.Formated = msg.String()
			item.Msg = msg
			result.Results[j] = item
		}
		results[i].Result = result
	}
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func main() {
	gob.RegisterName("github.com/turbobytes/pulse/utils.MtrRequest", pulse.MtrRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.MtrResult", pulse.MtrResult{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.CurlRequest", pulse.CurlRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.CurlResult", pulse.CurlResult{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.DNSRequest", pulse.DNSRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.DNSResult", pulse.DNSResult{})
	tracker = NewTracker()
	var err error
	session, err = mgo.Dial("localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	gia, err = geoip.OpenType(geoip.GEOIP_ASNUM_EDITION)
	if err != nil {
		log.Fatal("Could not open GeoIP database\n")
	}

	var caFile, certificateFile, privateKeyFile string
	flag.StringVar(&caFile, "ca", "ca.crt", "Path to CA")
	flag.StringVar(&certificateFile, "crt", "server.crt", "Path to Server Certificate")
	flag.StringVar(&privateKeyFile, "key", "server.key", "Path to Private key")
	flag.Parse()
	cfg := pulse.GetTLSConfig(caFile, certificateFile, privateKeyFile)

	listener, err := tls.Listen("tcp", ":7777", cfg)
	if err != nil {
		log.Fatal(err)
	}
	go func() {

		http.HandleFunc("/", makeGzipHandler(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "index-dist.html")
		}))
		http.HandleFunc("/dns/", makeGzipHandler(runtest))
		http.HandleFunc("/curl/", makeGzipHandler(runcurl))
		http.HandleFunc("/mtr/", makeGzipHandler(runmtr))

		log.Fatal(http.ListenAndServe(":7778", nil))

	}()
	log.Println("monitoring")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go tracker.Register(conn) //Async cause this pings also
		log.Println(conn.RemoteAddr(), "at your service")
		//workers[worker.addr.String()] = worker
	}
}
