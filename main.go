package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/akosmarton/papipes"
	"github.com/nonoo/kappanhang/log"
)

var streams struct {
	control controlStream
	audio   audioStream
}

var audioPipes struct {
	source papipes.Source
	sink   papipes.Sink
}

func exit(err error) {
	if err != nil {
		log.Error(err.Error())
	}

	streams.audio.sendDisconnect()
	streams.control.sendDisconnect()

	if audioPipes.source.IsOpen() {
		if err := audioPipes.source.Close(); err != nil {
			log.Error(err)
		}
	}

	if audioPipes.sink.IsOpen() {
		if err := audioPipes.sink.Close(); err != nil {
			log.Error(err)
		}
	}

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

	audioPipes.source.Name = "kappanhang"
	audioPipes.source.Filename = "/tmp/kappanhang.source"
	audioPipes.source.Rate = 48000
	audioPipes.source.Format = "s16le"
	audioPipes.source.Channels = 1
	audioPipes.source.SetProperty("device.buffering.buffer_size", 1920*5)
	audioPipes.source.SetProperty("device.description", "kappanhang input")

	audioPipes.sink.Name = "kappanhang"
	audioPipes.sink.Filename = "/tmp/kappanhang.sink"
	audioPipes.sink.Rate = 48000
	audioPipes.sink.Format = "s16le"
	audioPipes.sink.Channels = 1
	audioPipes.sink.SetProperty("device.buffering.buffer_size", 1920*5)
	audioPipes.sink.SetProperty("device.description", "kappanhang output")

	if err := audioPipes.source.Open(); err != nil {
		exit(err)
	}

	if err := audioPipes.sink.Open(); err != nil {
		exit(err)
	}

	setupCloseHandler()

	streams.audio.init()
	streams.control.init()

	streams.control.start()
}
