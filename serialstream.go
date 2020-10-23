package main

import (
	"github.com/nonoo/kappanhang/log"
)

type serialStream struct {
	common streamCommon

	serialPort serialPortStruct

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool
}

func (s *serialStream) handleRead(r []byte) {
}

func (s *serialStream) loop() {
	for {
		select {
		case r := <-s.common.readChan:
			s.handleRead(r)
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
	if s.deinitNeededChan != nil {
		s.deinitNeededChan <- true
		<-s.deinitFinishedChan
	}
	s.common.deinit()
	s.serialPort.deinit()
}
