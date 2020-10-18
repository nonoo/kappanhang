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
	log.Print("disconnecting")
	streams.control.sendDisconnect()
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
		exit(nil)
	}()
}

func main() {
	log.Init()
	log.Print("kappanhang by Norbert Varga HA2NON https://github.com/nonoo/kappanhang")
	parseArgs()
	setupCloseHandler()

	streams.control.start()
}
