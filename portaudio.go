package main

import (
	"encoding/binary"

	"github.com/nonoo/kappanhang/log"
)

type portAudio struct {
	port portCommon
}

func (p *portAudio) StartStream() {
	p.port.open(50003)

	p.port.sendPkt3()

	// Expecting a Pkt4 answer.
	// Example answer from radio: 0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x8c, 0x7d, 0x45, 0x7a, 0x1d, 0xf6, 0xe9, 0x0b
	r := p.port.expect(16, []byte{0x10, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00})
	p.port.remoteSID = binary.BigEndian.Uint32(r[8:12])
	p.port.sendPkt6()

	log.Debugf("got remote session id %.8x", p.port.remoteSID)
}
