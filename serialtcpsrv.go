package main

import (
	"fmt"
	"io"
	"net"
)

type serialTCPSrv struct {
	listener net.Listener
	client   net.Conn

	fromClient chan []byte
	toClient   chan []byte

	writeLoopDeinitNeededChan   chan bool
	writeLoopDeinitFinishedChan chan bool

	deinitFinishedChan chan bool
}

func (s *serialTCPSrv) writeLoop(errChan chan error) {
	s.toClient = make(chan []byte)
	defer func() {
		s.toClient = nil
	}()

	var b []byte
	for {
		select {
		case b = <-s.toClient:
		case <-s.writeLoopDeinitNeededChan:
			s.writeLoopDeinitFinishedChan <- true
			return
		}

		for len(b) > 0 {
			written, err := s.client.Write(b)
			if err != nil {
				errChan <- err
			}
			b = b[written:]
		}
	}
}

func (s *serialTCPSrv) disconnectClient() {
	if s.client != nil {
		s.client.Close()
	}
	if s.writeLoopDeinitNeededChan != nil {
		s.writeLoopDeinitNeededChan <- true
		<-s.writeLoopDeinitFinishedChan

		s.writeLoopDeinitNeededChan = nil
		s.writeLoopDeinitFinishedChan = nil
	}
}

func (s *serialTCPSrv) loop() {
	for {
		var err error
		s.client, err = s.listener.Accept()

		if err != nil {
			if err != io.EOF {
				reportError(err)
			}
			s.listener.Close()
			s.disconnectClient()
			s.deinitFinishedChan <- true
			return
		}

		log.Print("client ", s.client.RemoteAddr().String(), " connected")

		s.writeLoopDeinitNeededChan = make(chan bool)
		s.writeLoopDeinitFinishedChan = make(chan bool)
		writeErrChan := make(chan error)
		go s.writeLoop(writeErrChan)

		connected := true
		for connected {
			b := make([]byte, maxSerialFrameLength)
			n, err := s.client.Read(b)
			if err != nil {
				break
			}

			select {
			case s.fromClient <- b[:n]:
			case <-writeErrChan:
				connected = false
			}
		}

		s.client.Close()
		log.Print("client ", s.client.RemoteAddr().String(), " disconnected")
	}
}

func (s *serialTCPSrv) start() (err error) {
	s.listener, err = net.Listen("tcp", fmt.Sprint(":", serialTCPPort))
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Print("exposing serial port on tcp port ", serialTCPPort)

	s.fromClient = make(chan []byte)

	s.deinitFinishedChan = make(chan bool)
	go s.loop()
	return
}

func (s *serialTCPSrv) stop() {
	if s.listener != nil {
		s.listener.Close()
	}

	s.disconnectClient()
	if s.fromClient != nil {
		close(s.fromClient)
	}
	if s.deinitFinishedChan != nil {
		<-s.deinitFinishedChan
	}
}
