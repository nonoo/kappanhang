package main

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/nonoo/kappanhang/log"
)

type streamCommon struct {
	name      string
	conn      *net.UDPConn
	localSID  uint32
	remoteSID uint32
	sendSeq   uint16
}

func (s *streamCommon) send(d []byte) {
	_, err := s.conn.Write(d)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *streamCommon) read() ([]byte, error) {
	err := s.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		log.Fatal(err)
	}

	b := make([]byte, 1500)
	n, _, err := s.conn.ReadFromUDP(b)
	if err != nil {
		log.Fatal(err)
	}
	return b[:n], err
}

func (s *streamCommon) reader(c chan []byte) {
	var errCount int
	for {
		r, err := s.read()
		if err == nil {
			c <- r
		} else {
			errCount++
			if errCount > 5 {
				log.Fatal(s.name + "/timeout")
			}
			log.Error(s.name + "/stream break detected")
		}
		errCount = 0
	}
}

func (s *streamCommon) expect(packetLength int, b []byte) []byte {
	var r []byte
	expectStart := time.Now()
	for {
		r, _ = s.read()
		if len(r) == packetLength && bytes.Equal(r[:len(b)], b) {
			break
		}
		if time.Since(expectStart) > time.Second {
			log.Fatal("expect timeout")
		}
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
	laddr := net.UDPAddr{
		Port: portNumber,
	}
	s.conn, err = net.DialUDP("udp", &laddr, raddr)
	if err != nil {
		log.Fatal(err)
	}

	s.localSID = uint32(time.Now().Unix())
	log.Debugf(s.name+"/using session id %.8x", s.localSID)
}

func (p *streamCommon) sendPkt3() {
	p.send([]byte{0x10, 0x00, 0x00, 0x00, 0x03, 0x00, byte(p.sendSeq), byte(p.sendSeq >> 8),
		byte(p.localSID >> 24), byte(p.localSID >> 16), byte(p.localSID >> 8), byte(p.localSID),
		byte(p.remoteSID >> 24), byte(p.remoteSID >> 16), byte(p.remoteSID >> 8), byte(p.remoteSID)})
}

func (p *streamCommon) sendPkt6() {
	p.send([]byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00,
		byte(p.localSID >> 24), byte(p.localSID >> 16), byte(p.localSID >> 8), byte(p.localSID),
		byte(p.remoteSID >> 24), byte(p.remoteSID >> 16), byte(p.remoteSID >> 8), byte(p.remoteSID)})
}
