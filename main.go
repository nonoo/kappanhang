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

	if err := c.init(); err != nil {
		log.Error(err)
		c.deinit()
		return true, 1
	}
	if err := c.start(); err != nil {
		log.Error(err)
		c.deinit()
		return
	}

	select {
	case <-gotErrChan:
		c.deinit()

		// Need to wait before reinit because the IC-705 will disconnect our audio stream eventually if we relogin
		// in a too short interval without a deauth...
		t := time.NewTicker(time.Second)
		for sec := 65; sec > 0; sec-- {
			log.Print("waiting ", sec, " seconds...")
			select {
			case <-t.C:
			case <-osSignal:
				return true, 0
			}
		}
		return
	case <-osSignal:
		log.Print("sigterm received")
		c.deinit()
		return true, 0
	}
}

func reportError(err error) {
	if !strings.Contains(err.Error(), "use of closed network connection") {
		log.ErrorC(log.GetCallerFileName(true), ": ", err)
	}

	// Non-blocking notify.
	select {
	case gotErrChan <- false:
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
		}
	}

	log.Print("exiting")
	os.Exit(exitCode)
}
