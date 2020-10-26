package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"
)

const audioTimeoutDuration = 3 * time.Second
const audioRxSeqBufLength = 100 * time.Millisecond

type audioStream struct {
	common streamCommon

	audio audioStruct

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool

	timeoutTimer         *time.Timer
	receivedAudio        bool
	lastReceivedAudioSeq uint16

	rxSeqBuf          seqBuf
	rxSeqBufEntryChan chan seqBufEntry

	audioSendSeq uint16
}

// sendPart1 expects 1364 bytes of PCM data.
func (s *audioStream) sendPart1(pcmData []byte) error {
	err := s.common.pkt0.sendTrackedPacket(&s.common,
		append([]byte{0x6c, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
			byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
			0x80, 0x00, byte((s.audioSendSeq - 1) >> 8), byte(s.audioSendSeq - 1), 0x00, 0x00, byte(len(pcmData) >> 8), byte(len(pcmData))},
			pcmData...))
	if err != nil {
		return err
	}
	s.audioSendSeq++
	return nil
}

// sendPart2 expects 556 bytes of PCM data.
func (s *audioStream) sendPart2(pcmData []byte) error {
	err := s.common.pkt0.sendTrackedPacket(&s.common, append([]byte{0x44, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0x80, 0x00, byte((s.audioSendSeq - 1) >> 8), byte(s.audioSendSeq - 1), 0x00, 0x00, byte(len(pcmData) >> 8), byte(len(pcmData))},
		pcmData...))
	if err != nil {
		return err
	}
	s.audioSendSeq++
	return nil
}

func (s *audioStream) handleRxSeqBufEntry(e seqBufEntry) {
	gotSeq := uint16(e.seq)
	if s.receivedAudio {
		expectedSeq := s.lastReceivedAudioSeq + 1
		if expectedSeq != gotSeq {
			var missingPkts int
			if gotSeq > expectedSeq {
				missingPkts = int(gotSeq) - int(expectedSeq)
			} else {
				missingPkts = int(gotSeq) + 65536 - int(expectedSeq)
			}
			log.Error("lost ", missingPkts, " audio packets")
		}
	}
	s.lastReceivedAudioSeq = gotSeq
	s.receivedAudio = true

	s.audio.play <- e.data
}

func (s *audioStream) handleAudioPacket(r []byte) error {
	if s.timeoutTimer != nil {
		s.timeoutTimer.Stop()
		s.timeoutTimer.Reset(audioTimeoutDuration)
	}

	gotSeq := binary.LittleEndian.Uint16(r[6:8])
	addedToFront, _ := s.rxSeqBuf.add(seqNum(gotSeq), r[24:])

	// If the packet is not added to the front of the seqbuf, then it means that it was an answer for a
	// retransmit request (or it was an out of order packet which we don't want start a retransmit).
	if !addedToFront {
		return nil
	}

	return s.common.requestRetransmitIfNeeded(gotSeq)
}

func (s *audioStream) handleRead(r []byte) error {
	if len(r) >= 580 && (bytes.Equal(r[:6], []byte{0x6c, 0x05, 0x00, 0x00, 0x00, 0x00}) || bytes.Equal(r[:6], []byte{0x44, 0x02, 0x00, 0x00, 0x00, 0x00})) {
		return s.handleAudioPacket(r)
	}
	return nil
}

func (s *audioStream) loop() {
	for {
		select {
		case r := <-s.common.readChan:
			if err := s.handleRead(r); err != nil {
				reportError(err)
			}
		case <-s.timeoutTimer.C:
			reportError(errors.New("audio stream timeout, try rebooting the radio"))
		case e := <-s.rxSeqBufEntryChan:
			s.handleRxSeqBufEntry(e)
		case d := <-s.audio.rec:
			if err := s.sendPart1(d[:1364]); err != nil {
				reportError(err)
			}
			if err := s.sendPart2(d[1364:1920]); err != nil {
				reportError(err)
			}
		case <-s.deinitNeededChan:
			s.deinitFinishedChan <- true
			return
		}
	}
}

func (s *audioStream) init(devName string) error {
	if err := s.common.init("audio", audioStreamPort); err != nil {
		return err
	}

	if err := s.audio.init(devName); err != nil {
		return err
	}

	if err := s.common.start(); err != nil {
		return err
	}

	s.common.pkt7.startPeriodicSend(&s.common, 1, false)
	// This stream does not use periodic pkt0 idle packets.
	s.audioSendSeq = 1

	log.Print("stream started")

	s.rxSeqBufEntryChan = make(chan seqBufEntry)
	s.rxSeqBuf.init(audioRxSeqBufLength, 0xffff, 0, s.rxSeqBufEntryChan)

	s.timeoutTimer = time.NewTimer(audioTimeoutDuration)

	s.deinitNeededChan = make(chan bool)
	s.deinitFinishedChan = make(chan bool)
	go s.loop()
	return nil
}

func (s *audioStream) deinit() {
	if s.deinitNeededChan != nil {
		s.deinitNeededChan <- true
		<-s.deinitFinishedChan
	}
	if s.timeoutTimer != nil {
		s.timeoutTimer.Stop()
	}
	s.common.deinit()
	s.rxSeqBuf.deinit()
	s.audio.deinit()
}
