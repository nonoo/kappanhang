package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"

	"github.com/nonoo/kappanhang/log"
)

const pkt7TimeoutDuration = 3 * time.Second

type pkt7Type struct {
	sendSeq          uint16
	innerSendSeq     uint16
	lastConfirmedSeq uint16

	sendTicker   *time.Ticker
	timeoutTimer *time.Timer
	latency      time.Duration
	lastSendAt   time.Time

	periodicStopNeededChan   chan bool
	periodicStopFinishedChan chan bool
}

func (p *pkt7Type) isPkt7(r []byte) bool {
	return len(r) == 21 && bytes.Equal(r[1:6], []byte{0x00, 0x00, 0x00, 0x07, 0x00}) // Note that the first byte can be 0x15 or 0x00, so we ignore that.
}

func (p *pkt7Type) handle(s *streamCommon, r []byte) error {
	gotSeq := binary.LittleEndian.Uint16(r[6:8])
	if r[16] == 0x00 { // This is a pkt7 request from the radio.
		// Replying to the radio.
		// Example request from radio: 0x00, 0x00, 0x00, 0x00, 0x07, 0x00, 0x1c, 0x0e, 0xe4, 0x35, 0xdd, 0x72, 0xbe, 0xd9, 0xf2, 0x63, 0x00, 0x57, 0x2b, 0x12, 0x00
		// Example answer from PC:     0x15, 0x00, 0x00, 0x00, 0x07, 0x00, 0x1c, 0x0e, 0xbe, 0xd9, 0xf2, 0x63, 0xe4, 0x35, 0xdd, 0x72, 0x01, 0x57, 0x2b, 0x12, 0x00
		if p.sendTicker != nil { // Only replying if the auth is already done.
			if err := p.sendReply(s, r[17:21], gotSeq); err != nil {
				return err
			}
		}
	} else { // This is a pkt7 reply to our request.
		if p.sendTicker != nil { // Auth is already done?
			if p.timeoutTimer != nil {
				p.timeoutTimer.Stop()
				p.timeoutTimer.Reset(pkt7TimeoutDuration)
			}

			// Only measure latency after the timeout has been initialized, so the auth is already done.
			p.latency += time.Since(p.lastSendAt)
			p.latency /= 2
		}

		expectedSeq := p.lastConfirmedSeq + 1
		if expectedSeq != gotSeq && gotSeq != p.lastConfirmedSeq {
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
	return nil
}

func (p *pkt7Type) sendDo(s *streamCommon, replyID []byte, seq uint16) error {
	// Example request from PC:  0x15, 0x00, 0x00, 0x00, 0x07, 0x00, 0x09, 0x00, 0xbe, 0xd9, 0xf2, 0x63, 0xe4, 0x35, 0xdd, 0x72, 0x00, 0x78, 0x40, 0xf6, 0x02
	// Example reply from radio: 0x00, 0x00, 0x00, 0x00, 0x07, 0x00, 0x09, 0x00, 0xe4, 0x35, 0xdd, 0x72, 0xbe, 0xd9, 0xf2, 0x63, 0x01, 0x78, 0x40, 0xf6, 0x02
	var replyFlag byte
	if replyID == nil {
		replyID = make([]byte, 4)
		var randID [1]byte
		if _, err := rand.Read(randID[:]); err != nil {
			return err
		}
		replyID[0] = randID[0]
		replyID[1] = byte(p.innerSendSeq)
		replyID[2] = byte(p.innerSendSeq >> 8)
		replyID[3] = 0x06
		p.innerSendSeq++
	} else {
		replyFlag = 0x01
	}

	d := []byte{0x15, 0x00, 0x00, 0x00, 0x07, 0x00, byte(seq), byte(seq >> 8),
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID),
		replyFlag, replyID[0], replyID[1], replyID[2], replyID[3]}
	if err := s.send(d); err != nil {
		return err
	}
	return nil
}

func (p *pkt7Type) send(s *streamCommon) error {
	if err := p.sendDo(s, nil, p.sendSeq); err != nil {
		return err
	}
	p.lastSendAt = time.Now()
	p.sendSeq++
	return nil
}

func (p *pkt7Type) sendReply(s *streamCommon, replyID []byte, seq uint16) error {
	return p.sendDo(s, replyID, seq)
}

func (p *pkt7Type) loop(s *streamCommon) {
	for {
		if p.timeoutTimer != nil {
			select {
			case <-p.sendTicker.C:
				if err := p.send(s); err != nil {
					reportError(err)
				}
			case <-p.timeoutTimer.C:
				reportError(errors.New(s.name + "/ping timeout"))
			case <-p.periodicStopNeededChan:
				p.periodicStopFinishedChan <- true
				return
			}
		} else {
			select {
			case <-p.sendTicker.C:
				if err := p.send(s); err != nil {
					reportError(err)
				}
			case <-p.periodicStopNeededChan:
				p.periodicStopFinishedChan <- true
				return
			}
		}
	}
}

func (p *pkt7Type) startPeriodicSend(s *streamCommon, firstSeqNo uint16, checkPingTimeout bool) {
	p.sendSeq = firstSeqNo
	p.innerSendSeq = 0x8304
	p.lastConfirmedSeq = p.sendSeq - 1

	p.sendTicker = time.NewTicker(100 * time.Millisecond)
	if checkPingTimeout {
		p.timeoutTimer = time.NewTimer(pkt7TimeoutDuration)
	}

	p.periodicStopNeededChan = make(chan bool)
	p.periodicStopFinishedChan = make(chan bool)
	go p.loop(s)
}

func (p *pkt7Type) stopPeriodicSend() {
	if p.sendTicker == nil { // Periodic send has not started?
		return
	}

	p.periodicStopNeededChan <- true
	<-p.periodicStopFinishedChan

	if p.timeoutTimer != nil {
		p.timeoutTimer.Stop()
	}
	p.sendTicker.Stop()
}
