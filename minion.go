package main

//This is a worker

import (
	"flag"
	"log"

	"github.com/turbobytes/pulse/utils"
)

var version string //This variable is populated during build of production binaries.

func main() {
	var cnc, caFile, certificateFile, privateKeyFile, reqFile, servers string
	flag.StringVar(&caFile, "ca", "ca.crt", "Path to CA")
	flag.StringVar(&certificateFile, "crt", "minion.crt", "Path to Server Certificate")
	flag.StringVar(&privateKeyFile, "key", "minion.key", "Path to Private key")
	flag.StringVar(&reqFile, "req", "minion.crt.request", "Path to request file")
	flag.StringVar(&cnc, "cnc", "localhost:7777", "Location of command and control?")
	flag.StringVar(&servers, "servers", "", "Legacy, this arg is ignored. It is here because old deployments might still set it")
	flag.Parse()
	log.Println("servers", servers)
	log.Fatal(pulse.Runminion(cnc, caFile, certificateFile, privateKeyFile, reqFile, version))
}
