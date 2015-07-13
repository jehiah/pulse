package pulse

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type CurlResult struct {
	Status      int           //HTTP status of result
	Header      http.Header   //Headers
	Remote      string        //Remote IP the connection was made to
	Err         string        //Any Errors that happened. Usually for DNS fail or connection errors.
	Proto       string        //Response protocol
	StatusStr   string        //Status in stringified form
	DialTime    time.Duration //Time it took for DNS + TCP connect. TODO: Split DNS and Connect
	TLSTime     time.Duration //Time it took for TLS handshake when running in SSL mode
	Ttfb        time.Duration //Time it took since sending GET and getting results : total time minus DialTime minus TLSTime
	DialTimeStr string        //Stringified
	TLSTimeStr  string        //Stringified
	TtfbStr     string        //Stringified
}

type CurlRequest struct {
	Path     string
	Endpoint string
	Host     string
	Ssl      bool
}

type myDialer struct {
	RemoteStr           string
	DialTime            time.Duration
	TLSTime             time.Duration
	TLSClientConfig     *tls.Config
	TLSHandshakeTimeout time.Duration
}

//TODO: Split out DNS and Connect times. Which means we need to
//do DNS by hand... not rely on net.Dialer for that.
//ref: http://golang.org/src/net/dial.go?s=4639:4699#L147
func (md *myDialer) Dial(network, address string) (net.Conn, error) {
	timer := time.Now()
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	con, err := dialer.Dial(network, address)
	if err == nil {
		md.RemoteStr = con.RemoteAddr().String()
		a, _ := con.RemoteAddr().(*net.TCPAddr)

		if islocalip(a.IP) {
			fmt.Println(a.IP)
			con.Close()
			return nil, securityerr
		}

	}
	md.DialTime = time.Since(timer)
	log.Println(md.DialTime)
	return con, err
}

//Doing DialTLS by hand, and initiating Handshake() just so to get the
//TLS time.
func (md *myDialer) DialTLS(network, address string) (net.Conn, error) {
	con, err := md.Dial(network, address)
	tlstimer := time.Now()
	if err != nil {
		return con, err
	}
	tcon := tls.Client(con, md.TLSClientConfig)
	errc := make(chan error, 2)
	var timer *time.Timer
	timer = time.AfterFunc(md.TLSHandshakeTimeout, func() {
		errc <- tlsHandshakeTimeoutError
	})
	go func() {
		err := tcon.Handshake()
		if timer != nil {
			timer.Stop()
		}
		errc <- err
	}()
	err = <-errc
	if err != nil {
		con.Close()
		//return nil, err
	}
	md.TLSTime = time.Since(tlstimer)
	//err = tcon.Handshake()
	log.Println(md.TLSTime)
	return tcon, err
}

func CurlImpl(r *CurlRequest) *CurlResult {
	var url string
	if r.Ssl {
		url = fmt.Sprintf("https://%s%s", r.Endpoint, r.Path)
	} else {
		url = fmt.Sprintf("http://%s%s", r.Endpoint, r.Path)
	}
	log.Println(url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &CurlResult{0, nil, "", err.Error(), "", "", time.Duration(0), time.Duration(0), time.Duration(0), "", "", ""}
	}
	tlshost := r.Endpoint //Validate with endpoint if no host given
	if r.Host != "" {
		req.Host = r.Host
		tlshost = r.Host //Validate with Host hdr if present
	}
	tlscfg := &tls.Config{
		MinVersion: tls.VersionTLS10, //TLS 1.0 minimum. Depricating SSLv3 RFC 7568
		ServerName: tlshost,          //Override the hostname to validate
	}
	md := &myDialer{
		TLSHandshakeTimeout: 15 * time.Second,
		TLSClientConfig:     tlscfg,
	}
	MyTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DisableKeepAlives:     true,
		Dial:                  md.Dial,
		DialTLS:               md.DialTLS,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	start := time.Now()
	resp, err := MyTransport.RoundTrip(req)
	ttfb := time.Since(start) - (md.DialTime + md.TLSTime)
	log.Println(ttfb)
	if err != nil {
		return &CurlResult{0, nil, md.RemoteStr, err.Error(), "", "", md.DialTime, md.TLSTime, ttfb, md.DialTime.String(), md.TLSTime.String(), ttfb.String()}
	}
	//log.Println(myDialer.RemoteStr)
	//t, _ := http.DefaultTransport.(*http.Transport)
	resp.Body.Close()
	return &CurlResult{resp.StatusCode, resp.Header, md.RemoteStr, "", resp.Proto, resp.Status, md.DialTime, md.TLSTime, ttfb, md.DialTime.String(), md.TLSTime.String(), ttfb.String()}
}
