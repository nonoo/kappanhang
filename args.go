package main

import (
	"os"

	"github.com/pborman/getopt"
)

var connectAddress string
var connectPort uint16

func parseArgs() {
	h := getopt.BoolLong("help", 'h', "display help")
	a := getopt.StringLong("address", 'a', "IC-705", "Connect to address")
	p := getopt.Uint16Long("port", 'p', 50001, "Connect to UDP port")

	getopt.Parse()

	if *h || *a == "" {
		getopt.Usage()
		os.Exit(1)
	}

	connectAddress = *a
	connectPort = *p
}
