package pulse

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type CurlResult struct {
	Status    int
	Header    http.Header
	Remote    string
	Err       string
	Proto     string
	StatusStr string
}

type CurlRequest struct {
	Path     string
	Endpoint string
	Host     string
	Ssl      bool
}

type MyDialer struct {
	RemoteStr string
}

var (
	localipv4 = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "127.0.0.0/8", "100.64.0.0/10"}
	localipv6 = []string{"fd00::/8"}
)

func (md *MyDialer) Dial(network, address string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	con, err := dialer.Dial(network, address)
	if err == nil {
		md.RemoteStr = con.RemoteAddr().String()
		a, _ := con.RemoteAddr().(*net.TCPAddr)
		ipv4 := a.IP.To4()
		if ipv4 != nil {
			for _, cidr := range localipv4 {
				_, inet, _ := net.ParseCIDR(cidr)
				if inet.Contains(ipv4) {
					con.Close()
					return nil, errors.New("Security error: Not allowed to connect to local IP: " + ipv4.String())
				}
			}
		}
		ipv6 := a.IP.To16()
		if ipv6 != nil {
			for _, cidr := range localipv6 {
				_, inet, _ := net.ParseCIDR(cidr)
				if inet.Contains(ipv6) {
					con.Close()
					return nil, errors.New("Security error: Not allowed to connect to local IP: " + ipv6.String())
				}
			}
		}
	}
	return con, err
}

//example: Curl("/Forsaken.rar", "wpc.6e8d.chicdn.net", "levelup-test.turbobytes.net", false)
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
		return &CurlResult{0, nil, "", err.Error(), "", ""}
	}
	tlshost := r.Endpoint //Validate with endpoint if no host given
	if r.Host != "" {
		req.Host = r.Host
		tlshost = r.Host //Validate with Host hdr if present
	}
	myDialer := &MyDialer{}
	MyTransport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
		Dial:              myDialer.Dial,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		TLSClientConfig:       &tls.Config{ServerName: tlshost}, //Override the hostname to validate
	}
	resp, err := MyTransport.RoundTrip(req)
	if err != nil {
		return &CurlResult{0, nil, myDialer.RemoteStr, err.Error(), "", ""}
	}
	//log.Println(myDialer.RemoteStr)
	//t, _ := http.DefaultTransport.(*http.Transport)
	resp.Body.Close()
	return &CurlResult{resp.StatusCode, resp.Header, myDialer.RemoteStr, "", resp.Proto, resp.Status}
}
