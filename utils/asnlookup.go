package pulse

import (
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var LookupErr = errors.New("Ipinfo error")

//Look up ASN name using Team Cymru's DNS api
func LookupASN(asn string) (name string, err error) {
	c := new(dns.Client)
	c.DialTimeout = time.Second * 2
	c.ReadTimeout = time.Second * 2
	c.WriteTimeout = time.Second * 2
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{asn + ".asn.cymru.com.", dns.TypeTXT, dns.ClassINET}
	msg, _, err := c.Exchange(m1, "8.8.8.8:53")
	if err != nil {
		return
	}
	for _, ans := range msg.Answer {
		if t, ok := ans.(*dns.TXT); ok {
			n := t.Txt[0]
			sp := strings.Split(n, "|")
			name = sp[len(sp)-1]
			name = strings.TrimSpace(name)
			return
		}
	}
	return

}

//Lookup ASN and name using ipinfo.io
func IpInfoOrg(ip string) (orginfo string, err error) {
	url := fmt.Sprintf("http://ipinfo.io/%s/org", ip)
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = LookupErr
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	orginfo = strings.TrimSpace(string(data))
	if orginfo == "" {
		err = LookupErr
	}
	return
}
