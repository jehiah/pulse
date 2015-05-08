package pulse

import (
	"fmt"
	"math/big"
	"time"
)

type Resolver struct {
	Servers []string
	Version string
}

const (
	TypeDNS  = 1
	TypeMTR  = 2
	TypeCurl = 3
)

type CombinedRequest struct {
	Type        int
	Args        interface{}
	RequestedAt time.Time
}

type CombinedResult struct {
	Type         int
	Result       interface{}
	CompletedAt  time.Time
	TimeTaken    time.Duration
	TimeTakenStr string
	Err          string
	Version      string
	Name         string
	Agent        string
	ASN          *string
	ASName       *string
	Country      string
	State        string
	City         string
	Id           *big.Int
}

func (r *Resolver) Combined(req *CombinedRequest, out *CombinedResult) error {
	st := time.Now()
	tmp := new(CombinedResult)
	tmp.Type = req.Type
	switch req.Type {
	case TypeDNS:
		//TODO Run dns and populate result
		args, ok := req.Args.(DNSRequest)
		if !ok {
			tmp.Err = "Error parsing request"
		} else {
			tmp.Result = DNSImpl(&args)
		}
	case TypeMTR:
		//Run MTR and populate result
		args, ok := req.Args.(MtrRequest)
		if !ok {
			tmp.Err = "Error parsing request"
		} else {
			tmp.Result = MtrImpl(&args)
		}
	case TypeCurl:
		//Run curl and populate result
		args, ok := req.Args.(CurlRequest)
		if !ok {
			tmp.Err = "Error parsing request"
		} else {
			tmp.Result = CurlImpl(&args)
		}
	default:
		//ERR
		tmp.Err = fmt.Sprintf("Unknown test type : %d", req.Type)
	}
	tmp.CompletedAt = time.Now()
	tmp.Version = r.Version
	tmp.TimeTaken = time.Since(st)
	tmp.TimeTakenStr = tmp.TimeTaken.String()
	*out = *tmp
	return nil
}
