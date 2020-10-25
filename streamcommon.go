package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/nonoo/kappanhang/log"
)

const expectTimeoutDuration = time.Second

type streamCommon struct {
	name                    string
	conn                    *net.UDPConn
	localSID                uint32
	remoteSID               uint32
	gotRemoteSID            bool
	readChan                chan []byte
	readerCloseNeededChan   chan bool
	readerCloseFinishedChan chan bool

	pkt0 pkt0Type
	pkt7 pkt7Type
}

func (s *streamCommon) send(d []byte) error {
	if _, err := s.conn.Write(d); err != nil {
		return err
	}
	return nil
}

func (s *streamCommon) read() ([]byte, error) {
	b := make([]byte, 1500)
	n, _, err := s.conn.ReadFromUDP(b)
	return b[:n], err
}

func (s *streamCommon) reader() {
	for {
		r, err := s.read()
		if err != nil {
			reportError(err)
		} else if s.pkt7.isPkt7(r) {
			if err := s.pkt7.handle(s, r); err != nil {
				reportError(err)
			}
			continue
		}

		select {
		case s.readChan <- r:
		case <-s.readerCloseNeededChan:
			s.readerCloseFinishedChan <- true
			return
		}
	}
}

func (s *streamCommon) tryReceivePacket(timeout time.Duration, packetLength, matchStartByte int, b []byte) []byte {
	var r []byte
	timer := time.NewTimer(timeout)
	for {
		select {
		case r = <-s.readChan:
		case <-timer.C:
			return nil
		}

		if len(r) == packetLength && bytes.Equal(r[matchStartByte:len(b)+matchStartByte], b) {
			break
		}
	}
	return r
}

func (s *streamCommon) expect(packetLength int, b []byte) ([]byte, error) {
	r := s.tryReceivePacket(expectTimeoutDuration, packetLength, 0, b)
	if r == nil {
		return nil, errors.New(s.name + "/expect timeout")
	}
	return r, nil
}

func (s *streamCommon) sendPkt3() error {
	p := []byte{0x10, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)}
	if err := s.send(p); err != nil {
		return err
	}
	if err := s.send(p); err != nil {
		return err
	}
	return nil
}

func (s *streamCommon) waitForPkt4Answer() error {
	log.Debug(s.name + "/expecting a pkt4 answer")
	// Example answer from radio: 0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x8c, 0x7d, 0x45, 0x7a, 0x1d, 0xf6, 0xe9, 0x0b
	r, err := s.expect(16, []byte{0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00})
	if err != nil {
		return err
	}
	s.remoteSID = binary.BigEndian.Uint32(r[8:12])
	s.gotRemoteSID = true
	return nil
}

func (s *streamCommon) sendPkt6() error {
	p := []byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00,
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)}
	if err := s.send(p); err != nil {
		return err
	}
	if err := s.send(p); err != nil {
		return err
	}
	return nil
}

func (s *streamCommon) waitForPkt6Answer() error {
	log.Debug(s.name + "/expecting pkt6 answer")
	// Example answer from radio: 0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00, 0xe8, 0xd0, 0x44, 0x50, 0xa0, 0x61, 0x39, 0xbe
	_, err := s.expect(16, []byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00})
	return err
}

func (s *streamCommon) sendDisconnect() error {
	log.Print(s.name + "/disconnecting")
	p := []byte{0x10, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00,
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)}
	if err := s.send(p); err != nil {
		return err
	}
	if err := s.send(p); err != nil {
		return err
	}
	return nil
}

func (s *streamCommon) init(name string, portNumber int) error {
	s.name = name
	hostPort := fmt.Sprint(connectAddress, ":", portNumber)
	log.Print(s.name+"/connecting to ", hostPort)
	raddr, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		return err
	}

	s.conn, err = net.DialUDP("udp", &net.UDPAddr{Port: portNumber}, raddr)
	if err != nil {
		return err
	}

	// Constructing the local session ID by combining the local IP address and port.
	laddr := s.conn.LocalAddr().(*net.UDPAddr)
	s.localSID = binary.BigEndian.Uint32(laddr.IP[len(laddr.IP)-4:])<<16 | uint32(laddr.Port&0xffff)

	s.readChan = make(chan []byte)
	s.readerCloseNeededChan = make(chan bool)
	s.readerCloseFinishedChan = make(chan bool)
	go s.reader()
	return nil
}

func (s *streamCommon) deinit() {
	s.pkt0.stopPeriodicSend()
	s.pkt7.stopPeriodicSend()
	if s.gotRemoteSID && s.conn != nil {
		_ = s.sendDisconnect()
	}
	s.conn.Close()

	if s.readerCloseNeededChan != nil {
		s.readerCloseNeededChan <- true
		<-s.readerCloseFinishedChan
	}
}
