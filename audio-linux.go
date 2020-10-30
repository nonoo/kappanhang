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

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool

	// Send to this channel to play audio.
	play chan []byte
	// Read from this channel for audio.
	rec chan []byte

	virtualSoundcardStream struct {
		source papipes.Source
		sink   papipes.Sink

		mutex   sync.Mutex
		playBuf *bytes.Buffer
		canPlay chan bool
	}

	defaultSoundcardStream struct {
		togglePlaybackChan chan bool
		stream             *pulse.Stream

		mutex   sync.Mutex
		playBuf *bytes.Buffer
		canPlay chan bool
	}
}

var audio audioStruct

func (a *audioStruct) defaultSoundCardStreamDeinit() {
	_ = a.defaultSoundcardStream.stream.Drain()
	a.defaultSoundcardStream.stream.Free()
	a.defaultSoundcardStream.stream = nil
}

func (a *audioStruct) togglePlaybackToDefaultSoundcard() {
	if a.defaultSoundcardStream.togglePlaybackChan == nil {
		return
	}

	// Non-blocking send to channel.
	select {
	case a.defaultSoundcardStream.togglePlaybackChan <- true:
	default:
	}
}

func (a *audioStruct) doTogglePlaybackToDefaultSoundcard() {
	if a.defaultSoundcardStream.stream == nil {
		log.Print("turned on audio playback")
		statusLog.reportAudioMon(true)
		ss := pulse.SampleSpec{Format: pulse.SAMPLE_S16LE, Rate: 48000, Channels: 1}
		a.defaultSoundcardStream.stream, _ = pulse.Playback("kappanhang", a.devName, &ss)
	} else {
		a.defaultSoundCardStreamDeinit()
		log.Print("turned off audio playback")
		statusLog.reportAudioMon(false)
	}
}

func (a *audioStruct) playLoopToDefaultSoundcard(deinitNeededChan, deinitFinishedChan chan bool) {
	for {
		select {
		case <-a.defaultSoundcardStream.canPlay:
		case <-a.defaultSoundcardStream.togglePlaybackChan:
			a.doTogglePlaybackToDefaultSoundcard()
		case <-deinitNeededChan:
			deinitFinishedChan <- true
			return
		}

		for {
			a.defaultSoundcardStream.mutex.Lock()
			if a.defaultSoundcardStream.playBuf.Len() < audioFrameSize {
				a.defaultSoundcardStream.mutex.Unlock()
				break
			}

			d := make([]byte, audioFrameSize)
			bytesToWrite, err := a.defaultSoundcardStream.playBuf.Read(d)
			a.defaultSoundcardStream.mutex.Unlock()
			if err != nil {
				log.Error(err)
				break
			}
			if bytesToWrite != len(d) {
				log.Error("buffer underread")
				break
			}

			for len(d) > 0 && a.defaultSoundcardStream.stream != nil {
				written, err := a.defaultSoundcardStream.stream.Write(d)
				if err != nil {
					if _, ok := err.(*os.PathError); !ok {
						reportError(err)
					}
					break
				}
				d = d[written:]
			}
		}
	}
}

func (a *audioStruct) playLoopToVirtualSoundcard(deinitNeededChan, deinitFinishedChan chan bool) {
	for {
		select {
		case <-a.virtualSoundcardStream.canPlay:
		case <-deinitNeededChan:
			deinitFinishedChan <- true
			return
		}

		for {
			a.virtualSoundcardStream.mutex.Lock()
			if a.virtualSoundcardStream.playBuf.Len() < audioFrameSize {
				a.virtualSoundcardStream.mutex.Unlock()
				break
			}

			d := make([]byte, audioFrameSize)
			bytesToWrite, err := a.virtualSoundcardStream.playBuf.Read(d)
			a.virtualSoundcardStream.mutex.Unlock()
			if err != nil {
				log.Error(err)
				break
			}
			if bytesToWrite != len(d) {
				log.Error("buffer underread")
				break
			}

			for len(d) > 0 {
				written, err := a.virtualSoundcardStream.source.Write(d)
				if err != nil {
					if _, ok := err.(*os.PathError); !ok {
						reportError(err)
					}
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

	frameBuf := make([]byte, audioFrameSize)
	buf := bytes.NewBuffer([]byte{})

	for {
		select {
		case <-deinitNeededChan:
			return
		default:
		}

		n, err := a.virtualSoundcardStream.sink.Read(frameBuf)
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
	go a.playLoopToVirtualSoundcard(playLoopDeinitNeededChan, playLoopDeinitFinishedChan)
	playLoopToDefaultSoundcardDeinitNeededChan := make(chan bool)
	playLoopToDefaultSoundcardDeinitFinishedChan := make(chan bool)
	go a.playLoopToDefaultSoundcard(playLoopToDefaultSoundcardDeinitNeededChan, playLoopToDefaultSoundcardDeinitFinishedChan)

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

			if a.defaultSoundcardStream.stream != nil {
				a.defaultSoundCardStreamDeinit()
			}

			playLoopToDefaultSoundcardDeinitNeededChan <- true
			<-playLoopToDefaultSoundcardDeinitFinishedChan

			a.deinitFinishedChan <- true
			return
		}

		a.virtualSoundcardStream.mutex.Lock()
		free := maxPlayBufferSize - a.virtualSoundcardStream.playBuf.Len()
		if free < len(d) {
			b := make([]byte, len(d)-free)
			_, _ = a.virtualSoundcardStream.playBuf.Read(b)
		}
		a.virtualSoundcardStream.playBuf.Write(d)
		a.virtualSoundcardStream.mutex.Unlock()

		// Non-blocking notify.
		select {
		case a.virtualSoundcardStream.canPlay <- true:
		default:
		}

		if a.defaultSoundcardStream.stream != nil {
			a.defaultSoundcardStream.mutex.Lock()
			free := maxPlayBufferSize - a.defaultSoundcardStream.playBuf.Len()
			if free < len(d) {
				b := make([]byte, len(d)-free)
				_, _ = a.defaultSoundcardStream.playBuf.Read(b)
			}
			a.defaultSoundcardStream.playBuf.Write(d)
			a.defaultSoundcardStream.mutex.Unlock()

			// Non-blocking notify.
			select {
			case a.defaultSoundcardStream.canPlay <- true:
			default:
			}
		}
	}
}

// We only init the audio once, with the first device name we acquire, so apps using the virtual sound card
// won't have issues with the interface going down while the app is running.
func (a *audioStruct) initIfNeeded(devName string) error {
	a.devName = devName
	bufferSizeInBits := (audioSampleRate * audioSampleBytes * 8) / 1000 * pulseAudioBufferLength.Milliseconds()

	if !a.virtualSoundcardStream.source.IsOpen() {
		a.virtualSoundcardStream.source.Name = "kappanhang-" + a.devName
		a.virtualSoundcardStream.source.Filename = "/tmp/kappanhang-" + a.devName + ".source"
		a.virtualSoundcardStream.source.Rate = audioSampleRate
		a.virtualSoundcardStream.source.Format = "s16le"
		a.virtualSoundcardStream.source.Channels = 1
		a.virtualSoundcardStream.source.SetProperty("device.buffering.buffer_size", bufferSizeInBits)
		a.virtualSoundcardStream.source.SetProperty("device.description", "kappanhang: "+a.devName)

		// Cleanup previous pipes.
		sources, err := papipes.GetActiveSources()
		if err == nil {
			for _, i := range sources {
				if i.Filename == a.virtualSoundcardStream.source.Filename {
					i.Close()
				}
			}
		}

		if err := a.virtualSoundcardStream.source.Open(); err != nil {
			return err
		}
	}

	if !a.virtualSoundcardStream.sink.IsOpen() {
		a.virtualSoundcardStream.sink.Name = "kappanhang-" + a.devName
		a.virtualSoundcardStream.sink.Filename = "/tmp/kappanhang-" + a.devName + ".sink"
		a.virtualSoundcardStream.sink.Rate = audioSampleRate
		a.virtualSoundcardStream.sink.Format = "s16le"
		a.virtualSoundcardStream.sink.Channels = 1
		a.virtualSoundcardStream.sink.UseSystemClockForTiming = true
		a.virtualSoundcardStream.sink.SetProperty("device.buffering.buffer_size", bufferSizeInBits)
		a.virtualSoundcardStream.sink.SetProperty("device.description", "kappanhang: "+a.devName)

		// Cleanup previous pipes.
		sinks, err := papipes.GetActiveSinks()
		if err == nil {
			for _, i := range sinks {
				if i.Filename == a.virtualSoundcardStream.sink.Filename {
					i.Close()
				}
			}
		}

		if err := a.virtualSoundcardStream.sink.Open(); err != nil {
			return err
		}
	}

	if a.virtualSoundcardStream.playBuf == nil {
		log.Print("opened device " + a.virtualSoundcardStream.source.Name)

		a.play = make(chan []byte)
		a.rec = make(chan []byte)

		a.virtualSoundcardStream.playBuf = bytes.NewBuffer([]byte{})
		a.defaultSoundcardStream.playBuf = bytes.NewBuffer([]byte{})
		a.virtualSoundcardStream.canPlay = make(chan bool)
		a.defaultSoundcardStream.canPlay = make(chan bool)
		a.defaultSoundcardStream.togglePlaybackChan = make(chan bool)

		a.deinitNeededChan = make(chan bool)
		a.deinitFinishedChan = make(chan bool)
		go a.loop()
	}
	return nil
}

func (a *audioStruct) closeIfNeeded() {
	if a.virtualSoundcardStream.source.IsOpen() {
		if err := a.virtualSoundcardStream.source.Close(); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				log.Error(err)
			}
		}
	}

	if a.virtualSoundcardStream.sink.IsOpen() {
		if err := a.virtualSoundcardStream.sink.Close(); err != nil {
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
