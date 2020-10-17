package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/nonoo/kappanhang/log"
)

type PortAudio struct {
	conn      *net.UDPConn
	sendSeq   uint16
	localSID  uint32
	remoteSID uint32
}

func (p *PortAudio) send(d []byte) {
	_, err := p.conn.Write(d)
	if err != nil {
		log.Fatal(err)
	}
}

func (p *PortAudio) read() ([]byte, error) {
	err := p.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		log.Fatal(err)
	}

	b := make([]byte, 1500)
	n, _, err := p.conn.ReadFromUDP(b)
	if err != nil {
		if err, ok := err.(net.Error); ok && !err.Timeout() {
			log.Fatal(err)
		}
	}
	return b[:n], err
}

func (p *PortAudio) expect(packetLength int, b []byte) []byte {
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

func (p *PortAudio) sendAudioPkt3() {
	p.send([]byte{0x10, 0x00, 0x00, 0x00, 0x03, 0x00, byte(p.sendSeq), byte(p.sendSeq >> 8),
		byte(p.localSID >> 24), byte(p.localSID >> 16), byte(p.localSID >> 8), byte(p.localSID),
		byte(p.remoteSID >> 24), byte(p.remoteSID >> 16), byte(p.remoteSID >> 8), byte(p.remoteSID)})
}

func (p *PortAudio) sendAudioPkt6() {
	p.send([]byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00,
		byte(p.localSID >> 24), byte(p.localSID >> 16), byte(p.localSID >> 8), byte(p.localSID),
		byte(p.remoteSID >> 24), byte(p.remoteSID >> 16), byte(p.remoteSID >> 8), byte(p.remoteSID)})
}

func (p *PortAudio) StartStream() {
	hostPort := fmt.Sprint(connectAddress, ":50003")
	log.Print("connecting to ", hostPort)
	raddr, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		log.Fatal(err)
	}
	laddr := net.UDPAddr{
		Port: 50003,
	}
	p.conn, err = net.DialUDP("udp", &laddr, raddr)
	if err != nil {
		log.Fatal(err)
	}

	p.localSID = uint32(time.Now().Unix())
	log.Debugf("using local session id %.8x", p.localSID)

	p.sendAudioPkt3()

	// Expecting a Pkt4 answer.
	// Example answer from radio: 0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x8c, 0x7d, 0x45, 0x7a, 0x1d, 0xf6, 0xe9, 0x0b
	r := p.expect(16, []byte{0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00})
	p.remoteSID = binary.BigEndian.Uint32(r[8:12])
	p.sendAudioPkt6()

	log.Debugf("got remote session id %.8x", p.remoteSID)
}
