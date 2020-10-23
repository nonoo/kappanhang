package main

import (
	"bytes"
	"errors"
	"os"
	"sync"

	"github.com/akosmarton/papipes"
	"github.com/nonoo/kappanhang/log"
)

type audioStruct struct {
	source papipes.Source
	sink   papipes.Sink

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool

	// Send to this channel to play audio.
	play chan []byte
	// Read from this channel for audio.
	rec chan []byte

	mutex   sync.Mutex
	playBuf *bytes.Buffer
	canPlay chan bool
}

func (a *audioStruct) playLoop(deinitNeededChan, deinitFinishedChan chan bool) {
	for {
		select {
		case <-a.canPlay:
		case <-deinitNeededChan:
			deinitFinishedChan <- true
			return
		}

		for {
			a.mutex.Lock()
			if a.playBuf.Len() < 1920 {
				a.mutex.Unlock()
				break
			}

			d := make([]byte, 1920)
			bytesToWrite, err := a.playBuf.Read(d)
			a.mutex.Unlock()
			if err != nil {
				log.Error(err)
				break
			}
			if bytesToWrite != len(d) {
				log.Error("buffer underread")
				break
			}

			for {
				written, err := a.source.Write(d)
				if err != nil {
					if _, ok := err.(*os.PathError); !ok {
						reportError(err)
					}
				}
				bytesToWrite -= written
				if bytesToWrite == 0 {
					break
				}
				d = d[written:]
			}
		}
	}
}

func (a *audioStruct) recLoop(deinitNeededChan, deinitFinishedChan chan bool) {
	defer func() {
		deinitFinishedChan <- true
	}()

	frameBuf := make([]byte, 1920)
	buf := bytes.NewBuffer([]byte{})

	for {
		select {
		case <-deinitNeededChan:
			return
		default:
		}

		n, err := a.sink.Read(frameBuf)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				reportError(err)
			}
		}

		buf.Write(frameBuf[:n])

		for buf.Len() >= len(frameBuf) {
			n, err = buf.Read(frameBuf)
			if err != nil {
				reportError(err)
			}
			if n != len(frameBuf) {
				reportError(errors.New("audio buffer read error"))
			}

			select {
			case a.rec <- frameBuf:
			case <-deinitNeededChan:
				return
			}
		}
	}
}

func (a *audioStruct) loop() {
	playLoopDeinitNeededChan := make(chan bool)
	playLoopDeinitFinishedChan := make(chan bool)
	go a.playLoop(playLoopDeinitNeededChan, playLoopDeinitFinishedChan)
	recLoopDeinitNeededChan := make(chan bool)
	recLoopDeinitFinishedChan := make(chan bool)
	go a.recLoop(recLoopDeinitNeededChan, recLoopDeinitFinishedChan)

	var d []byte
	for {
		select {
		case d = <-a.play:
		case <-a.deinitNeededChan:
			recLoopDeinitNeededChan <- true
			<-recLoopDeinitFinishedChan
			playLoopDeinitNeededChan <- true
			<-playLoopDeinitFinishedChan

			a.deinitFinishedChan <- true
			return
		}

		a.mutex.Lock()
		a.playBuf.Write(d)
		a.mutex.Unlock()

		// Non-blocking notify.
		select {
		case a.canPlay <- true:
		default:
		}
	}
}

func (a *audioStruct) init(devName string) error {
	a.source.Name = "kappanhang-" + devName
	a.source.Filename = "/tmp/kappanhang-" + devName + ".source"
	a.source.Rate = 48000
	a.source.Format = "s16le"
	a.source.Channels = 1
	a.source.SetProperty("device.buffering.buffer_size", (48000*16)/10) // 100 ms
	a.source.SetProperty("device.description", "kappanhang: "+devName)

	a.sink.Name = "kappanhang-" + devName
	a.sink.Filename = "/tmp/kappanhang-" + devName + ".sink"
	a.sink.Rate = 48000
	a.sink.Format = "s16le"
	a.sink.Channels = 1
	a.sink.SetProperty("device.buffering.buffer_size", (48000*16)/10)
	a.sink.SetProperty("device.description", "kappanhang: "+devName)

	if err := a.source.Open(); err != nil {
		return err
	}

	if err := a.sink.Open(); err != nil {
		return err
	}

	a.playBuf = bytes.NewBuffer([]byte{})
	a.play = make(chan []byte)
	a.canPlay = make(chan bool)
	a.rec = make(chan []byte)
	a.deinitNeededChan = make(chan bool)
	a.deinitFinishedChan = make(chan bool)
	go a.loop()
	return nil
}

func (a *audioStruct) deinit() {
	if a.source.IsOpen() {
		if err := a.source.Close(); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				log.Error(err)
			}
		}
	}

	if a.sink.IsOpen() {
		if err := a.sink.Close(); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				log.Error(err)
			}
		}
	}

	if a.deinitNeededChan != nil {
		a.deinitNeededChan <- true
		<-a.deinitFinishedChan
	}
}
