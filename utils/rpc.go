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
	AgentFilter []*big.Int
}

//Clone a CombinedRequest.. sort of deepcopy
func (original *CombinedRequest) Clone() *CombinedRequest {
	return &CombinedRequest{
		Type:        original.Type,
		Args:        original.Args,
		RequestedAt: original.RequestedAt,
		AgentFilter: original.AgentFilter,
	}
}

type CombinedResult struct {
	Type         int           //Test type. 1=dns, 2=mtr, 3=curl
	Result       interface{}   //DNSResult for dns, CurlResult for curl and MtrResult for mtr
	CompletedAt  time.Time     //Time the test was completed
	TimeTaken    time.Duration //Time taken to run the test
	TimeTakenStr string        //Time taken to run the test in humanized form
	Err          string        //Any error, typically at RPC level
	Version      string        //The version of the minion that ran this test
	Name         string        //The name assigned to this agent.
	Agent        string        // /24 IP of the agent.
	ASN          *string       //ASN of the agent.
	ASName       *string       //ASN description of the agent.
	Country      string        //Agent's country
	State        string        //Agent's state
	City         string        //Agent's city
	Id           *big.Int      //Agent's ID - this is unique per agent
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
