package main

import (
	"os"

	"github.com/google/goterm/term"
	"github.com/nonoo/kappanhang/log"
)

type serialPortStruct struct {
	pty *term.PTY

	// Read from this channel to receive serial data.
	read chan []byte
	// Write to this channel to send serial data.
	write chan []byte
}

var serialPort serialPortStruct

func (s *serialPortStruct) writeLoop() {
	s.write = make(chan []byte)
	var b []byte
	for {
		b = <-s.write
		bytesToWrite := len(b)

		for bytesToWrite > 0 {
			written, err := s.pty.Master.Write(b)
			if err != nil {
				if _, ok := err.(*os.PathError); !ok {
					exit(err)
				}
			}
			b = b[written:]
			bytesToWrite -= written
		}
	}
}

func (s *serialPortStruct) readLoop() {
	s.read = make(chan []byte)
	b := make([]byte, 1500)
	for {
		n, err := s.pty.Master.Read(b)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				exit(err)
			}
		}
		s.read <- b[:n]
	}
}

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

	go s.readLoop()
	go s.writeLoop()
}

func (s *serialPortStruct) deinit() {
	if s.pty != nil {
		s.pty.Close()
	}
}
