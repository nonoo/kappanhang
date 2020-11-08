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

	clientLoopDeinitNeededChan   chan bool
	clientLoopDeinitFinishedChan chan bool

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool

	clientConnected bool
	mutex           sync.Mutex
}

var serialTCPSrv serialTCPSrvStruct

func (s *serialTCPSrvStruct) isClientConnected() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.clientConnected
}

func (s *serialTCPSrvStruct) writeLoop(writeLoopDeinitNeededChan, writeLoopDeinitFinishedChan chan bool,
	errChan chan error) {

	var b []byte
	for {
		select {
		case b = <-s.toClient:
		case <-writeLoopDeinitNeededChan:
			writeLoopDeinitFinishedChan <- true
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
	if s.clientLoopDeinitNeededChan != nil {
		s.clientLoopDeinitNeededChan <- true
		<-s.clientLoopDeinitFinishedChan

		s.clientLoopDeinitNeededChan = nil
		s.clientLoopDeinitFinishedChan = nil
	}
}

func (s *serialTCPSrvStruct) clientLoop() {
	s.mutex.Lock()
	s.clientConnected = true
	s.mutex.Unlock()

	defer func() {
		s.mutex.Lock()
		s.clientConnected = false
		s.mutex.Unlock()
	}()

	log.Print("client ", s.client.RemoteAddr().String(), " connected")

	writeLoopDeinitNeededChan := make(chan bool)
	writeLoopDeinitFinishedChan := make(chan bool)
	writeErrChan := make(chan error)
	go s.writeLoop(writeLoopDeinitNeededChan, writeLoopDeinitFinishedChan, writeErrChan)

	defer func() {
		s.client.Close()
		log.Print("client ", s.client.RemoteAddr().String(), " disconnected")

		writeLoopDeinitNeededChan <- true
		<-writeLoopDeinitFinishedChan

		<-s.clientLoopDeinitNeededChan
		s.clientLoopDeinitFinishedChan <- true
	}()

	for {
		b := make([]byte, maxSerialFrameLength)
		n, err := s.client.Read(b)
		if err != nil {
			break
		}

		select {
		case s.fromClient <- b[:n]:
		case <-writeErrChan:
			return
		case <-s.clientLoopDeinitNeededChan:
			writeLoopDeinitNeededChan <- true
			<-writeLoopDeinitFinishedChan

			s.clientLoopDeinitFinishedChan <- true
			return
		}
	}
}

func (s *serialTCPSrvStruct) loop() {
	for {
		newClient, err := s.listener.Accept()

		s.disconnectClient()
		s.deinitClient()

		s.clientLoopDeinitNeededChan = make(chan bool)
		s.clientLoopDeinitFinishedChan = make(chan bool)

		if err != nil {
			if err != io.EOF {
				reportError(err)
			}
			<-s.deinitNeededChan
			s.deinitFinishedChan <- true
			return
		}

		s.client = newClient

		go s.clientLoop()
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
	if s.listener != nil {
		s.listener.Close()
	}

	if s.deinitNeededChan != nil {
		s.deinitNeededChan <- true
		<-s.deinitFinishedChan
	}
}
