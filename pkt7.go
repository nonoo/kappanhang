package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"time"

	"github.com/nonoo/kappanhang/log"
)

const pkt7TimeoutDuration = 3 * time.Second

type pkt7Type struct {
	sendSeq          uint16
	randIDBytes      [2]byte
	lastConfirmedSeq uint16

	sendTicker   *time.Ticker
	timeoutTimer *time.Timer
	latency      time.Duration
	lastSendAt   time.Time
}

func (p *pkt7Type) isPkt7(r []byte) bool {
	return len(r) == 21 && bytes.Equal(r[1:6], []byte{0x00, 0x00, 0x00, 0x07, 0x00}) // Note that the first byte can be 0x15 or 0x00, so we ignore that.
}

func (p *pkt7Type) tryReceive(timeout time.Duration, s *streamCommon) []byte {
	return s.tryReceivePacket(timeout, 21, 1, []byte{0x00, 0x00, 0x00, 0x07, 0x00})
}

func (p *pkt7Type) handle(s *streamCommon, r []byte) {
	gotSeq := binary.LittleEndian.Uint16(r[6:8])
	if r[16] == 0x00 { // This is a pkt7 request from the radio.
		// Replying to the radio.
		// Example request from radio: 0x00, 0x00, 0x00, 0x00, 0x07, 0x00, 0x1c, 0x0e, 0xe4, 0x35, 0xdd, 0x72, 0xbe, 0xd9, 0xf2, 0x63, 0x00, 0x57, 0x2b, 0x12, 0x00
		// Example answer from PC:     0x15, 0x00, 0x00, 0x00, 0x07, 0x00, 0x1c, 0x0e, 0xbe, 0xd9, 0xf2, 0x63, 0xe4, 0x35, 0xdd, 0x72, 0x01, 0x57, 0x2b, 0x12, 0x00
		if p.timeoutTimer != nil { // Only replying if the auth is already done.
			p.sendReply(s, r[17:21], gotSeq)
		}
	} else { // This is a pkt7 reply to our request.
		if p.timeoutTimer != nil {
			p.timeoutTimer.Stop()
			p.timeoutTimer.Reset(pkt7TimeoutDuration)

			// Only measure latency after the timeout has been initialized, so the auth is already done.
			p.latency += time.Since(p.lastSendAt)
			p.latency /= 2
		}

		expectedSeq := p.lastConfirmedSeq + 1
		if expectedSeq != gotSeq {
			var missingPkts int
			if gotSeq > expectedSeq {
				missingPkts = int(gotSeq) - int(expectedSeq)
			} else {
				missingPkts = int(gotSeq) + 65536 - int(expectedSeq)
			}
			log.Error(s.name+"/lost ", missingPkts, " packets")
		}
		p.lastConfirmedSeq = gotSeq
	}
}

func (p *pkt7Type) sendDo(s *streamCommon, replyID []byte, seq uint16) {
	// Example request from PC:  0x15, 0x00, 0x00, 0x00, 0x07, 0x00, 0x09, 0x00, 0xbe, 0xd9, 0xf2, 0x63, 0xe4, 0x35, 0xdd, 0x72, 0x00, 0x78, 0x40, 0xf6, 0x02
	// Example reply from radio: 0x00, 0x00, 0x00, 0x00, 0x07, 0x00, 0x09, 0x00, 0xe4, 0x35, 0xdd, 0x72, 0xbe, 0xd9, 0xf2, 0x63, 0x01, 0x78, 0x40, 0xf6, 0x02
	var replyFlag byte
	if replyID == nil {
		replyID = make([]byte, 4)
		var randID [2]byte
		_, err := rand.Read(randID[:])
		if err != nil {
			log.Fatal(err)
		}
		replyID[0] = randID[0]
		replyID[1] = randID[1]
		replyID[2] = p.randIDBytes[0]
		replyID[3] = p.randIDBytes[1]
	} else {
		replyFlag = 0x01
	}

	s.send([]byte{0x15, 0x00, 0x00, 0x00, 0x07, 0x00, byte(seq), byte(seq >> 8),
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID),
		replyFlag, replyID[0], replyID[1], replyID[2], replyID[3]})
}

func (p *pkt7Type) send(s *streamCommon) {
	p.sendDo(s, nil, p.sendSeq)
	p.lastSendAt = time.Now()
	p.sendSeq++
}

func (p *pkt7Type) sendReply(s *streamCommon, replyID []byte, seq uint16) {
	p.sendDo(s, replyID, seq)
}

func (p *pkt7Type) startPeriodicSend(s *streamCommon, firstSeqNo uint16) {
	p.sendSeq = firstSeqNo
	p.lastConfirmedSeq = p.sendSeq - 1

	p.sendTicker = time.NewTicker(100 * time.Millisecond)
	p.timeoutTimer = time.NewTimer(pkt7TimeoutDuration)

	go func() {
		for {
			select {
			case <-p.sendTicker.C:
				p.send(s)
			case <-p.timeoutTimer.C:
				log.Fatal(s.name + "/ping timeout")
			}
		}
	}()
}
