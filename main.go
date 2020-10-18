package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nonoo/kappanhang/log"
)

var streams struct {
	control controlStream
	audio   audioStream
}

func exit(err error) {
	streams.control.SendDisconnect()
	if err == nil {
		os.Exit(0)
	} else {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Print("disconnecting")
		exit(nil)
	}()
}

func main() {
	log.Init()
	parseArgs()
	setupCloseHandler()

	streams.control.Start()
}
