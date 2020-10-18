package main

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/nonoo/kappanhang/log"
)

type streamConnection struct {
	conn      *net.UDPConn
	localSID  uint32
	remoteSID uint32
	sendSeq   uint16
}

func (p *streamConnection) send(d []byte) {
	_, err := p.conn.Write(d)
	if err != nil {
		log.Fatal(err)
	}
}

func (p *streamConnection) read() ([]byte, error) {
	err := p.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		log.Fatal(err)
	}

	b := make([]byte, 1500)
	n, _, err := p.conn.ReadFromUDP(b)
	if err != nil {
		log.Fatal(err)
	}
	return b[:n], err
}

func (p *streamConnection) reader(c chan []byte) {
	var errCount int
	for {
		r, err := p.read()
		if err == nil {
			c <- r
		} else {
			errCount++
			if errCount > 5 {
				log.Fatal("timeout")
			}
			log.Error("stream break detected")
		}
		errCount = 0
	}
}

func (p *streamConnection) expect(packetLength int, b []byte) []byte {
	var r []byte
	expectStart := time.Now()
	for {
		r, _ = p.read()
		if len(r) == packetLength && bytes.Equal(r[:len(b)], b) {
			break
		}
		if time.Since(expectStart) > time.Second {
			log.Fatal("expect timeout")
		}
	}
	return r
}

func (p *streamConnection) open(portNumber int) {
	hostPort := fmt.Sprint(connectAddress, ":", portNumber)
	log.Print("connecting to ", hostPort)
	raddr, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		log.Fatal(err)
	}
	laddr := net.UDPAddr{
		Port: portNumber,
	}
	p.conn, err = net.DialUDP("udp", &laddr, raddr)
	if err != nil {
		log.Fatal(err)
	}

	p.localSID = uint32(time.Now().Unix())
	log.Debugf("using session id %.8x", p.localSID)
}

func (p *streamConnection) sendPkt3() {
	p.send([]byte{0x10, 0x00, 0x00, 0x00, 0x03, 0x00, byte(p.sendSeq), byte(p.sendSeq >> 8),
		byte(p.localSID >> 24), byte(p.localSID >> 16), byte(p.localSID >> 8), byte(p.localSID),
		byte(p.remoteSID >> 24), byte(p.remoteSID >> 16), byte(p.remoteSID >> 8), byte(p.remoteSID)})
}

func (p *streamConnection) sendPkt6() {
	p.send([]byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00,
		byte(p.localSID >> 24), byte(p.localSID >> 16), byte(p.localSID >> 8), byte(p.localSID),
		byte(p.remoteSID >> 24), byte(p.remoteSID >> 16), byte(p.remoteSID >> 8), byte(p.remoteSID)})
}
