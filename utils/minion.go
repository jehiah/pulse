package pulse

//This is a worker

import (
	"crypto/tls"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"time"
)

var version string
var pinger *Pinger

type Pinger struct {
	Last time.Time
}

//The goal here is cnc will ping the minion regularly...
//if ping is not received then minion should revreate the connection...
func (p *Pinger) Ping(host, out *bool) error {
	log.Println("Got pung")
	p.Last = time.Now()
	*out = true
	return nil
}

type serverflag []string

func (i *serverflag) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *serverflag) Set(value string) error {
	//fmt.Printf("hdr %s\n", value)
	//m := *i
	*i = append(*i, value)
	return nil
}

func listen(cnc string, servers []string, cfg *tls.Config) {
	dialer := new(net.Dialer)
	dialer.Timeout = time.Minute

	conn, err := tls.DialWithDialer(dialer, "tcp", cnc, cfg)
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second * 5)
		return
	}
	//log.Println(conn)
	//conn.SetKeepAlive(true)
	//conn.SetKeepAlivePeriod(time.Minute)
	pinger.Last = time.Now()
	var signal bool
	go func(c *tls.Conn) {
		for {
			if signal {
				log.Println("Returning because of signal")
				return
			}
			time.Sleep(time.Second * 20)
			d := time.Since(pinger.Last)
			log.Println("Since last ping", d)
			if d > time.Minute {
				c.Close()
				return
			}
		}
	}(conn)
	rpc.ServeConn(conn)
	signal = true
}

// If new version is available... commit suicide.
func versionsuicide() {
	localversion := strings.TrimSpace(version)
	//start := time.Now()
	for {
		//if time.Since(start) > time.Hour*24 {
		//	log.Fatal("Suiciding")
		//}
		resp, err := http.Get("https://tb-minion.turbobytes.net/latest")
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				strbody := strings.TrimSpace(string(body))
				if strbody != localversion {
					log.Fatal("New version " + strbody + " is available. currently using " + localversion)
					os.Remove("current")
				} else {
					log.Println("On latest version")
				}
			} else {
				log.Println(err)
			}
		} else {
			log.Println(err)
		}
		time.Sleep(time.Minute * 5) //Sleep 5 mins...

	}
}

func Runminion(cnc, caFile, certificateFile, privateKeyFile, reqFile, ver string, servers []string) error {
	gob.RegisterName("github.com/turbobytes/pulse/utils.MtrRequest", MtrRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.MtrResult", MtrResult{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.CurlRequest", CurlRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.CurlResult", CurlResult{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.DNSRequest", DNSRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.DNSResult", DNSResult{})
	servers = []string{"8.8.8.8:53", "208.67.222.222:53"}

	log.Println("servers", servers)
	version = ver
	if version == "" {
		log.Println("No version information provided, not doing autoupdate")
		version = "dirty"
	} else {
		go versionsuicide()
	}

	resolver := new(Resolver)
	resolver.Servers = servers
	resolver.Version = version
	pinger = &Pinger{}
	rpc.Register(resolver)
	rpc.Register(pinger)

	// If CA certificate does not exist where expected, download from S3
	if _, err := os.Stat(caFile); os.IsNotExist(err) {
		log.Println("CA cert not found ", privateKeyFile)
		log.Println("downloading..")
		resp, err := http.Get("https://tb-minion.turbobytes.net/ca.crt")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			errors.New("Status was not 200!")
		}
		f, err := os.Create(caFile)
		_, err = io.Copy(f, resp.Body)
		f.Close()
		if err != nil {
			return err
		}
	}

	// If private key does not exist where expected, create it.
	if _, err := os.Stat(privateKeyFile); os.IsNotExist(err) {
		log.Println("Private key file not found ", privateKeyFile)
		log.Println("generating..")
		GeneratePrivKeyFile(privateKeyFile)
	}

	// If Certificate file does not exist where expected, generate a CSR to send.
	if _, err := os.Stat(certificateFile); os.IsNotExist(err) {
		log.Println("Certificate file not found ", certificateFile)
		log.Println("generating..")
		//Hmm ... create a full blown CSR... or just send pub key...
		hash := PrintCertRequest(privateKeyFile, reqFile)
		//Lets see with S3 if Cert is available there...
		log.Println("Checking if certificate has been uploaded yet...")
		url := "https://tb-minion.turbobytes.net/certs/" + hash + ".crt"
		resp, err := http.Get(url)
		if err == nil {
			if resp.StatusCode == 200 {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				f, err := os.Create(certificateFile)
				if err != nil {
					//Permission issue?
					return err
				}
				f.Write(body)
				f.Close()
			} else {
				//404 or 403 cause cert not yet uploaded
				return errors.New(resp.Status)
			}
		} else {
			//Error contacting S3, FAIL here because we know cert is missing
			return err
		}
	}

	for {
		//Infinite loop... i.e. reconnect when booboo
		cfg := GetTLSConfig(caFile, certificateFile, privateKeyFile)
		listen(cnc, servers, cfg)
	}
	return nil
}
