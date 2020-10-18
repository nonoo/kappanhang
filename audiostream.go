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
	lastReceivedAudioSeq uint16
}

func (s *audioStream) sendDisconnect() {
	s.common.sendDisconnect()
}

func (s *audioStream) handleAudioPacket(r []byte) {
	gotSeq := binary.LittleEndian.Uint16(r[6:8])

	if s.timeoutTimer != nil {
		s.timeoutTimer.Stop()
		s.timeoutTimer.Reset(audioTimeoutDuration)
	}

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
	s.lastReceivedAudioSeq = gotSeq

	// log.Print("got audio packet ", len(r[24:]), " bytes")
}

func (s *audioStream) handleRead(r []byte) {
	if len(r) >= 580 && (bytes.Equal(r[:6], []byte{0x6c, 0x05, 0x00, 0x00, 0x00, 0x00}) || bytes.Equal(r[:6], []byte{0x44, 0x02, 0x00, 0x00, 0x00, 0x00})) {
		s.handleAudioPacket(r)
	}
}

func (s *audioStream) start() {
	s.common.open("audio", 50003)

	s.common.sendPkt3()
	s.common.waitForPkt4Answer()
	s.common.sendPkt6()
	s.common.waitForPkt6Answer()

	log.Print("stream opened")

	s.timeoutTimer = time.NewTimer(audioTimeoutDuration)

	s.common.pkt7.sendSeq = 1
	s.common.pkt7.startPeriodicSend(&s.common)

	var r []byte
	for {
		select {
		case r = <-s.common.readChan:
			s.handleRead(r)
		case <-s.timeoutTimer.C:
			log.Fatal("timeout")
		}
	}
}
