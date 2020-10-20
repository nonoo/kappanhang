package main

import (
	"github.com/google/goterm/term"
	"github.com/nonoo/kappanhang/log"
)

type serialStream struct {
	common streamCommon

	pty *term.PTY
}

func (s *serialStream) init() {
	s.common.open("serial", 50002)

	var err error
	s.pty, err = term.OpenPTY()
	if err != nil {
		exit(err)
	}
	n, err := s.pty.PTSName()
	if err != nil {
		exit(err)
	}
	log.Print("opened ", n)
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

func (s *serialStream) deinit() {
	s.common.sendDisconnect()
	if s.pty != nil {
		s.pty.Close()
	}
}
