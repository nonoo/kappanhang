package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"github.com/nonoo/kappanhang/log"
)

const audioTimeoutDuration = 3 * time.Second
const rxSeqBufLength = 100 * time.Millisecond

type audioStream struct {
	common streamCommon

	timeoutTimer         *time.Timer
	receivedAudio        bool
	lastReceivedAudioSeq uint16
	rxSeqBuf             seqBuf
	rxSeqBufEntryChan    chan seqBufEntry

	audioSendSeq uint16
}

func (s *audioStream) sendDisconnect() {
	s.common.sendDisconnect()
}

// sendPart1 expects 1364 bytes of PCM data.
func (s *audioStream) sendPart1(pcmData []byte) {
	s.common.send(append([]byte{0x6c, 0x05, 0x00, 0x00, 0x00, 0x00, byte(s.audioSendSeq), byte(s.audioSendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0x80, 0x00, byte((s.audioSendSeq - 1) >> 8), byte(s.audioSendSeq - 1), 0x00, 0x00, byte(len(pcmData) >> 8), byte(len(pcmData))},
		pcmData...))
	s.audioSendSeq++
}

// sendPart2 expects 556 bytes of PCM data.
func (s *audioStream) sendPart2(pcmData []byte) {
	s.common.send(append([]byte{0x44, 0x02, 0x00, 0x00, 0x00, 0x00, byte(s.audioSendSeq), byte(s.audioSendSeq >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID),
		0x80, 0x00, byte((s.audioSendSeq - 1) >> 8), byte(s.audioSendSeq - 1), 0x00, 0x00, byte(len(pcmData) >> 8), byte(len(pcmData))},
		pcmData...))
	s.audioSendSeq++
}

func (s *audioStream) sendRetransmitRequest(seqNum uint16) {
	p := []byte{0x10, 0x00, 0x00, 0x00, 0x01, 0x00, byte(seqNum), byte(seqNum >> 8),
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID)}
	s.common.send(p)
	s.common.send(p)
}

type seqNumRange [2]uint16

func (s *audioStream) sendRetransmitRequestForRanges(seqNumRanges []seqNumRange) {
	seqNumBytes := make([]byte, len(seqNumRanges)*4)
	for i := 0; i < len(seqNumRanges); i++ {
		seqNumBytes[i*2] = byte(seqNumRanges[i][0])
		seqNumBytes[i*2+1] = byte(seqNumRanges[i][0] >> 8)
		seqNumBytes[i*2+2] = byte(seqNumRanges[i][1])
		seqNumBytes[i*2+3] = byte(seqNumRanges[i][1] >> 8)
	}
	p := append([]byte{0x18, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		byte(s.common.localSID >> 24), byte(s.common.localSID >> 16), byte(s.common.localSID >> 8), byte(s.common.localSID),
		byte(s.common.remoteSID >> 24), byte(s.common.remoteSID >> 16), byte(s.common.remoteSID >> 8), byte(s.common.remoteSID)},
		seqNumBytes...)
	s.common.send(p)
	s.common.send(p)
}

func (s *audioStream) handleRxSeqBufEntry(e seqBufEntry) {
	gotSeq := uint16(e.seq)
	if s.receivedAudio {
		expectedSeq := s.lastReceivedAudioSeq + 1
		if expectedSeq != gotSeq {
			var missingPkts int
			if gotSeq > expectedSeq {
				missingPkts = int(gotSeq) - int(expectedSeq)
			} else {
				missingPkts = int(gotSeq) + 65536 - int(expectedSeq)
			}
			log.Error("lost ", missingPkts, " audio packets")
		}
	}
	s.lastReceivedAudioSeq = gotSeq
	s.receivedAudio = true

	audio.play <- e.data
}

func (s *audioStream) handleAudioPacket(r []byte) {
	if s.timeoutTimer != nil {
		s.timeoutTimer.Stop()
		s.timeoutTimer.Reset(audioTimeoutDuration)
	}

	gotSeq := binary.LittleEndian.Uint16(r[6:8])
	err := s.rxSeqBuf.add(seqNum(gotSeq), r[24:])
	if err != nil {
		log.Error(err)
	}
}

func (s *audioStream) handleRead(r []byte) {
	if len(r) >= 580 && (bytes.Equal(r[:6], []byte{0x6c, 0x05, 0x00, 0x00, 0x00, 0x00}) || bytes.Equal(r[:6], []byte{0x44, 0x02, 0x00, 0x00, 0x00, 0x00})) {
		s.handleAudioPacket(r)
	}
}

func (s *audioStream) init() {
	s.common.open("audio", 50003)
	s.rxSeqBufEntryChan = make(chan seqBufEntry)
	s.rxSeqBuf.init(rxSeqBufLength, 0xffff, 0, s.rxSeqBufEntryChan)
}

func (s *audioStream) start() {
	s.common.sendPkt3()
	s.common.waitForPkt4Answer()
	s.common.sendPkt6()
	s.common.waitForPkt6Answer()

	log.Print("stream started")

	s.timeoutTimer = time.NewTimer(audioTimeoutDuration)

	s.common.pkt7.startPeriodicSend(&s.common, 1)

	s.audioSendSeq = 1

	for {
		select {
		case r := <-s.common.readChan:
			s.handleRead(r)
		case <-s.timeoutTimer.C:
			exit(errors.New("audio stream timeout"))
		case e := <-s.rxSeqBufEntryChan:
			s.handleRxSeqBufEntry(e)
		case d := <-audio.rec:
			s.sendPart1(d[:1364])
			s.sendPart2(d[1364:1920])
		}
	}
}
