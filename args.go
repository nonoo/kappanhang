package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/getopt"
)

var verboseLog bool
var connectAddress string
var serialTCPPort uint16
var enableSerialDevice bool
var statusLogInterval time.Duration

func parseArgs() {
	h := getopt.BoolLong("help", 'h', "display help")
	v := getopt.BoolLong("verbose", 'v', "Enable verbose (debug) logging")
	a := getopt.StringLong("address", 'a', "IC-705", "Connect to address")
	t := getopt.Uint16Long("serial-tcp-port", 'p', 4533, "Expose radio's serial port on this TCP port")
	s := getopt.BoolLong("enable-serial-device", 's', "Expose radio's serial port as a virtual serial port")
	i := getopt.Uint16Long("log-interval", 'i', 100, "Status log interval in milliseconds")

	getopt.Parse()

	if *h || *a == "" {
		fmt.Println(getAboutStr())
		getopt.Usage()
		os.Exit(1)
	}

	verboseLog = *v
	connectAddress = *a
	serialTCPPort = *t
	enableSerialDevice = *s
	statusLogInterval = time.Duration(*i) * time.Millisecond
}
