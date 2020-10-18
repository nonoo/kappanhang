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

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Print("disconnecting")
		streams.control.SendDisconnect()
		os.Exit(0)
	}()
}

func main() {
	log.Init()
	parseArgs()
	setupCloseHandler()

	streams.control.Start()
}
