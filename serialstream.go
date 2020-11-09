package main

import (
	"bytes"
	"encoding/binary"
	"time"
)

const maxSerialFrameLength = 80 // Max. frame length according to Hamlib.
const serialRxSeqBufLength = 200 * time.Millisecond

type serialStream struct {
	common streamCommon

	sendSeq uint16

	rxSeqBuf          seqBuf
	rxSeqBufEntryChan chan seqBufEntry

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
	l := byte(len(d))
	p := append([]byte{0x15 + l, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0xc1, l, 0x00, byte(s.sendSeq >> 8), byte(s.sendSeq)}, d...)
	if err := s.common.pkt0.sendTrackedPacket(&s.common, p); err != nil {
		return err
	}
	s.sendSeq++
	return nil
}

func (s *serialStream) sendOpenClose(close bool) error {
	var magic byte
	if close {
		magic = 0x00
	} else {
		magic = 0x05
	}

	p := []byte{0x16, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0xc0, 0x01, 0x00, byte(s.sendSeq >> 8), byte(s.sendSeq), magic}
	if err := s.common.pkt0.sendTrackedPacket(&s.common, p); err != nil {
		return err
	}
	s.sendSeq++
	return nil
}

func (s *serialStream) handleRxSeqBufEntry(e seqBufEntry) {
	gotSeq := uint16(e.seq)
	if s.receivedSerialData {
		// Out of order packets can happen if we receive a retransmitted packet, but too late.
		if s.rxSeqBuf.leftOrRightCloserToSeq(e.seq, seqNum(s.lastReceivedSeq)) != left {
			log.Debug("got out of order pkt seq #", e.seq)
			return
		}

		expectedSeq := s.lastReceivedSeq + 1
		if expectedSeq != gotSeq {
			var missingPkts int
			if gotSeq > expectedSeq {
				missingPkts = int(gotSeq) - int(expectedSeq)
			} else {
				missingPkts = int(gotSeq) + 65536 - int(expectedSeq)
			}
			netstat.reportLoss(missingPkts)
			log.Error("lost ", missingPkts, " packets")
		}
	}
	s.lastReceivedSeq = gotSeq
	s.receivedSerialData = true

	if s.common.pkt0.isPkt0(e.data) {
		return
	}

	e.data = e.data[21:]

	if !civControl.decode(e.data) {
		return
	}

	if serialPort.write != nil {
		serialPort.write <- e.data
	}
	if serialTCPSrv.isClientConnected() {
		serialTCPSrv.toClient <- e.data
	}
}

func (s *serialStream) handleSerialPacket(r []byte) error {
	gotSeq := binary.LittleEndian.Uint16(r[6:8])
	addedToFront, _ := s.rxSeqBuf.add(seqNum(gotSeq), r)

	// If the packet is not added to the front of the seqbuf, then it means that it was an answer for a
	// retransmit request (or it was an out of order packet which we don't want start a retransmit).
	if !addedToFront {
		return nil
	}

	return s.common.requestRetransmitIfNeeded(gotSeq)
}

func (s *serialStream) handleRead(r []byte) error {
	// We add both idle pkt0 and serial data to the seqbuf.
	if s.common.pkt0.isIdlePkt0(r) || (len(r) >= 22 && r[16] == 0xc1 && r[0]-0x15 == r[17]) {
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
	if enableSerialDevice {
		for {
			select {
			case r := <-serialPort.read:
				s.gotDataForRadio(r)

			case r := <-s.common.readChan:
				if err := s.handleRead(r); err != nil {
					reportError(err)
				}
			case e := <-s.rxSeqBufEntryChan:
				s.handleRxSeqBufEntry(e)
			case r := <-serialTCPSrv.fromClient:
				s.gotDataForRadio(r)
			case <-s.readFromSerialPort.frameTimeout.C:
				s.readFromSerialPort.buf.Reset()
				s.readFromSerialPort.frameStarted = false
			case <-s.deinitNeededChan:
				s.deinitFinishedChan <- true
				return
			}
		}
	} else {
		for {
			select {
			case r := <-s.common.readChan:
				if err := s.handleRead(r); err != nil {
					reportError(err)
				}
			case e := <-s.rxSeqBufEntryChan:
				s.handleRxSeqBufEntry(e)
			case r := <-serialTCPSrv.fromClient:
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
}

func (s *serialStream) init(devName string) error {
	if err := s.common.init("serial", serialStreamPort); err != nil {
		return err
	}

	if enableSerialDevice {
		if err := serialPort.initIfNeeded(devName); err != nil {
			return err
		}
	}
	if err := serialTCPSrv.initIfNeeded(); err != nil {
		return err
	}

	if err := s.common.start(); err != nil {
		return err
	}

	s.common.pkt7.startPeriodicSend(&s.common, 1, false)
	s.common.pkt0.init(&s.common)
	s.common.pkt0.startPeriodicSend(&s.common)

	if err := s.sendOpenClose(false); err != nil {
		return err
	}

	log.Print("stream started")

	s.rxSeqBufEntryChan = make(chan seqBufEntry)
	s.rxSeqBuf.init(serialRxSeqBufLength, 0xffff, 0, s.rxSeqBufEntryChan)

	s.deinitNeededChan = make(chan bool)
	s.deinitFinishedChan = make(chan bool)

	s.readFromSerialPort.frameTimeout = time.NewTimer(0)
	<-s.readFromSerialPort.frameTimeout.C

	civControl.deinit()
	civControl = civControlStruct{}
	if err := civControl.init(s); err != nil {
		return err
	}

	go s.loop()
	return nil
}

func (s *serialStream) deinit() {
	if s.common.conn != nil {
		_ = s.sendOpenClose(true)
	}

	if s.deinitNeededChan != nil {
		s.deinitNeededChan <- true
		<-s.deinitFinishedChan
	}
	civControl.deinit()
	s.common.deinit()
	s.rxSeqBuf.deinit()
}
