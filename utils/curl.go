package pulse

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

var (
	tlshandshaketimeout = time.Second * 15
	dialtimeout         = time.Second * 15
	responsetimeout     = time.Second * 30
	keepalive           = time.Second * 30
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

func upgradetls(con net.Conn, tlshost string, result *CurlResult) (net.Conn, error) {
	tlstimer := time.Now()
	tlsconf := &tls.Config{
		MinVersion: tls.VersionTLS10, //TLS 1.0 minimum. Depricating SSLv3 RFC 7568
		ServerName: tlshost,          //Override the hostname to validate
	}

	tcon := tls.Client(con, tlsconf)
	errc := make(chan error, 2)
	var timer *time.Timer
	timer = time.AfterFunc(tlshandshaketimeout, func() {
		errc <- tlsHandshakeTimeoutError
	})
	go func() {
		err := tcon.Handshake()
		if timer != nil {
			timer.Stop()
		}
		errc <- err
	}()
	err := <-errc
	if err != nil {
		con.Close()
		//return nil, err
	}
	result.TLSTime = time.Since(tlstimer)
	result.TLSTimeStr = result.TLSTime.String()
	return tcon, err
}

func dial(endpoint, tlshost string, ssl bool, result *CurlResult) (net.Conn, error) {
	//If endpoint does not contain a port, add it here
	if !strings.Contains(endpoint, ":") {
		if ssl {
			endpoint = endpoint + ":443"
		} else {
			endpoint = endpoint + ":80"
		}
	}
	timer := time.Now()
	dialer := &net.Dialer{
		Timeout:   dialtimeout,
		KeepAlive: keepalive,
		DualStack: true,
	}
	//TODO: Go even lower level cause we need DNS time
	con, err := dialer.Dial("tcp", endpoint)
	if err == nil {
		result.Remote = con.RemoteAddr().String()
		a, _ := con.RemoteAddr().(*net.TCPAddr)

		if islocalip(a.IP) {
			fmt.Println(a.IP)
			con.Close()
			return nil, securityerr
		}

	}
	result.DialTime = time.Since(timer)
	result.DialTimeStr = result.DialTime.String()

	if ssl {
		return upgradetls(con, tlshost, result)
	}

	return con, err
}

type response struct {
	statusline string
	header     http.Header
	err        error
}

func readresp(rawconn net.Conn, resc chan response) {
	rd := bufio.NewReader(rawconn)
	//Read first line which contains status
	statusline, _ := rd.ReadString('\n')
	tp := textproto.NewReader(rd)
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		resc <- response{statusline, nil, err}
		return
	}
	httpheader := http.Header(mimeHeader)
	resc <- response{statusline, httpheader, err}
}

func parseresponse(rawconn net.Conn, result *CurlResult) error {
	respchan := make(chan response, 0)
	go readresp(rawconn, respchan)
	var resp1 response
	select {
	case resp1 = <-respchan:
		break
	case <-time.After(responsetimeout):
		rawconn.Close()
		return errors.New("Request timed out")
	}

	if resp1.err != nil {
		return resp1.err
	}

	//Extract Proto, Status and StatusStr
	splitted := strings.SplitN(resp1.statusline, " ", 2)
	log.Println("splitted:", splitted)
	if len(splitted) < 2 {
		return errors.New("Error reading response")
	}
	result.Proto = splitted[0]
	log.Println("Proto:", result.Proto)
	result.StatusStr = strings.Trim(splitted[1], "\r\n")
	log.Println("StatusStr:", result.StatusStr)
	splitted = strings.Split(result.StatusStr, " ")
	i, err := strconv.Atoi(splitted[0])
	if err != nil {
		return errors.New("Error reading response")
	}
	result.Status = i
	log.Println("Status:", result.Status)

	result.Header = resp1.header
	log.Println("Header:", result.Header)
	return nil
}

func CurlImpl(r *CurlRequest) *CurlResult {
	result := &CurlResult{0, nil, "", "", "", "", time.Duration(0), time.Duration(0), time.Duration(0), "", "", ""}
	var url string
	if r.Ssl {
		url = fmt.Sprintf("https://%s%s", r.Endpoint, r.Path)
	} else {
		url = fmt.Sprintf("http://%s%s", r.Endpoint, r.Path)
	}
	log.Println(url)
	//Create a request object
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.Err = err.Error()
		return result
	}

	//Override Host header if needed
	tlshost := r.Endpoint //Validate with endpoint if no host given
	if r.Host != "" {
		req.Host = r.Host
		tlshost = r.Host //Validate with Host hdr if present
	}

	//Get Raw payload which we will eventually write on the wire
	payload, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		result.Err = err.Error()
		return result
	}
	//Make a raw connection
	rawconn, err := dial(r.Endpoint, tlshost, r.Ssl, result)
	if err != nil {
		result.Err = err.Error()
		return result
	}
	defer rawconn.Close()
	//Start ttfb timer
	ttfbtimer := time.Now()

	//Write the GET request
	_, err = rawconn.Write(payload)
	if err != nil {
		result.Err = err.Error()
		return result
	}

	//read and parse the response
	err = parseresponse(rawconn, result)

	if err != nil {
		result.Err = err.Error()
	}

	result.Ttfb = time.Since(ttfbtimer)
	result.TtfbStr = result.Ttfb.String()
	return result

}
