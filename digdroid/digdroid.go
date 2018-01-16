package digdroid

//Package digdroid is Android and (maybe) iOS binding for running DNS test using pulse.
//We don;t use pulse directly cause bind doesnt work with types such as interface{} and uint16
//gomobile bind github.com/turbobytes/pulse/digdroid

import (
	"bytes"
	"fmt"
	"log"
	"text/tabwriter"

	"github.com/miekg/dns"
	"github.com/turbobytes/pulse/utils"
)

type DNSResult struct {
	Err    string
	Output string
	Rtt    string
}

//We use RunDNS as proxy instead of using pulse directly because gomobile bind
//can't make bindings for types that use interface{} or uint16
func RunDNS(host, target, qtypestr string, norec bool) *DNSResult {
	var qtype uint16
	switch qtypestr {
	case "A":
		qtype = 1
	case "AAAA":
		qtype = 28
	case "NS":
		qtype = 2
	case "CNAME":
		qtype = 5
	case "MX":
		qtype = 15
	case "PTR":
		qtype = 12
	case "SOA":
		qtype = 6
	case "SRV":
		qtype = 33
	case "TXT":
		qtype = 16
	case "ANY":
		qtype = 255
	}
	req := &pulse.DNSRequest{
		Host:        host,
		QType:       qtype,
		Targets:     []string{target},
		NoRecursion: norec,
	}
	result := pulse.DNSImpl(req)
	res := &DNSResult{}
	if result.Err != "" {
		res.Err = result.Err
	} else if len(result.Results) < 1 {
		res.Err = "No results returned"
	} else if result.Results[0].Err != "" {
		res.Err = result.Results[0].Err
		res.Rtt = result.Results[0].RttStr
	} else {
		msg := &dns.Msg{}
		msg.Unpack(result.Results[0].Raw)
		log.Println(res.Output)
		res.Rtt = result.Results[0].RttStr
		//Using tabwriter.Writer to make formated output with spaces instead of tabs.
		//Because TextView in java does not know how to format tabs into columns.
		w := new(tabwriter.Writer)
		var b bytes.Buffer
		w.Init(&b, 0, 8, 2, ' ', 0)
		fmt.Fprintln(w, msg.String())
		w.Flush()
		res.Output = string(b.Bytes())

	}
	return res
}
