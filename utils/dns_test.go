package pulse

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// Returns an available UDP port from kernel
func getfreeport() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.LocalAddr().(*net.UDPAddr).Port, nil
}

func TestDNSImpl(t *testing.T) {
	port, err := getfreeport()
	if err != nil {
		t.Fatal(err)
	}
	mock := fmt.Sprintf("127.0.0.1:%d", port)
	server := &dns.Server{Addr: mock, Net: "udp"}
	go server.ListenAndServe()
	//Setup handlers
	//Always responds 1.1.1.1 and only to qtype A
	dns.HandleFunc("foo.pulse.", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = false
		if len(r.Question) > 0 {
			if r.Question[0].Qtype == dns.TypeA {
				//Only include answer for type A
				aRec := &dns.A{
					Hdr: dns.RR_Header{
						Name:   r.Question[0].Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    10,
					},
					A: net.ParseIP("1.1.1.1").To4(),
				}
				m.Answer = append(m.Answer, aRec)
			}
		}
		w.WriteMsg(m)
	})

	req := &DNSRequest{
		Host:        "1.foo.pulse.",
		QType:       dns.TypeA,
		Targets:     []string{mock},
		NoRecursion: false,
	}
	//Run query
	resp := DNSImpl(req)
	//Perform checks
	if resp.Err != "" {
		t.Fatalf(resp.Err)
	}
	//There should be exactly 1 result
	if len(resp.Results) != 1 {
		t.Fatalf("There should be exactly 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Err != "" {
		t.Fatalf(resp.Results[0].Err)
	}
	if resp.Results[0].Rtt > time.Millisecond {
		t.Errorf("Took %s to RTT.", resp.Results[0].Rtt)
	}
	m1 := new(dns.Msg)
	err = m1.Unpack(resp.Results[0].Raw)
	if err != nil {
		t.Fatal(err)
	}

	//Assert things in response message
	if len(m1.Answer) != 1 {
		t.Fatalf("There should be exactly 1 Answer, got %d", len(m1.Answer))
	}
	if m1.Answer[0].Header().Rrtype != dns.TypeA {
		t.Fatalf("Expected TypeA, got %d", m1.Answer[0].Header().Rrtype)
	}
	arec, ok := m1.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("Error casting to dns.A")
	}
	if arec.Hdr.Name != "1.foo.pulse." {
		t.Errorf("Expected 1.foo.pulse., got %d", arec.Hdr.Name)
	}
	if arec.A.String() != "1.1.1.1" {
		t.Errorf("Expected 1.1.1.1, got %d", arec.A.String())
	}
}
