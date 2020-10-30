package main

import (
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

var gotErrChan = make(chan bool)

func getAboutStr() string {
	var v string
	bi, ok := debug.ReadBuildInfo()
	if ok {
		v = bi.Main.Version
	} else {
		v = "(devel)"
	}
	return "kappanhang " + v + " by Norbert Varga HA2NON and Akos Marton ES1AKOS https://github.com/nonoo/kappanhang"
}

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

	ctrl := &controlStream{}

	if err := ctrl.init(); err != nil {
		log.Error(err)
		ctrl.deinit()
		t := time.NewTimer(time.Second)
		select {
		case <-t.C:
			return false, 0
		case <-osSignal:
			return true, 0
		}
	}

	select {
	case requireWait := <-gotErrChan:
		ctrl.deinit()

		if !requireWait {
			return
		}

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
		ctrl.deinit()
		return true, 0
	}
}

func reportError(err error) {
	if !strings.Contains(err.Error(), "use of closed network connection") {
		log.ErrorC(log.GetCallerFileName(true), ": ", err)
	}

	requireWait := true
	if strings.Contains(err.Error(), "got radio disconnected") {
		requireWait = false
	}

	// Non-blocking notify.
	select {
	case gotErrChan <- requireWait:
	default:
	}
}

func main() {
	parseArgs()
	log.Init()
	log.Print(getAboutStr())

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, syscall.SIGTERM)

	if statusLog.isRealtime() {
		keyboard.init()
	}

	var shouldExit bool
	var exitCode int
	for !shouldExit {
		shouldExit, exitCode = runControlStream(osSignal)

		select {
		case <-osSignal:
			log.Print("sigterm received")
			shouldExit = true
		default:
		}

		if !shouldExit {
			log.Print("restarting control stream...")
		}
	}

	stopCmd()
	audio.deinit()
	serialTCPSrv.deinit()
	serialPort.deinit()

	if statusLog.isRealtime() {
		keyboard.deinit()
	}

	log.Print("exiting")
	os.Exit(exitCode)
}
