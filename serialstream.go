package main

import (
	"github.com/nonoo/kappanhang/log"
)

type serialStream struct {
	common streamCommon
}

func (s *serialStream) init() {
	s.common.open("serial", 50002)
}

func (s *serialStream) handleRead(r []byte) {
}

func (s *serialStream) start() {
	s.common.sendPkt3()
	s.common.waitForPkt4Answer()
	s.common.sendPkt6()
	s.common.waitForPkt6Answer()

	log.Print("stream started")

	s.common.pkt7.startPeriodicSend(&s.common, 1, false)

	for {
		select {
		case r := <-s.common.readChan:
			s.handleRead(r)
		}
	}
}

func (s *serialStream) sendDisconnect() {
	s.common.sendDisconnect()
}
