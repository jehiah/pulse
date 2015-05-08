package pulse

import (
	"github.com/miekg/dns"
	"log"
	"strings"
	"time"
)

type IndividualDNSResult struct {
	Server   string
	Err      string
	RttStr   string
	Rtt      time.Duration
	Raw      []byte
	Formated string
	Msg      *dns.Msg
	ASN      *string
	ASName   *string
}

type DNSResult struct {
	Results []IndividualDNSResult
	Err     string
}

type DNSRequest struct {
	Host    string
	QType   uint16
	Targets []string
}

func rundnsquery(host, server string, ch chan IndividualDNSResult, qclass uint16, retry bool) {
	res := IndividualDNSResult{}
	res.Server = strings.Split(server, ":")[0]
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{host, qclass, dns.ClassINET}
	c := new(dns.Client)
	c.DialTimeout = time.Millisecond * 500
	c.ReadTimeout = time.Millisecond * 500
	c.WriteTimeout = time.Millisecond * 500
	log.Println("Asking", server, "for", host)
	msg, rtt, err := c.Exchange(m1, server)
	res.RttStr = rtt.String()
	res.Rtt = rtt
	if err != nil {
		res.Err = err.Error()
		if retry {
			//If fail at first... try again .. once...
			//I could tell a UDP joke... but you might not get it...
			rundnsquery(host, server, ch, qclass, false)
		} else {
			ch <- res
		}
	} else {
		//res.Result = msg.String()
		res.Raw, _ = msg.Pack()
		//res.Formated = msg.String()
		ch <- res
	}
}

func DNSImpl(r *DNSRequest) *DNSResult {
	//TODO: validate r.Target before sending
	res := new(DNSResult)
	n := len(r.Targets)
	res.Results = make([]IndividualDNSResult, n)
	ch := make(chan IndividualDNSResult, n)
	for _, server := range r.Targets {
		go rundnsquery(r.Host, server, ch, r.QType, true)
		time.Sleep(time.Millisecond * 5) //Pace out the packets a bit
	}
	for i := 0; i < n; i++ {
		item := <-ch
		res.Results[i] = item
		//res := runquery(*host, server)
	}

	return res
}
