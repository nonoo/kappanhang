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

type streamCommon struct {
	name      string
	conn      *net.UDPConn
	localSID  uint32
	remoteSID uint32
	readChan  chan []byte

	pkt7 struct {
		sendSeq    uint16
		randIDByte [1]byte
	}
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

func (s *streamCommon) reader() {
	var errCount int
	for {
		r, err := s.read()
		if err == nil {
			s.readChan <- r
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
		r = <-s.readChan
		if len(r) == packetLength && bytes.Equal(r[:len(b)], b) {
			break
		}
		if time.Since(expectStart) > time.Second {
			log.Fatal(s.name + "/expect timeout")
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
	s.conn, err = net.DialUDP("udp", nil, raddr)
	if err != nil {
		log.Fatal(err)
	}

	// Constructing the local session ID by combining the local IP address and port.
	laddr := s.conn.LocalAddr().(*net.UDPAddr)
	s.localSID = binary.BigEndian.Uint32(laddr.IP[len(laddr.IP)-4:])<<16 | uint32(laddr.Port&0xffff)
	log.Debugf(s.name+"/using session id %.8x", s.localSID)

	_, err = rand.Read(s.pkt7.randIDByte[:])
	if err != nil {
		log.Fatal(err)
	}

	s.readChan = make(chan []byte)
	go s.reader()
}

func (s *streamCommon) sendPkt3() {
	s.send([]byte{0x10, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)})
}

func (s *streamCommon) sendPkt6() {
	s.send([]byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00,
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID)})
}

func (s *streamCommon) sendPkt7Do(replyID []byte, seq uint16) {
	// Example request from PC:  0x15, 0x00, 0x00, 0x00, 0x07, 0x00, 0x09, 0x00, 0xbe, 0xd9, 0xf2, 0x63, 0xe4, 0x35, 0xdd, 0x72, 0x00, 0x78, 0x40, 0xf6, 0x02
	// Example reply from radio: 0x00, 0x00, 0x00, 0x00, 0x07, 0x00, 0x09, 0x00, 0xe4, 0x35, 0xdd, 0x72, 0xbe, 0xd9, 0xf2, 0x63, 0x01, 0x78, 0x40, 0xf6, 0x02
	var replyFlag byte
	if replyID == nil {
		replyID = make([]byte, 4)
		var randID [2]byte
		_, err := rand.Read(randID[:])
		if err != nil {
			log.Fatal(err)
		}
		replyID[0] = randID[0]
		replyID[1] = randID[1]
		replyID[2] = s.pkt7.randIDByte[0]
		replyID[3] = 0x03
	} else {
		replyFlag = 0x01
	}

	s.send([]byte{0x15, 0x00, 0x00, 0x00, 0x07, 0x00, byte(seq), byte(seq >> 8),
		byte(s.localSID >> 24), byte(s.localSID >> 16), byte(s.localSID >> 8), byte(s.localSID),
		byte(s.remoteSID >> 24), byte(s.remoteSID >> 16), byte(s.remoteSID >> 8), byte(s.remoteSID),
		replyFlag, replyID[0], replyID[1], replyID[2], replyID[3]})
}

func (s *streamCommon) sendPkt7() {
	s.sendPkt7Do(nil, s.pkt7.sendSeq)
	s.pkt7.sendSeq++
}

func (s *streamCommon) sendPkt7Reply(replyID []byte, seq uint16) {
	s.sendPkt7Do(replyID, seq)
}
