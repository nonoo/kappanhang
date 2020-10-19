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
	if err != nil {
		log.Error(err.Error())
	}

	streams.audio.sendDisconnect()
	streams.control.sendDisconnect()

	log.Print("exiting")
	if err == nil {
		os.Exit(0)
	} else {
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

	streams.audio.init()
	streams.control.init()

	streams.control.start()
}
