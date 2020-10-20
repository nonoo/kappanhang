package main

import (
	"bytes"

	"github.com/akosmarton/papipes"
	"github.com/nonoo/kappanhang/log"
)

type audioStruct struct {
	source papipes.Source
	sink   papipes.Sink

	// Send to this channel to play audio.
	play chan []byte
	rec  chan []byte

	playBuf *bytes.Buffer
	canPlay chan bool
}

var audio audioStruct

func (a *audioStruct) playLoop() {
	for {
		<-a.canPlay

		d := make([]byte, a.playBuf.Len())
		bytesToWrite, err := a.playBuf.Read(d)
		if err == nil {
			for {
				written, err := a.source.Write(d)
				if err != nil {
					log.Error(err)
					break
				}
				bytesToWrite -= written
				if bytesToWrite == 0 {
					break
				}
				d = d[written:]
			}
		} else {
			log.Error(err)
		}
	}
}

func (a *audioStruct) loop() {
	go a.playLoop()

	for {
		select {
		case d := <-a.play:
			a.playBuf.Write(d)

			select {
			case a.canPlay <- true:
			default:
			}
		}
	}
}

func (a *audioStruct) init() {
	a.source.Name = "kappanhang"
	a.source.Filename = "/tmp/kappanhang.source"
	a.source.Rate = 48000
	a.source.Format = "s16le"
	a.source.Channels = 1
	a.source.SetProperty("device.buffering.buffer_size", 1920*5)
	a.source.SetProperty("device.description", "kappanhang input")

	a.sink.Name = "kappanhang"
	a.sink.Filename = "/tmp/kappanhang.sink"
	a.sink.Rate = 48000
	a.sink.Format = "s16le"
	a.sink.Channels = 1
	a.sink.SetProperty("device.buffering.buffer_size", 1920*5)
	a.sink.SetProperty("device.description", "kappanhang output")

	if err := a.source.Open(); err != nil {
		exit(err)
	}

	if err := a.sink.Open(); err != nil {
		exit(err)
	}

	a.playBuf = bytes.NewBuffer([]byte{})
	a.play = make(chan []byte)
	a.canPlay = make(chan bool)
	a.rec = make(chan []byte)
	go a.loop()
}

func (a *audioStruct) deinit() {
	if a.source.IsOpen() {
		if err := a.source.Close(); err != nil {
			log.Error(err)
		}
	}

	if a.sink.IsOpen() {
		if err := a.sink.Close(); err != nil {
			log.Error(err)
		}
	}
}
