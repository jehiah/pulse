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
	IPv    string //blank for auto, 4 for IPv4, 6 for IPv6
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
	var cmd *exec.Cmd
	switch r.IPv {
	case "4":
		cmd = exec.Command("mtr", "--report-wide", "--report", "-4", tgt)
	case "6":
		cmd = exec.Command("mtr", "--report-wide", "--report", "-6", tgt)
	default:
		cmd = exec.Command("mtr", "--report-wide", "--report", tgt)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	log.Println(cmd)
	err := cmd.Run()
	if err != nil {
		return &MtrResult{"", err.Error()}
	}
	return &MtrResult{out.String(), ""}
}
