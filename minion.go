package main

//This is a worker

import (
	"flag"
	"fmt"
	"github.com/turbobytes/pulse/utils"
	"log"
)

var version string

type serverflag []string

func (i *serverflag) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *serverflag) Set(value string) error {
	//fmt.Printf("hdr %s\n", value)
	//m := *i
	*i = append(*i, value)
	return nil
}

func main() {
	var cnc string
	var servers serverflag
	servers = []string{"8.8.8.8:53", "208.67.222.222:53"}
	var caFile, certificateFile, privateKeyFile, reqFile string
	flag.StringVar(&caFile, "ca", "ca.crt", "Path to CA")
	flag.StringVar(&certificateFile, "crt", "minion.crt", "Path to Server Certificate")
	flag.StringVar(&privateKeyFile, "key", "minion.key", "Path to Private key")
	flag.StringVar(&reqFile, "req", "minion.crt.request", "Path to request file")
	flag.StringVar(&cnc, "cnc", "localhost:7777", "Location of command and control?")
	flag.Var(&servers, "servers", "DNS servers to query 8.8.8.8:53 and 208.67.222.222:53 included by default")
	flag.Parse()
	log.Println("servers", servers)
	log.Fatal(pulse.Runminion(cnc, caFile, certificateFile, privateKeyFile, reqFile, version, servers))
}
