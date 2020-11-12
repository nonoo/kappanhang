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
var civAddress byte
var serialTCPPort uint16
var enableSerialDevice bool
var rigctldPort uint16
var runCmd string
var runCmdOnSerialPortCreated string
var statusLogInterval time.Duration

func parseArgs() {
	h := getopt.BoolLong("help", 'h', "display help")
	v := getopt.BoolLong("verbose", 'v', "Enable verbose (debug) logging")
	a := getopt.StringLong("address", 'a', "IC-705", "Connect to address")
	u := getopt.StringLong("username", 'u', "beer", "Username")
	p := getopt.StringLong("password", 'p', "beerbeer", "Password")
	c := getopt.UintLong("civ-address", 'c', 0xa4, "CI-V address")
	t := getopt.Uint16Long("serial-tcp-port", 't', 4531, "Expose radio's serial port on this TCP port")
	s := getopt.BoolLong("enable-serial-device", 's', "Expose radio's serial port as a virtual serial port")
	r := getopt.Uint16Long("rigctld-port", 'r', 4532, "Use this TCP port for the internal rigctld")
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
	civAddress = byte(*c)
	serialTCPPort = *t
	enableSerialDevice = *s
	rigctldPort = *r
	runCmd = *e
	runCmdOnSerialPortCreated = *o
	statusLogInterval = time.Duration(*i) * time.Millisecond
}
