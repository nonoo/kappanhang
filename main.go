package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nonoo/kappanhang/log"
)

var gotErrChan = make(chan bool)

func runControlStream(osSignal chan os.Signal) (shouldExit bool, exitCode int) {
	// Depleting gotErrChan.
	var finished bool
	for !finished {
		select {
		case <-gotErrChan:
		default:
			finished = true
		}
	}

	c := controlStream{}
	defer c.deinit()

	if err := c.init(); err != nil {
		log.Error(err)
		return true, 1
	}
	if err := c.start(); err != nil {
		log.Error(err)
		return
	}

	select {
	case <-gotErrChan:
		return
	case <-osSignal:
		log.Print("sigterm received")
		return true, 0
	}
}

func reportError(err error) {
	if !strings.Contains(err.Error(), "use of closed network connection") {
		log.ErrorC(log.GetCallerFileName(true), ": ", err)
	}

	// Non-blocking notify.
	select {
	case gotErrChan <- true:
	default:
	}
}

func main() {
	log.Init()
	log.Print("kappanhang by Norbert Varga HA2NON and Akos Marton ES1AKOS https://github.com/nonoo/kappanhang")
	parseArgs()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, syscall.SIGTERM)

	var shouldExit bool
	var exitCode int
	for !shouldExit {
		shouldExit, exitCode = runControlStream(osSignal)
		if !shouldExit {
			log.Print("restarting control stream...")
			select {
			case <-time.NewTimer(3 * time.Second).C:
			case <-osSignal:
				shouldExit = true
			}
		}
	}

	log.Print("exiting")
	os.Exit(exitCode)
}
