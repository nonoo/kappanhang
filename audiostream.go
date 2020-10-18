package main

import "github.com/nonoo/kappanhang/log"

type audioStream struct {
	common streamCommon
}

func (s *audioStream) sendDisconnect() {
	s.common.sendDisconnect()
}

func (s *audioStream) handleRead(r []byte) {
	// TODO
}

func (s *audioStream) start() {
	s.common.open("audio", 50003)

	s.common.sendPkt3()
	s.common.waitForPkt4Answer()
	s.common.sendPkt6()
	s.common.waitForPkt6Answer()

	log.Print("stream opened")

	s.common.pkt7.sendSeq = 1
	s.common.pkt7.startPeriodicSend(&s.common)

	var r []byte
	for {
		select {
		case r = <-s.common.readChan:
			s.handleRead(r)
		}
	}
}
