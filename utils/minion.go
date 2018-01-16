package pulse

//This is a worker

import (
	"crypto/tls"
	"encoding/gob"
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

func listen(cnc string, cfg *tls.Config) {
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
	for range time.Tick(time.Minute * 5) {
		//if time.Since(start) > time.Hour*24 {
		//	log.Fatal("Suiciding")
		//}
		v, err := expectedVersion()
		if err != nil {
			log.Println(err)
		} else if v != localversion {
			log.Fatalf("New version %s is available. currently using %s", v, localversion)
			// unreachable
			os.Remove("current")
		} else {
			log.Println("On latest version", localversion)
		}
	}
}

// expectedVersion gets the expected versoin that should be running
func expectedVersion() (string, error) {
	resp, err := http.Get("https://tb-minion.turbobytes.net/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Got status code %d expected 200", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func Runminion(cnc, caFile, certificateFile, privateKeyFile, reqFile, ver string) error {
	gob.RegisterName("github.com/turbobytes/pulse/utils.MtrRequest", MtrRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.MtrResult", MtrResult{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.CurlRequest", CurlRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.CurlResult", CurlResult{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.DNSRequest", DNSRequest{})
	gob.RegisterName("github.com/turbobytes/pulse/utils.DNSResult", DNSResult{})

	version = ver
	if version == "" {
		log.Println("No version information provided, not doing autoupdate")
		version = "dirty"
	} else {
		go versionsuicide()
	}

	resolver := new(Resolver)
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
			return fmt.Errorf("Got status code %d expected 200", resp.StatusCode)
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
		//Error contacting S3, FAIL here because we know cert is missing
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			//404 or 403 cause cert not yet uploaded
			return fmt.Errorf("Got status code %d expected 200", resp.StatusCode)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(certificateFile, body, 0666)
		if err != nil {
			//Permission issue?
			return err
		}
	}

	for {
		//Infinite loop... i.e. reconnect when booboo
		cfg := GetTLSConfig(caFile, certificateFile, privateKeyFile)
		listen(cnc, cfg)
	}
	return nil
}
