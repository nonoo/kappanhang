// +build linux

package main

import (
	"os"

	"github.com/google/goterm/term"
)

type serialPortStruct struct {
	pty     *term.PTY
	symlink string

	writeLoopDeinitNeededChan   chan bool
	writeLoopDeinitFinishedChan chan bool
	readLoopDeinitNeededChan    chan bool
	readLoopDeinitFinishedChan  chan bool

	// Read from this channel to receive serial data.
	read chan []byte
	// Write to this channel to send serial data.
	write chan []byte
}

var serialPort serialPortStruct

func (s *serialPortStruct) writeLoop() {
	var b []byte
	for {
		select {
		case b = <-s.write:
		case <-s.writeLoopDeinitNeededChan:
			s.writeLoopDeinitFinishedChan <- true
			return
		}

		for len(b) > 0 {
			written, err := s.pty.Master.Write(b)
			if err != nil {
				if _, ok := err.(*os.PathError); !ok {
					reportError(err)
				}
			}
			b = b[written:]
		}
	}
}

func (s *serialPortStruct) readLoop() {
	for {
		b := make([]byte, maxSerialFrameLength)
		n, err := s.pty.Master.Read(b)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				reportError(err)
			}
		}

		select {
		case s.read <- b[:n]:
		case <-s.readLoopDeinitNeededChan:
			s.readLoopDeinitFinishedChan <- true
			return
		}
	}
}

// We only init the virtual serial port once, with the first device name we acquire, so apps using the
// virtual serial port won't have issues with the interface going down while the app is running.
func (s *serialPortStruct) initIfNeeded(devName string) (err error) {
	if s.pty != nil {
		// Depleting channel which may contain data while the serial connection to the server was offline.
		for {
			select {
			case <-s.read:
			default:
				return
			}
		}
	}

	s.pty, err = term.OpenPTY()
	if err != nil {
		return err
	}

	var t term.Termios
	t.Raw()
	err = t.Set(s.pty.Master)
	if err != nil {
		return err
	}
	err = t.Set(s.pty.Slave)
	if err != nil {
		return err
	}

	n, err := s.pty.PTSName()
	if err != nil {
		return err
	}
	s.symlink = "/tmp/kappanhang-" + devName + ".pty"
	_ = os.Remove(s.symlink)
	if err := os.Symlink(n, s.symlink); err != nil {
		return err
	}
	log.Print("opened ", n, " as ", s.symlink)

	s.write = make(chan []byte)
	s.read = make(chan []byte)

	s.readLoopDeinitNeededChan = make(chan bool)
	s.readLoopDeinitFinishedChan = make(chan bool)
	go s.readLoop()
	s.writeLoopDeinitNeededChan = make(chan bool)
	s.writeLoopDeinitFinishedChan = make(chan bool)
	go s.writeLoop()
	return nil
}

func (s *serialPortStruct) deinit() {
	if s.pty != nil {
		s.pty.Close()
	}
	_ = os.Remove(s.symlink)
	if s.readLoopDeinitNeededChan != nil {
		s.readLoopDeinitNeededChan <- true
		<-s.readLoopDeinitFinishedChan
	}
	if s.writeLoopDeinitNeededChan != nil {
		s.writeLoopDeinitNeededChan <- true
		<-s.writeLoopDeinitFinishedChan
	}
}
