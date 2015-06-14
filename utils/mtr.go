package pulse

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
)

type MtrResult struct {
	Result string
	Err    string
}

type MtrRequest struct {
	Target string
}

func MtrImpl(r *MtrRequest) *MtrResult {
	//Validate r.Target before sending
	tgt := strings.Trim(r.Target, "\n \r") //Trim whitespace
	if strings.Contains(tgt, " ") {        //Ensure it doesnt contain space
		return &MtrResult{"", "Invalid hostname"}
	}
	if strings.HasPrefix(tgt, "-") { //Ensure it doesnt start with -
		return &MtrResult{"", "Invalid hostname"}
	}
	cmd := exec.Command("mtr", "--report-wide", "--report", tgt)
	var out bytes.Buffer
	cmd.Stdout = &out
	log.Println(cmd)
	err := cmd.Run()
	if err != nil {
		return &MtrResult{"", err.Error()}
	}
	return &MtrResult{out.String(), ""}
}
