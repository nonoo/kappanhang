package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nonoo/kappanhang/log"
)

var streams struct {
	control controlStream
	serial  serialStream
	audio   audioStream
}

func exit(err error) {
	if err != nil {
		log.Error(err.Error())
	}

	streams.audio.common.close()
	streams.serial.common.close()
	streams.control.common.close()
	serialPort.deinit()
	audio.deinit()

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
	log.Print("kappanhang by Norbert Varga HA2NON and Akos Marton ES1AKOS https://github.com/nonoo/kappanhang")
	parseArgs()

	serialPort.init()
	streams.audio.init()
	streams.serial.init()
	streams.control.init()

	setupCloseHandler()

	streams.control.start()
}
