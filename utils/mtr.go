package pulse

import (
	"bytes"
	"log"
	"os/exec"
)

type MtrResult struct {
	Result string
	Err    string
}

type MtrRequest struct {
	Target string
}

func MtrImpl(r *MtrRequest) *MtrResult {
	//TODO: validate r.Target before sending
	cmd := exec.Command("mtr", "--report-wide", "--report", r.Target)
	var out bytes.Buffer
	cmd.Stdout = &out
	log.Println(cmd)
	err := cmd.Run()
	if err != nil {
		return &MtrResult{"", err.Error()}
	}
	return &MtrResult{out.String(), ""}
}
