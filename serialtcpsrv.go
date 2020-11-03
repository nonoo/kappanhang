package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type serialTCPSrvStruct struct {
	listener net.Listener
	client   net.Conn

	fromClient chan []byte
	toClient   chan []byte

	writeLoopDeinitNeededChan   chan bool
	writeLoopDeinitFinishedChan chan bool

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool

	deinitializing bool
	mutex          sync.Mutex
}

var serialTCPSrv serialTCPSrvStruct

func (s *serialTCPSrvStruct) isClientConnected() bool {
	return s.writeLoopDeinitNeededChan != nil
}

func (s *serialTCPSrvStruct) writeLoop(errChan chan error) {
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
				break
			}
			b = b[written:]
		}
	}
}

func (s *serialTCPSrvStruct) disconnectClient() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *serialTCPSrvStruct) deinitClient() {
	if s.writeLoopDeinitNeededChan != nil {
		s.writeLoopDeinitNeededChan <- true
		<-s.writeLoopDeinitFinishedChan

		s.writeLoopDeinitNeededChan = nil
		s.writeLoopDeinitFinishedChan = nil
	}
}

func (s *serialTCPSrvStruct) loop() {
	for {
		var err error
		s.client, err = s.listener.Accept()

		if err != nil {
			if err != io.EOF {
				reportError(err)
			}
			s.disconnectClient()
			s.deinitClient()
			<-s.deinitNeededChan
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
			case <-s.deinitNeededChan:
				s.disconnectClient()
				s.deinitClient()
				s.deinitFinishedChan <- true
				return
			}
		}

		s.disconnectClient()
		s.deinitClient()
		log.Print("client ", s.client.RemoteAddr().String(), " disconnected")

		s.mutex.Lock()
		if !s.deinitializing {
			rigctldRunner.restart()
		}
		s.mutex.Unlock()
	}
}

// We only init the serial port TCP server once, with the first device name we acquire, so apps using the
// serial port TCP server won't have issues with the interface going down while the app is running.
func (s *serialTCPSrvStruct) initIfNeeded() (err error) {
	if s.listener != nil {
		// Depleting channel which may contain data while the serial connection to the server was offline.
		for {
			select {
			case <-s.fromClient:
			default:
				return
			}
		}
	}

	s.listener, err = net.Listen("tcp", fmt.Sprint(":", serialTCPPort))
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Print("exposing serial port on tcp port ", serialTCPPort)

	s.fromClient = make(chan []byte)
	s.toClient = make(chan []byte)

	s.deinitNeededChan = make(chan bool)
	s.deinitFinishedChan = make(chan bool)
	go s.loop()
	return
}

func (s *serialTCPSrvStruct) deinit() {
	s.mutex.Lock()
	s.deinitializing = true
	s.mutex.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}

	s.disconnectClient()
	if s.deinitNeededChan != nil {
		s.deinitNeededChan <- true
		<-s.deinitFinishedChan
	}
}
