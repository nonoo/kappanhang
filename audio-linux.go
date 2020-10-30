// +build linux

package main

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/akosmarton/papipes"
	"github.com/mesilliac/pulse-simple"
)

const audioSampleRate = 48000
const audioSampleBytes = 2
const pulseAudioBufferLength = 100 * time.Millisecond
const audioFrameSize = 1920 // 20ms
const maxPlayBufferSize = audioFrameSize * 5

type audioStruct struct {
	devName string

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

	togglePlaybackToDefaultSoundcardChan chan bool
	defaultSoundCardStream               *pulse.Stream
}

var audio audioStruct

func (a *audioStruct) defaultSoundCardStreamDeinit() {
	_ = a.defaultSoundCardStream.Drain()
	a.defaultSoundCardStream.Free()
	a.defaultSoundCardStream = nil
}

func (a *audioStruct) togglePlaybackToDefaultSoundcard() {
	if a.defaultSoundCardStream == nil {
		log.Print("turned on audio playback")
		ss := pulse.SampleSpec{Format: pulse.SAMPLE_S16LE, Rate: 48000, Channels: 1}
		a.defaultSoundCardStream, _ = pulse.Playback("kappanhang", a.devName, &ss)
	} else {
		a.defaultSoundCardStreamDeinit()
		log.Print("turned off audio playback")
	}
}

func (a *audioStruct) playToVirtualSoundcard(d []byte) {
	for len(d) > 0 {
		written, err := a.source.Write(d)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				reportError(err)
			}
			break
		}
		d = d[written:]
	}
}

func (a *audioStruct) playToDefaultSoundcard(d []byte) {
	for len(d) > 0 {
		written, err := a.defaultSoundCardStream.Write(d)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				reportError(err)
			}
			break
		}
		d = d[written:]
	}
}

func (a *audioStruct) playLoop(deinitNeededChan, deinitFinishedChan chan bool) {
	for {
		select {
		case <-a.canPlay:
		case <-a.togglePlaybackToDefaultSoundcardChan:
			a.togglePlaybackToDefaultSoundcard()
		case <-deinitNeededChan:
			if a.defaultSoundCardStream != nil {
				a.defaultSoundCardStreamDeinit()
			}
			deinitFinishedChan <- true
			return
		}

		for {
			a.mutex.Lock()
			if a.playBuf.Len() < audioFrameSize {
				a.mutex.Unlock()
				break
			}

			d := make([]byte, audioFrameSize)
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

			a.playToVirtualSoundcard(d[:bytesToWrite])
			if a.defaultSoundCardStream != nil {
				a.playToDefaultSoundcard(d[:bytesToWrite])
			}
		}
	}
}

func (a *audioStruct) recLoop(deinitNeededChan, deinitFinishedChan chan bool) {
	defer func() {
		deinitFinishedChan <- true
	}()

	frameBuf := make([]byte, audioFrameSize)
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

		// Do not send silence frames to the radio unnecessarily
		if isAllZero(frameBuf[:n]) {
			continue
		}
		buf.Write(frameBuf[:n])

		for buf.Len() >= len(frameBuf) {
			// We need to create a new []byte slice for each chunk to be able to send it through the rec chan.
			b := make([]byte, len(frameBuf))
			n, err = buf.Read(b)
			if err != nil {
				reportError(err)
			}
			if n != len(frameBuf) {
				reportError(errors.New("audio buffer read error"))
			}

			select {
			case a.rec <- b:
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
			a.closeIfNeeded()

			recLoopDeinitNeededChan <- true
			<-recLoopDeinitFinishedChan
			playLoopDeinitNeededChan <- true
			<-playLoopDeinitFinishedChan

			a.deinitFinishedChan <- true
			return
		}

		a.mutex.Lock()
		free := maxPlayBufferSize - a.playBuf.Len()
		if free < len(d) {
			b := make([]byte, len(d)-free)
			_, _ = a.playBuf.Read(b)
		}
		a.playBuf.Write(d)
		a.mutex.Unlock()

		// Non-blocking notify.
		select {
		case a.canPlay <- true:
		default:
		}
	}
}

// We only init the audio once, with the first device name we acquire, so apps using the virtual sound card
// won't have issues with the interface going down while the app is running.
func (a *audioStruct) initIfNeeded(devName string) error {
	a.devName = devName
	bufferSizeInBits := (audioSampleRate * audioSampleBytes * 8) / 1000 * pulseAudioBufferLength.Milliseconds()

	if !a.source.IsOpen() {
		a.source.Name = "kappanhang-" + a.devName
		a.source.Filename = "/tmp/kappanhang-" + a.devName + ".source"
		a.source.Rate = audioSampleRate
		a.source.Format = "s16le"
		a.source.Channels = 1
		a.source.SetProperty("device.buffering.buffer_size", bufferSizeInBits)
		a.source.SetProperty("device.description", "kappanhang: "+a.devName)

		// Cleanup previous pipes.
		sources, err := papipes.GetActiveSources()
		if err == nil {
			for _, i := range sources {
				if i.Filename == a.source.Filename {
					i.Close()
				}
			}
		}

		if err := a.source.Open(); err != nil {
			return err
		}
	}

	if !a.sink.IsOpen() {
		a.sink.Name = "kappanhang-" + a.devName
		a.sink.Filename = "/tmp/kappanhang-" + a.devName + ".sink"
		a.sink.Rate = audioSampleRate
		a.sink.Format = "s16le"
		a.sink.Channels = 1
		a.sink.UseSystemClockForTiming = true
		a.sink.SetProperty("device.buffering.buffer_size", bufferSizeInBits)
		a.sink.SetProperty("device.description", "kappanhang: "+a.devName)

		// Cleanup previous pipes.
		sinks, err := papipes.GetActiveSinks()
		if err == nil {
			for _, i := range sinks {
				if i.Filename == a.sink.Filename {
					i.Close()
				}
			}
		}

		if err := a.sink.Open(); err != nil {
			return err
		}
	}

	if a.playBuf == nil {
		log.Print("opened device " + a.source.Name)

		a.playBuf = bytes.NewBuffer([]byte{})
		a.play = make(chan []byte)
		a.canPlay = make(chan bool)
		a.rec = make(chan []byte)
		a.togglePlaybackToDefaultSoundcardChan = make(chan bool)
		a.deinitNeededChan = make(chan bool)
		a.deinitFinishedChan = make(chan bool)
		go a.loop()
	}
	return nil
}

func (a *audioStruct) closeIfNeeded() {
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
}

func (a *audioStruct) deinit() {
	a.closeIfNeeded()

	if a.deinitNeededChan != nil {
		a.deinitNeededChan <- true
		<-a.deinitFinishedChan
	}
}
