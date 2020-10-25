package main

import (
	"sync"
	"time"
)

type pkt0Type struct {
	sendSeq uint16
	mutex   sync.Mutex

	sendTicker *time.Ticker

	periodicStopNeededChan   chan bool
	periodicStopFinishedChan chan bool
}

func (p *pkt0Type) sendSeqLock() {
	p.mutex.Lock()
}

func (p *pkt0Type) sendSeqUnlock() {
	p.mutex.Unlock()
}

func (p *pkt0Type) send(s *streamCommon) error {
	p.sendSeqLock()
	defer p.sendSeqUnlock()

	d := []byte{0x10, 0x00, 0x00, 0x00, 0x00, 0x00, byte(p.sendSeq), byte(p.sendSeq >> 8),
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)}
	if err := s.send(d); err != nil {
		return err
	}
	p.sendSeq++
	return nil
}

func (p *pkt0Type) loop(s *streamCommon) {
	for {
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

func (p *pkt0Type) startPeriodicSend(s *streamCommon) {
	p.sendTicker = time.NewTicker(100 * time.Millisecond)

	p.periodicStopNeededChan = make(chan bool)
	p.periodicStopFinishedChan = make(chan bool)
	go p.loop(s)
}

func (p *pkt0Type) stopPeriodicSend() {
	if p.sendTicker == nil { // Periodic send has not started?
		return
	}

	p.periodicStopNeededChan <- true
	<-p.periodicStopFinishedChan

	p.sendTicker.Stop()
}
