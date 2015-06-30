package pulse

import (
	"github.com/sajal/mtrparser"
	"strings"
)

type MtrResult struct {
	Result *mtrparser.MTROutPut
	Err    string
}

type MtrRequest struct {
	Target string
	IPv    string //blank for auto, 4 for IPv4, 6 for IPv6
}

func MtrImpl(r *MtrRequest) *MtrResult {
	//Validate r.Target before sending
	tgt := strings.Trim(r.Target, "\n \r") //Trim whitespace
	if strings.Contains(tgt, " ") {        //Ensure it doesnt contain space
		return &MtrResult{nil, "Invalid hostname"}
	}
	if strings.HasPrefix(tgt, "-") { //Ensure it doesnt start with -
		return &MtrResult{nil, "Invalid hostname"}
	}
	out, err := mtrparser.ExecuteMTR(tgt, r.IPv)
	if err != nil {
		return &MtrResult{nil, err.Error()}
	}
	return &MtrResult{out, ""}
}
