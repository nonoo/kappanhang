package main

import (
	"bytes"
	"encoding/binary"
	"time"
)

type audioStream struct {
	common streamCommon
}

func (s *audioStream) sendDisconnect() {
	s.common.sendDisconnect()
}

func (s *audioStream) handleRead(r []byte) {
	switch len(r) {
	case 21:
		if bytes.Equal(r[1:6], []byte{0x00, 0x00, 0x00, 0x07, 0x00}) { // Note that the first byte can be 0x15 or 0x00, so we ignore that.
			gotSeq := binary.LittleEndian.Uint16(r[6:8])
			if r[16] == 0x00 { // This is a pkt7 request from the radio.
				// Replying to the radio.
				// Example request from radio: 0x00, 0x00, 0x00, 0x00, 0x07, 0x00, 0x1c, 0x0e, 0xe4, 0x35, 0xdd, 0x72, 0xbe, 0xd9, 0xf2, 0x63, 0x00, 0x57, 0x2b, 0x12, 0x00
				// Example answer from PC:     0x15, 0x00, 0x00, 0x00, 0x07, 0x00, 0x1c, 0x0e, 0xbe, 0xd9, 0xf2, 0x63, 0xe4, 0x35, 0xdd, 0x72, 0x01, 0x57, 0x2b, 0x12, 0x00
				s.common.sendPkt7Reply(r[17:21], gotSeq)
			} else { // This is a pkt7 reply to our request.
			}
		}
	}
}

func (s *audioStream) start() {
	s.common.open("audio", 50003)

	s.common.sendPkt3()
	s.common.waitForPkt4Answer()
	s.common.sendPkt6()
	s.common.waitForPkt6Answer()

	s.common.pkt7.sendSeq = 1

	pingTicker := time.NewTicker(100 * time.Millisecond)

	var r []byte
	for {
		select {
		case r = <-s.common.readChan:
			s.handleRead(r)
		case <-pingTicker.C:
			// s.expectedPkt7ReplySeq = s.common.pkt7.sendSeq
			// s.lastPkt7SendAt = time.Now()
			s.common.sendPkt7()
		}
	}
}
