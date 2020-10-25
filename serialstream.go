package main

import (
	"github.com/nonoo/kappanhang/log"
)

type serialStream struct {
	common streamCommon

	serialPort serialPortStruct

	sendSeq uint16

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool
}

func (s *serialStream) send(d []byte) error {
	s.common.pkt0.sendSeqLock()
	defer s.common.pkt0.sendSeqUnlock()

	l := byte(len(d))
	p := append([]byte{0x15 + l, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.common.pkt0.sendSeq), byte(s.common.pkt0.sendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0xc1, l, 0x00, byte(s.sendSeq >> 8), byte(s.sendSeq)}, d...)
	if err := s.common.send(p); err != nil {
		return err
	}
	s.sendSeq++
	return nil
}

func (s *serialStream) sendOpenClose(close bool) error {
	s.common.pkt0.sendSeqLock()
	defer s.common.pkt0.sendSeqUnlock()

	var magic byte
	if close {
		magic = 0x00
	} else {
		magic = 0x05
	}

	p := []byte{0x16, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.common.pkt0.sendSeq), byte(s.common.pkt0.sendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0xc0, 0x01, 0x00, byte(s.sendSeq >> 8), byte(s.sendSeq), magic}
	if err := s.common.send(p); err != nil {
		return err
	}
	s.sendSeq++
	return nil
}

func (s *serialStream) handleRead(r []byte) {
	if len(r) >= 22 {
		if r[16] == 0xc1 && r[0]-0x15 == r[17] {
			s.serialPort.write <- r[21:]
		}
	}

}

func (s *serialStream) loop() {
	for {
		select {
		case r := <-s.common.readChan:
			s.handleRead(r)
		case r := <-s.serialPort.read:
			if err := s.send(r); err != nil {
				reportError(err)
			}
		case <-s.deinitNeededChan:
			s.deinitFinishedChan <- true
			return
		}
	}
}

func (s *serialStream) start(devName string) error {
	if err := s.serialPort.init(devName); err != nil {
		return err
	}

	if err := s.common.sendPkt3(); err != nil {
		return err
	}
	if err := s.common.waitForPkt4Answer(); err != nil {
		return err
	}
	if err := s.common.sendPkt6(); err != nil {
		return err
	}
	if err := s.common.waitForPkt6Answer(); err != nil {
		return err
	}

	log.Print("stream started")

	s.common.pkt7.startPeriodicSend(&s.common, 1, false)
	s.common.pkt0.startPeriodicSend(&s.common)

	if err := s.sendOpenClose(false); err != nil {
		return err
	}

	s.deinitNeededChan = make(chan bool)
	s.deinitFinishedChan = make(chan bool)
	go s.loop()
	return nil
}

func (s *serialStream) init() error {
	if err := s.common.init("serial", 50002); err != nil {
		return err
	}
	return nil
}

func (s *serialStream) deinit() {
	if s.common.pkt0.sendTicker != nil { // Stream opened?
		_ = s.sendOpenClose(true)
	}

	if s.deinitNeededChan != nil {
		s.deinitNeededChan <- true
		<-s.deinitFinishedChan
	}
	s.common.deinit()
	s.serialPort.deinit()
}
