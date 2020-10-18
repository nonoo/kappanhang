package main

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/nonoo/kappanhang/log"
)

const audioTimeoutDuration = 3 * time.Second

type audioStream struct {
	common streamCommon

	timeoutTimer         *time.Timer
	receivedAudio        bool
	lastReceivedAudioSeq uint16

	audioSendSeq uint16
}

func (s *audioStream) sendDisconnect() {
	s.common.sendDisconnect()
}

func (s *audioStream) handleAudioPacket(r []byte) {
	if s.timeoutTimer != nil {
		s.timeoutTimer.Stop()
		s.timeoutTimer.Reset(audioTimeoutDuration)
	}

	gotSeq := binary.LittleEndian.Uint16(r[6:8])
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

	// log.Print("got audio packet ", len(r[24:]), " bytes")
}

func (s *audioStream) handleRead(r []byte) {
	if len(r) >= 580 && (bytes.Equal(r[:6], []byte{0x6c, 0x05, 0x00, 0x00, 0x00, 0x00}) || bytes.Equal(r[:6], []byte{0x44, 0x02, 0x00, 0x00, 0x00, 0x00})) {
		s.handleAudioPacket(r)
	}
}

// sendPart1 expects 1364 bytes of PCM data.
func (s *audioStream) sendPart1(pcmData []byte) {
	s.common.send(append([]byte{0x6c, 0x05, 0x00, 0x00, 0x00, 0x00, byte(s.audioSendSeq), byte(s.audioSendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0x80, 0x00, byte((s.audioSendSeq - 1) >> 8), byte(s.audioSendSeq - 1), 0x00, 0x00, byte(len(pcmData) >> 8), byte(len(pcmData))},
		pcmData...))
	s.audioSendSeq++
}

// sendPart2 expects 556 bytes of PCM data.
func (s *audioStream) sendPart2(pcmData []byte) {
	s.common.send(append([]byte{0x44, 0x02, 0x00, 0x00, 0x00, 0x00, byte(s.audioSendSeq), byte(s.audioSendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0x80, 0x00, byte((s.audioSendSeq - 1) >> 8), byte(s.audioSendSeq - 1), 0x00, 0x00, byte(len(pcmData) >> 8), byte(len(pcmData))},
		pcmData...))
	s.audioSendSeq++
}

func (s *audioStream) init() {
	s.common.open("audio", 50003)
}

func (s *audioStream) start() {
	s.common.sendPkt3()
	s.common.waitForPkt4Answer()
	s.common.sendPkt6()
	s.common.waitForPkt6Answer()

	log.Print("stream started")

	s.timeoutTimer = time.NewTimer(audioTimeoutDuration)

	s.common.pkt7.startPeriodicSend(&s.common, 1)

	s.audioSendSeq = 1

	testSendTicker := time.NewTicker(80 * time.Millisecond) // TODO: remove

	var r []byte
	for {
		select {
		case r = <-s.common.readChan:
			s.handleRead(r)
		case <-s.timeoutTimer.C:
			log.Fatal("timeout")
		case <-testSendTicker.C: // TODO: remove
			b1 := make([]byte, 1364)
			s.sendPart1(b1)
			b2 := make([]byte, 556)
			s.sendPart2(b2)
		}
	}
}
