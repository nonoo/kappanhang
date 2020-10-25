package main

import (
	"os"

	"github.com/pborman/getopt"
)

var connectAddress string
var serialTCPPort uint16

func parseArgs() {
	h := getopt.BoolLong("help", 'h', "display help")
	a := getopt.StringLong("address", 'a', "IC-705", "Connect to address")
	t := getopt.Uint16Long("serial-tcp-port", 'p', 4532, "Expose serial port as TCP port for rigctl")

	getopt.Parse()

	if *h || *a == "" {
		getopt.Usage()
		os.Exit(1)
	}

	connectAddress = *a
	serialTCPPort = *t
}
