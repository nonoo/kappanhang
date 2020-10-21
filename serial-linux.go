package main

import (
	"github.com/google/goterm/term"
	"github.com/nonoo/kappanhang/log"
)

type serialPortStruct struct {
	pty *term.PTY
}

var serialPort serialPortStruct

func (s *serialPortStruct) init() {
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

func (s *serialPortStruct) deinit() {
	if s.pty != nil {
		s.pty.Close()
	}
}
