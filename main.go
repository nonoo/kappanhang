package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nonoo/kappanhang/log"
)

var portControl PortControl
var portAudio PortAudio

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Print("disconnecting")
		portControl.SendDisconnect()
		os.Exit(0)
	}()
}

func main() {
	log.Init()
	parseArgs()
	setupCloseHandler()

	portControl.StartStream()
}
