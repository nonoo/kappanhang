package main

import (
	"os"

	"github.com/pborman/getopt"
)

var connectAddress string

func parseArgs() {
	h := getopt.BoolLong("help", 'h', "display help")
	a := getopt.StringLong("address", 'a', "IC-705", "Connect to address")

	getopt.Parse()

	if *h || *a == "" {
		getopt.Usage()
		os.Exit(1)
	}

	connectAddress = *a
}
