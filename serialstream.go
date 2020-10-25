package main

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/nonoo/kappanhang/log"
)

const maxSerialFrameLength = 80 // Max. frame length according to Hamlib.
const serialRxSeqBufLength = 100 * time.Millisecond

type serialStream struct {
	common streamCommon

	serialPort serialPortStruct
	tcpsrv     serialTCPSrv

	sendSeq uint16

	lastSeqBufFrontRxSeq uint16
	rxSeqBuf             seqBuf
	rxSeqBufEntryChan    chan seqBufEntry

	receivedSerialData bool
	lastReceivedSeq    uint16

	readFromSerialPort struct {
		buf          bytes.Buffer
		frameStarted bool
		frameTimeout *time.Timer
	}

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool
}

func (s *serialStream) send(d []byte) error {
	s.common.pkt0.sendSeqLock()
	defer s.common.pkt0.sendSeqUnlock()

	l := byte(len(d))
	p := append([]byte{0x15 + l, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.common.pkt0.sendSeq), byte(s.common.pkt0.sendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0xc1, l, 0x00, byte(s.sendSeq >> 8), byte(s.sendSeq)}, d...)
	if err := s.common.send(p); err != nil {
		return err
	}
	s.common.pkt0.sendSeq++
	s.sendSeq++
	return nil
}

func (s *serialStream) sendOpenClose(close bool) error {
	s.common.pkt0.sendSeqLock()
	defer s.common.pkt0.sendSeqUnlock()

	var magic byte
	if close {
		magic = 0x00
	} else {
		magic = 0x05
	}

	p := []byte{0x16, 0x00, 0x00, 0x00, 0x00, 0x00, byte(s.common.pkt0.sendSeq), byte(s.common.pkt0.sendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0xc0, 0x01, 0x00, byte(s.sendSeq >> 8), byte(s.sendSeq), magic}
	if err := s.common.send(p); err != nil {
		return err
	}
	s.common.pkt0.sendSeq++
	s.sendSeq++
	return nil
}

func (s *serialStream) handleRxSeqBufEntry(e seqBufEntry) {
	gotSeq := uint16(e.seq)
	if s.receivedSerialData {
		expectedSeq := s.lastReceivedSeq + 1
		if expectedSeq != gotSeq {
			var missingPkts int
			if gotSeq > expectedSeq {
				missingPkts = int(gotSeq) - int(expectedSeq)
			} else {
				missingPkts = int(gotSeq) + 65536 - int(expectedSeq)
			}
			log.Error("lost ", missingPkts, " packets")
		}
	}
	s.lastReceivedSeq = gotSeq
	s.receivedSerialData = true

	if len(e.data) == 16 { // Do not send pkt0s.
		return
	}

	e.data = e.data[21:]

	s.serialPort.write <- e.data
	if s.tcpsrv.toClient != nil {
		s.tcpsrv.toClient <- e.data
	}
}

func (s *serialStream) requestRetransmitIfNeeded(gotSeq uint16) error {
	prevExpectedSeq := gotSeq - 1
	if s.lastSeqBufFrontRxSeq != prevExpectedSeq {
		var missingPkts int
		var sr seqNumRange
		if prevExpectedSeq > s.lastSeqBufFrontRxSeq {
			sr[0] = s.lastSeqBufFrontRxSeq
			sr[1] = prevExpectedSeq
			missingPkts = int(prevExpectedSeq) - int(s.lastSeqBufFrontRxSeq)
		} else {
			sr[0] = prevExpectedSeq
			sr[1] = s.lastSeqBufFrontRxSeq
			missingPkts = int(prevExpectedSeq) + 65536 - int(s.lastSeqBufFrontRxSeq)
		}
		if missingPkts == 1 {
			log.Debug("request pkt #", sr[1], " retransmit")
			if err := s.common.sendRetransmitRequest(sr[1]); err != nil {
				return err
			}
		} else if missingPkts < 50 {
			log.Debug("request pkt #", sr[0], "-#", sr[1], " retransmit")
			if err := s.common.sendRetransmitRequestForRanges([]seqNumRange{sr}); err != nil {
				return err
			}
		}
	}
	s.lastSeqBufFrontRxSeq = gotSeq
	return nil
}

func (s *serialStream) handleSerialPacket(r []byte) error {
	gotSeq := binary.LittleEndian.Uint16(r[6:8])
	addedToFront, _ := s.rxSeqBuf.add(seqNum(gotSeq), r)

	// If the packet is not added to the front of the seqbuf, then it means that it was an answer for a
	// retransmit request (or it was an out of order packet which we don't want start a retransmit).
	if !addedToFront {
		return nil
	}

	return s.requestRetransmitIfNeeded(gotSeq)
}

func (s *serialStream) handleRead(r []byte) error {
	// We add both serial data and pkt0 to the seqbuf.
	if (len(r) == 16 && bytes.Equal(r[:6], []byte{0x10, 0x00, 0x00, 0x00, 0x00, 0x00})) || // Pkt0?
		(len(r) >= 22 && r[16] == 0xc1 && r[0]-0x15 == r[17]) { // Serial data?
		return s.handleSerialPacket(r)
	}
	return nil
}

func (s *serialStream) gotDataForRadio(r []byte) {
	for len(r) > 0 && !s.readFromSerialPort.frameStarted {
		if s.readFromSerialPort.buf.Len() > 1 {
			s.readFromSerialPort.buf.Reset()
		}
		if s.readFromSerialPort.buf.Len() == 0 {
			// Cut until we find the frame start byte.
			for r[0] != 0xfe {
				r = r[1:]
				if len(r) == 0 {
					return
				}
			}
			// Found the first start byte.
			s.readFromSerialPort.buf.WriteByte(r[0])
			r = r[1:]
		}
		if len(r) > 0 && s.readFromSerialPort.buf.Len() == 1 {
			if r[0] != 0xfe {
				s.readFromSerialPort.buf.Reset()
				r = r[1:]
			} else {
				// Found the second start byte.
				s.readFromSerialPort.buf.WriteByte(r[0])
				r = r[1:]
				s.readFromSerialPort.frameTimeout.Reset(100 * time.Millisecond)
				s.readFromSerialPort.frameStarted = true
			}
		}
	}

	for _, b := range r {
		s.readFromSerialPort.buf.WriteByte(b)
		if b == 0xfc || b == 0xfd || s.readFromSerialPort.buf.Len() == maxSerialFrameLength {
			if err := s.send(s.readFromSerialPort.buf.Bytes()); err != nil {
				reportError(err)
			}
			if !s.readFromSerialPort.frameTimeout.Stop() {
				<-s.readFromSerialPort.frameTimeout.C
			}
			s.readFromSerialPort.buf.Reset()
			s.readFromSerialPort.frameStarted = false
			break
		}
	}
}

func (s *serialStream) loop() {
	for {
		select {
		case r := <-s.common.readChan:
			if err := s.handleRead(r); err != nil {
				reportError(err)
			}
		case e := <-s.rxSeqBufEntryChan:
			s.handleRxSeqBufEntry(e)
		case r := <-s.serialPort.read:
			s.gotDataForRadio(r)
		case r := <-s.tcpsrv.fromClient:
			s.gotDataForRadio(r)
		case <-s.readFromSerialPort.frameTimeout.C:
			s.readFromSerialPort.buf.Reset()
			s.readFromSerialPort.frameStarted = false
		case <-s.deinitNeededChan:
			s.deinitFinishedChan <- true
			return
		}
	}
}

func (s *serialStream) start(devName string) error {
	if err := s.serialPort.init(devName); err != nil {
		return err
	}

	if err := s.common.sendPkt3(); err != nil {
		return err
	}
	if err := s.common.waitForPkt4Answer(); err != nil {
		return err
	}
	if err := s.common.sendPkt6(); err != nil {
		return err
	}
	if err := s.common.waitForPkt6Answer(); err != nil {
		return err
	}

	log.Print("stream started")

	s.common.pkt7.startPeriodicSend(&s.common, 1, false)
	s.common.pkt0.startPeriodicSend(&s.common)

	if err := s.sendOpenClose(false); err != nil {
		return err
	}

	if err := s.tcpsrv.start(); err != nil {
		return err
	}

	s.deinitNeededChan = make(chan bool)
	s.deinitFinishedChan = make(chan bool)

	s.readFromSerialPort.frameTimeout = time.NewTimer(0)
	<-s.readFromSerialPort.frameTimeout.C

	go s.loop()
	return nil
}

func (s *serialStream) init() error {
	if err := s.common.init("serial", 50002); err != nil {
		return err
	}
	s.rxSeqBufEntryChan = make(chan seqBufEntry)
	s.rxSeqBuf.init(serialRxSeqBufLength, 0xffff, 0, s.rxSeqBufEntryChan)
	return nil
}

func (s *serialStream) deinit() {
	if s.common.conn != nil {
		_ = s.sendOpenClose(true)
	}

	s.tcpsrv.stop()
	s.serialPort.deinit()

	if s.deinitNeededChan != nil {
		s.deinitNeededChan <- true
		<-s.deinitFinishedChan
	}
	s.common.deinit()
	s.rxSeqBuf.deinit()
}
