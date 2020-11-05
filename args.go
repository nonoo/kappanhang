package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/getopt"
)

var verboseLog bool
var connectAddress string
var username string
var password string
var serialTCPPort uint16
var enableSerialDevice bool
var rigctldModel uint
var disableRigctld bool
var runCmd string
var runCmdOnSerialPortCreated string
var statusLogInterval time.Duration

func parseArgs() {
	h := getopt.BoolLong("help", 'h', "display help")
	v := getopt.BoolLong("verbose", 'v', "Enable verbose (debug) logging")
	a := getopt.StringLong("address", 'a', "IC-705", "Connect to address")
	u := getopt.StringLong("username", 'u', "beer", "Username")
	p := getopt.StringLong("password", 'p', "beerbeer", "Password")
	t := getopt.Uint16Long("serial-tcp-port", 't', 4533, "Expose radio's serial port on this TCP port")
	s := getopt.BoolLong("enable-serial-device", 's', "Expose radio's serial port as a virtual serial port")
	m := getopt.UintLong("rigctld-model", 'm', 3085, "rigctld model number")
	r := getopt.BoolLong("disable-rigctld", 'r', "Disable starting rigctld")
	e := getopt.StringLong("exec", 'e', "", "Exec cmd when connected")
	o := getopt.StringLong("exec-serial", 'o', "socat /tmp/kappanhang-IC-705.pty /tmp/vmware.pty", "Exec cmd when virtual serial port is created, set to - to disable")
	i := getopt.Uint16Long("log-interval", 'i', 100, "Status bar/log interval in milliseconds")

	getopt.Parse()

	if *h || *a == "" {
		fmt.Println(getAboutStr())
		getopt.Usage()
		os.Exit(1)
	}

	verboseLog = *v
	connectAddress = *a
	username = *u
	password = *p
	serialTCPPort = *t
	enableSerialDevice = *s
	rigctldModel = *m
	disableRigctld = *r
	runCmd = *e
	runCmdOnSerialPortCreated = *o
	statusLogInterval = time.Duration(*i) * time.Millisecond
}
