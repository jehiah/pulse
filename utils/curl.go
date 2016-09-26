package pulse

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	//	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"time"
)

//User-Agents :-
//	TurboBytes-Pulse/1.1 = http implementation doing things by hand over the wire
//	TurboBytes-Pulse/1.2 = http implementation using net/http.Client and httptrace

const (
	useragent = "TurboBytes-Pulse/1.2" //Default user agent
)

var (
	tlshandshaketimeout = time.Second * 15 //Timeout for TLS handshake
	dialtimeout         = time.Second * 15 //Timeout for Dial (DNS + TCP connect)
	responsetimeout     = time.Second * 30 //Time out for response header
	keepalive           = time.Second * 30 //Keepalive timeout
)

type CurlResult struct {
	Status          int                  //HTTP status of result
	Header          http.Header          //Headers
	Remote          string               //Remote IP the connection was made to
	Err             string               //Any Errors that happened. Usually for DNS fail or connection errors.
	Proto           string               //Response protocol
	StatusStr       string               //Status in stringified form
	DialTime        time.Duration        //Time it took for DNS + TCP connect.
	DNSTime         time.Duration        //Time it took for DNS.
	ConnectTime     time.Duration        //Time it took for  TCP connect.
	TLSTime         time.Duration        //Time it took for TLS handshake when running in SSL mode
	Ttfb            time.Duration        //Time it took since sending GET and getting results : total time minus DialTime minus TLSTime
	DialTimeStr     string               //Stringified
	DNSTimeStr      string               //Stringified
	ConnectTimeStr  string               //Stringified
	TLSTimeStr      string               //Stringified
	TtfbStr         string               //Stringified
	ConnectionState *tls.ConnectionState //Additional TLS data when running test over https. We snip out PublicKey from the certs cause they dont serialize well.
}

type CurlRequest struct {
	Path        string
	Endpoint    string
	Host        string
	Ssl         bool
	AgentFilter []*big.Int
}

type conInfo struct {
	DNS     time.Duration
	Connect time.Duration
	SSL     time.Duration
	TTFB    time.Duration
	Total   time.Duration
	//Transfer    time.Duration No Transfer time because we don't consume body
	Addr string
}

type conTrack struct {
	DNSStart             time.Time
	DNSDone              time.Time
	ConnectStart         map[string]time.Time
	ConnectDone          map[string]time.Time
	Addr                 string
	WroteRequest         time.Time
	GotFirstResponseByte time.Time
}

func (ct *conTrack) getConInfo() *conInfo {
	ci := &conInfo{
		Addr: ct.Addr,
	}
	if ct.GotFirstResponseByte.After(ct.WroteRequest) {
		ci.TTFB = ct.GotFirstResponseByte.Sub(ct.WroteRequest)
	}
	if ct.DNSDone.After(ct.DNSStart) {
		ci.DNS = ct.DNSDone.Sub(ct.DNSStart)
	}
	if ct.Addr == "" && len(ct.ConnectStart) > 0 { //If no addr(cause FAIL) but map has key(s) use any
		for ct.Addr, _ = range ct.ConnectStart {
			//log.Println(ct.Addr)
		}
	}
	cs := ct.ConnectStart[ct.Addr]
	cd, ok := ct.ConnectDone[ct.Addr]
	if !ok {
		cd = time.Now() //If connect was never Done then use now to indicate how long we waited...
	}
	if cd.After(cs) {
		ci.Connect = cd.Sub(cs)
	}
	if ct.WroteRequest.After(cd) {
		ci.SSL = ct.WroteRequest.Sub(cd)
	}
	ci.Total = ci.DNS + ci.Connect + ci.SSL + ci.TTFB
	return ci
}

func dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	con, err := (&net.Dialer{
		Timeout:   dialtimeout, //DNS + Connect
		KeepAlive: keepalive,
	}).DialContext(ctx, network, address)
	if err == nil {
		//If a connection could be established, ensure its not local
		a, _ := con.RemoteAddr().(*net.TCPAddr)

		if islocalip(a.IP) {
			fmt.Println(a.IP)
			con.Close()
			return nil, securityerr
		}
	}
	return con, err
}

func CurlImpl(r *CurlRequest) *CurlResult {
	result := &CurlResult{}
	var url string
	if r.Ssl {
		url = fmt.Sprintf("https://%s%s", r.Endpoint, r.Path)
	} else {
		url = fmt.Sprintf("http://%s%s", r.Endpoint, r.Path)
	}
	//Create a request object
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.Err = err.Error()
		return result
	}
	req.Header.Set("User-Agent", useragent)
	//Override Host header if needed
	tlshost := r.Endpoint //Validate with endpoint if no host given
	if r.Host != "" {
		req.Host = r.Host
		tlshost = r.Host //Validate with Host hdr if present
	}

	// Currently the transport leaks FD because currently http2
	// does not respect IdleConnTimeout
	// https://github.com/golang/go/issues/16808

	//Configure our transport, new one for each request
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialContext,
		MaxIdleConns:          100,              //Irrelevant
		IdleConnTimeout:       90 * time.Second, //Irrelevant
		TLSHandshakeTimeout:   tlshandshaketimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Due to #16808, transport going out of scope does not cleanup
	// idle connections. We must do it by hand using CloseIdleConnections()
	defer func() {
		// There is something racey going on, noticed an issue on my dev machine
		// but not on prod. Does not hurt to sleep for a sec.
		time.Sleep(time.Second)
		transport.CloseIdleConnections()
	}()

	//Initialize our client
	client := http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}, //Since we now use high-level client we must stop redirects.
	}

	// A dilema... Configuring our own TLSClientConfig causes http
	// package to not kick in http2. Doing http2.ConfigureTransport
	// manually messes up httptrace. As a workaround we ho not configure
	// TLSClientConfig at first, then, if needed, we do a mock request
	// to fire off transport.onceSetNextProtoDefaults() and then sneak
	// in the transport.TLSClientConfig.ServerName that we want to configure.

	if r.Ssl {
		if tlshost != r.Endpoint {
			//Begin hacky workaround...
			//Make mock req to kick in onceSetNextProtoDefaults()
			server := httptest.NewServer(http.HandlerFunc(http.NotFound))
			reqtmp, _ := http.NewRequest("GET", server.URL, nil)
			client.Do(reqtmp) //Don't care about response..
			//Now mess with TLSClientConfig
			transport.TLSClientConfig.ServerName = tlshost
			//Not closing server was causing FD leak in prod but not in dev.. weird
			server.Close()
		}
	}

	//Initialize connection tracker
	ct := &conTrack{
		ConnectStart: make(map[string]time.Time),
		ConnectDone:  make(map[string]time.Time),
	}
	//Initialize httptrace
	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			ct.Addr = connInfo.Conn.RemoteAddr().String()
			//log.Println(ct.Addr)
		},
		DNSStart: func(ds httptrace.DNSStartInfo) {
			ct.DNSStart = time.Now()
		},
		DNSDone: func(dd httptrace.DNSDoneInfo) {
			ct.DNSDone = time.Now()
		},
		ConnectStart: func(network, addr string) {
			ct.ConnectStart[addr] = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			ct.ConnectDone[addr] = time.Now()
		},
		GotFirstResponseByte: func() {
			ct.GotFirstResponseByte = time.Now()
		},
		WroteRequest: func(wr httptrace.WroteRequestInfo) {
			ct.WroteRequest = time.Now()
		},
	}
	//Wrap trace into req
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	//Make the request
	resp, err := client.Do(req)
	ti := ct.getConInfo()
	//log.Println(ct, ti)
	//populate the result with timing info regardless of failure
	result.Remote = ti.Addr
	result.DialTime = ti.DNS + ti.Connect
	result.DNSTime = ti.DNS
	result.ConnectTime = ti.Connect
	result.TLSTime = ti.SSL
	result.Ttfb = ti.TTFB
	result.DialTimeStr = result.DialTime.String()
	result.DNSTimeStr = result.DNSTime.String()
	result.ConnectTimeStr = result.ConnectTime.String()
	result.TLSTimeStr = result.TLSTime.String()
	result.TtfbStr = result.Ttfb.String()

	//On error stamp err and return
	if err != nil {
		result.Err = err.Error()
		return result
	}
	resp.Body.Close()
	//Not a fail, extract more info
	result.Status = resp.StatusCode
	result.StatusStr = resp.Status
	result.Header = resp.Header
	result.Proto = resp.Proto
	//log.Println(resp)
	//Finally do the connectionstate things...
	cstate := resp.TLS
	if cstate != nil {
		//Remove PublicKey from certs
		for i, cert := range cstate.PeerCertificates {
			tmpcert := &x509.Certificate{}
			*tmpcert = *cert
			tmpcert.PublicKey = "removed" //We need to do this for now cause its PITA to serialize it
			cstate.PeerCertificates[i] = tmpcert
		}
		for i, chain := range cstate.VerifiedChains {
			for j, cert := range chain {
				tmpcert := &x509.Certificate{}
				*tmpcert = *cert
				tmpcert.PublicKey = "removed" //We need to do this for now cause its PITA to serialize it
				cstate.VerifiedChains[i][j] = tmpcert
			}
		}
		result.ConnectionState = cstate
	}
	return result
}
