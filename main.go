package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nonoo/kappanhang/log"
)

var ports struct {
	control portControl
	audio   portAudio
}

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Print("disconnecting")
		ports.control.SendDisconnect()
		os.Exit(0)
	}()
}

func main() {
	log.Init()
	parseArgs()
	setupCloseHandler()

	ports.control.StartStream()
}
