package main

import (
	"bytes"
	"encoding/binary"
	"sync"
	"time"
)

const pkt0DefaultSendInterval = 100 * time.Millisecond
const pkt0IdleAfter = time.Second
const pkt0IdleSendInterval = time.Second

type pkt0Type struct {
	sendSeq uint16
	mutex   sync.Mutex // Protects sendSeq

	sendTimer         *time.Timer
	lastTrackedSentAt time.Time

	txSeqBuf txSeqBufStruct

	periodicIntervalResetChan chan bool
	periodicStopNeededChan    chan bool
	periodicStopFinishedChan  chan bool
}

func (p *pkt0Type) retransmitRange(s *streamCommon, start, end uint16) error {
	log.Debug(s.name+"/got retransmit request for #", start, "-", end)
	for {
		netstat.reportRetransmit(1)
		d := p.txSeqBuf.get(seqNum(start))
		if d != nil {
			log.Debug(s.name+"/retransmitting #", start)
			if err := s.send(d); err != nil {
				return err
			}
			if err := s.send(d); err != nil {
				return err
			}
		} else {
			log.Debug(s.name+"/can't retransmit #", start, " - not found ")

			// Sending an idle with the requested seqnum.
			if err := p.sendIdle(s, false, start); err != nil {
				return err
			}
			if err := p.sendIdle(s, false, start); err != nil {
				return err
			}
		}

		if start == end {
			break
		}
		start++
	}
	return nil
}

func (p *pkt0Type) handle(s *streamCommon, r []byte) error {
	if len(r) < 16 {
		return nil
	}

	if bytes.Equal(r[:6], []byte{0x10, 0x00, 0x00, 0x00, 0x01, 0x00}) {
		seq := binary.LittleEndian.Uint16(r[6:8])
		d := p.txSeqBuf.get(seqNum(seq))
		log.Debug(s.name+"/got retransmit request for #", seq)
		if d != nil {
			log.Debug(s.name+"/retransmitting #", seq)
			netstat.reportRetransmit(1)
			if err := s.send(d); err != nil {
				return err
			}
			if err := s.send(d); err != nil {
				return err
			}
		} else {
			log.Debug(s.name+"/can't retransmit #", seq, " - not found")

			// Sending an idle with the requested seqnum.
			if err := p.sendIdle(s, false, seq); err != nil {
				return err
			}
			if err := p.sendIdle(s, false, seq); err != nil {
				return err
			}
		}
	} else if bytes.Equal(r[:6], []byte{0x18, 0x00, 0x00, 0x00, 0x01, 0x00}) {
		r = r[16:]
		for len(r) >= 4 {
			start := binary.LittleEndian.Uint16(r[0:2])
			end := binary.LittleEndian.Uint16(r[2:4])
			if err := p.retransmitRange(s, start, end); err != nil {
				return err
			}
			r = r[4:]
		}
	}
	return nil
}

func (p *pkt0Type) isIdlePkt0(r []byte) bool {
	return len(r) == 16 && bytes.Equal(r[:6], []byte{0x10, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func (p *pkt0Type) isPkt0(r []byte) bool {
	return len(r) >= 16 && (p.isIdlePkt0(r) ||
		bytes.Equal(r[:6], []byte{0x10, 0x00, 0x00, 0x00, 0x01, 0x00}) || // Retransmit request for 1 packet.
		bytes.Equal(r[:6], []byte{0x18, 0x00, 0x00, 0x00, 0x01, 0x00})) // Retransmit request for ranges.
}

//var drop int

// The radio can request retransmit for tracked packets. If there are no tracked packets to send, idle pkt0
// packets are periodically sent.
func (p *pkt0Type) sendTrackedPacket(s *streamCommon, d []byte) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// if s.name == "audio" {
	// 	if drop == 0 && time.Now().UnixNano()%100 == 0 {
	// 		log.Print(s.name+"/drop start - ", p.sendSeq)
	// 		drop = 1
	// 	} else if drop > 0 {
	// 		drop++
	// 		if drop == 3 {
	// 			log.Print(s.name+"/drop stop - ", p.sendSeq)
	// 			drop = 0
	// 		}
	// 	}
	// }

	d[6] = byte(p.sendSeq)
	d[7] = byte(p.sendSeq >> 8)
	p.txSeqBuf.add(seqNum(p.sendSeq), d)
	// if s.name != "audio" || drop == 0 {
	if err := s.send(d); err != nil {
		return err
	}
	// }
	p.sendSeq++

	if !p.isIdlePkt0(d) {
		p.lastTrackedSentAt = time.Now()
		if p.periodicIntervalResetChan != nil {
			// Non-blocking send.
			select {
			case p.periodicIntervalResetChan <- true:
			default:
			}
		}
	}

	return nil
}

func (p *pkt0Type) sendIdle(s *streamCommon, tracked bool, seqIfUntracked uint16) error {
	d := []byte{0x10, 0x00, 0x00, 0x00, 0x00, 0x00, byte(seqIfUntracked), byte(seqIfUntracked >> 8),
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)}
	if tracked {
		return p.sendTrackedPacket(s, d)
	}
	return s.send(d)
}

func (p *pkt0Type) loop(s *streamCommon) {
	for {
		select {
		case <-p.periodicIntervalResetChan:
			if !p.sendTimer.Stop() {
				<-p.sendTimer.C
			}
			p.sendTimer.Reset(pkt0DefaultSendInterval)
		case <-p.sendTimer.C:
			if err := p.sendIdle(s, true, 0); err != nil {
				reportError(err)
			}

			if time.Since(p.lastTrackedSentAt) >= pkt0IdleAfter {
				p.sendTimer.Reset(pkt0IdleSendInterval)
			} else {
				p.sendTimer.Reset(pkt0DefaultSendInterval)
			}
		case <-p.periodicStopNeededChan:
			p.sendTimer.Stop()
			p.periodicStopFinishedChan <- true
			return
		}
	}
}

func (p *pkt0Type) startPeriodicSend(s *streamCommon) {
	p.sendTimer = time.NewTimer(pkt0IdleSendInterval)

	p.periodicIntervalResetChan = make(chan bool)
	p.periodicStopNeededChan = make(chan bool)
	p.periodicStopFinishedChan = make(chan bool)
	go p.loop(s)
}

func (p *pkt0Type) stopPeriodicSend() {
	if p.sendTimer == nil { // Periodic send has not started?
		return
	}

	p.periodicStopNeededChan <- true
	<-p.periodicStopFinishedChan
}

func (p *pkt0Type) init(s *streamCommon) {
	p.sendSeq = 1
}
