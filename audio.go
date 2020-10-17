package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/nonoo/kappanhang/log"
)

var audioConn *net.UDPConn
var audioSendSeq uint16
var localAudioSID uint32
var remoteAudioSID uint32

func audioStreamSend(d []byte) {
	_, err := audioConn.Write(d)
	if err != nil {
		log.Fatal(err)
	}
}

func readAudioStream() ([]byte, error) {
	err := audioConn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		log.Fatal(err)
	}

	b := make([]byte, 1500)
	n, _, err := audioConn.ReadFromUDP(b)
	if err != nil {
		if err, ok := err.(net.Error); ok && !err.Timeout() {
			log.Fatal(err)
		}
	}
	return b[:n], err
}

func sendAudioPkt3() {
	audioStreamSend([]byte{0x10, 0x00, 0x00, 0x00, 0x03, 0x00, byte(audioSendSeq), byte(audioSendSeq >> 8),
		byte(localAudioSID >> 24), byte(localAudioSID >> 16), byte(localAudioSID >> 8), byte(localAudioSID),
		byte(remoteAudioSID >> 24), byte(remoteAudioSID >> 16), byte(remoteAudioSID >> 8), byte(remoteAudioSID)})
}

func sendAudioPkt6() {
	audioStreamSend([]byte{0x10, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00,
		byte(localSID >> 24), byte(localSID >> 16), byte(localSID >> 8), byte(localSID),
		byte(remoteSID >> 24), byte(remoteSID >> 16), byte(remoteSID >> 8), byte(remoteSID)})
}

func openAudioStream() {
	hostPort := fmt.Sprint(connectAddress, ":50003")
	log.Print("connecting to ", hostPort)
	raddr, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		log.Fatal(err)
	}
	laddr := net.UDPAddr{
		Port: 50003,
	}
	audioConn, err = net.DialUDP("udp", &laddr, raddr)
	if err != nil {
		log.Fatal(err)
	}

	localAudioSID = uint32(time.Now().Unix())
	log.Debugf("using audio session id %.8x", localAudioSID)

	sendAudioPkt3()
	for {
		// Expecting a Pkt4 answer.
		// Example answer from radio: 0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x8c, 0x7d, 0x45, 0x7a, 0x1d, 0xf6, 0xe9, 0x0b
		r, _ := readAudioStream()
		if len(r) == 16 && bytes.Equal(r[:8], []byte{0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00}) {
			remoteSID = binary.BigEndian.Uint32(r[8:12])
			break
		}
	}
	sendAudioPkt6()
}
