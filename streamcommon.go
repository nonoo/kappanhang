package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/nonoo/kappanhang/log"
)

const expectTimeoutDuration = time.Second

type streamCommon struct {
	name         string
	conn         *net.UDPConn
	localSID     uint32
	remoteSID    uint32
	gotRemoteSID bool
	readChan     chan []byte

	pkt7 pkt7Type
}

func (s *streamCommon) send(d []byte) {
	_, err := s.conn.Write(d)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *streamCommon) read() []byte {
	b := make([]byte, 1500)
	n, _, err := s.conn.ReadFromUDP(b)
	if err != nil {
		// Ignoring timeout errors.
		if err, ok := err.(net.Error); ok && !err.Timeout() {
			log.Fatal(err)
		}
	}
	return b[:n]
}

func (s *streamCommon) reader() {
	for {
		r := s.read()
		if s.pkt7.isPkt7(r) {
			s.pkt7.handle(s, r)
		}

		s.readChan <- r
	}
}

func (s *streamCommon) tryReceivePacket(timeout time.Duration, packetLength, matchStartByte int, b []byte) []byte {
	var r []byte
	expectStart := time.Now()
	for {
		err := s.conn.SetReadDeadline(time.Now().Add(timeout - time.Since(expectStart)))
		if err != nil {
			log.Fatal(err)
		}

		r = <-s.readChan

		err = s.conn.SetReadDeadline(time.Time{})
		if err != nil {
			log.Fatal(err)
		}

		if len(r) == packetLength && bytes.Equal(r[matchStartByte:len(b)+matchStartByte], b) {
			break
		}
		if time.Since(expectStart) > timeout {
			return nil
		}
	}
	return r
}

func (s *streamCommon) expect(packetLength int, b []byte) []byte {
	r := s.tryReceivePacket(expectTimeoutDuration, packetLength, 0, b)
	if r == nil {
		log.Fatal(s.name + "/expect timeout")
	}
	return r
}

func (s *streamCommon) open(name string, portNumber int) {
	s.name = name
	hostPort := fmt.Sprint(connectAddress, ":", portNumber)
	log.Print(s.name+"/connecting to ", hostPort)
	raddr, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		log.Fatal(err)
	}

	// Use the same local and remote port. The radio does not handle different ports well.
	l := net.UDPAddr{
		Port: portNumber,
	}
	s.conn, err = net.DialUDP("udp", &l, raddr)
	if err != nil {
		log.Fatal(err)
	}

	// Constructing the local session ID by combining the local IP address and port.
	laddr := s.conn.LocalAddr().(*net.UDPAddr)
	s.localSID = binary.BigEndian.Uint32(laddr.IP[len(laddr.IP)-4:])<<16 | uint32(laddr.Port&0xffff)
	log.Debugf(s.name+"/using session id %.8x", s.localSID)

	_, err = rand.Read(s.pkt7.randIDBytes[:])
	if err != nil {
		log.Fatal(err)
	}

	s.readChan = make(chan []byte)
	go s.reader()

	if r := s.pkt7.tryReceive(300*time.Millisecond, s); s.pkt7.isPkt7(r) {
		s.remoteSID = binary.BigEndian.Uint32(r[8:12])
		s.gotRemoteSID = true
		log.Print(s.name + "/closing running stream")
		s.sendDisconnect()
		time.Sleep(time.Second)
		s.gotRemoteSID = false
	}
}

func (s *streamCommon) sendPkt3() {
	s.send([]byte{0x10, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)})
}

func (s *streamCommon) waitForPkt4Answer() {
	log.Debug(s.name + "/expecting a pkt4 answer")
	// Example answer from radio: 0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x8c, 0x7d, 0x45, 0x7a, 0x1d, 0xf6, 0xe9, 0x0b
	r := s.expect(16, []byte{0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00})
	s.remoteSID = binary.BigEndian.Uint32(r[8:12])
	s.gotRemoteSID = true

	log.Debugf(s.name+"/got remote session id %.8x", s.remoteSID)
}

func (s *streamCommon) sendPkt6() {
	s.send([]byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00,
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)})
}

func (s *streamCommon) waitForPkt6Answer() {
	log.Debug(s.name + "/expecting pkt6 answer")
	// Example answer from radio: 0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00, 0xe8, 0xd0, 0x44, 0x50, 0xa0, 0x61, 0x39, 0xbe
	s.expect(16, []byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00})
}

func (s *streamCommon) sendDisconnect() {
	if !s.gotRemoteSID {
		return
	}

	s.send([]byte{0x10, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00,
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)})
}
