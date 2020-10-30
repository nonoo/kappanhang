package main

import (
	"math"
	"time"
)

const civAddress = 0xa4

type civControlStruct struct {
	st *serialStream

	state struct {
		ptt        bool
		tune       bool
		pwrPercent int
	}
}

var civControl *civControlStruct

func (s *civControlStruct) decode(d []byte) {
	if len(d) < 6 || d[0] != 0xfe || d[1] != 0xfe || d[len(d)-1] != 0xfd {
		return
	}

	payload := d[5 : len(d)-1]

	switch d[4] {
	case 0x00:
		s.decodeFreq(payload)
	case 0x01:
		s.decodeMode(payload)
	case 0x03:
		s.decodeFreq(payload)
	case 0x04:
		s.decodeMode(payload)
	case 0x1a:
		s.decodeDataMode(payload)
	case 0x14:
		s.decodePower(payload)
	case 0x1c:
		s.decodeTransmitStatus(payload)
	}
}

func (s *civControlStruct) decodeFreq(d []byte) {
	var f float64
	var pos int
	for _, v := range d {
		s1 := v & 0x0f
		s2 := v >> 4
		f += float64(s1) * math.Pow(10, float64(pos))
		pos++
		f += float64(s2) * math.Pow(10, float64(pos))
		pos++
	}
	statusLog.reportFrequency(f)
}

func (s *civControlStruct) decodeFilterValue(v byte) string {
	switch v {
	case 0x01:
		return "FIL1"
	case 0x02:
		return "FIL2"
	case 0x03:
		return "FIL3"
	}
	return ""
}

func (s *civControlStruct) decodeMode(d []byte) {
	if len(d) < 1 {
		return
	}

	var mode string
	switch d[0] {
	case 0x00:
		mode = "LSB"
	case 0x01:
		mode = "USB"
	case 0x02:
		mode = "AM"
	case 0x03:
		mode = "CW"
	case 0x04:
		mode = "RTTY"
	case 0x05:
		mode = "FM"
	case 0x06:
		mode = "WFM"
	case 0x07:
		mode = "CW-R"
	case 0x08:
		mode = "RTTY-R"
	case 0x17:
		mode = "DV"
	}

	var filter string
	if len(d) > 1 {
		filter = s.decodeFilterValue(d[1])
	}
	statusLog.reportMode(mode, filter)

	// The transceiver does not send the data mode setting automatically.
	_ = s.getDataMode()
}

func (s *civControlStruct) decodeDataMode(d []byte) {
	if len(d) < 3 || d[0] != 0x06 {
		return
	}

	var dataMode string
	var filter string
	if d[1] == 1 {
		dataMode = "-D"
		filter = s.decodeFilterValue(d[2])
	}

	statusLog.reportDataMode(dataMode, filter)
}

func (s *civControlStruct) decodePower(d []byte) {
	if len(d) < 3 || d[0] != 0x0a {
		return
	}

	hex := uint16(d[1])<<8 | uint16(d[2])
	s.state.pwrPercent = int(math.Round((float64(hex) / 0x0255) * 100))

	statusLog.reportTxPower(s.state.pwrPercent)
}

func (s *civControlStruct) decodeTransmitStatus(d []byte) {
	if len(d) < 2 {
		return
	}

	switch d[0] {
	case 0:
		if d[1] == 1 {
			s.state.ptt = true
		} else {
			s.state.ptt = false
		}
	case 1:
		if d[1] == 2 {
			s.state.tune = true

			// The transceiver does not send the tune state after it's finished.
			time.AfterFunc(time.Second, func() {
				_ = s.getTransmitStatus()
			})
		} else {
			s.state.tune = false
		}
	}
	statusLog.reportPTT(s.state.ptt, s.state.tune)
}

func (s *civControlStruct) setPwr(percent int) error {
	v := uint16(0x0255 * (float64(percent) / 100))
	return s.st.send([]byte{254, 254, civAddress, 224, 0x14, 0x0a, byte(v >> 8), byte(v & 0xff), 253})
}

func (s *civControlStruct) incPwr() error {
	if s.state.pwrPercent < 100 {
		s.state.pwrPercent++
		return s.setPwr(s.state.pwrPercent)
	}
	return nil
}

func (s *civControlStruct) decPwr() error {
	if s.state.pwrPercent > 0 {
		s.state.pwrPercent--
		return s.setPwr(s.state.pwrPercent)
	}
	return nil
}

func (s *civControlStruct) setPTT(enable bool) error {
	var b byte
	if enable {
		b = 1
	}
	return s.st.send([]byte{254, 254, civAddress, 224, 0x1c, 0, b, 253})
}

func (s *civControlStruct) toggleTune() error {
	if s.state.ptt {
		return nil
	}

	var b byte
	if !s.state.tune {
		b = 2
	} else {
		b = 1
	}
	return s.st.send([]byte{254, 254, civAddress, 224, 0x1c, 1, b, 253})
}

func (s *civControlStruct) getDataMode() error {
	return s.st.send([]byte{254, 254, civAddress, 224, 0x1a, 0x06, 253})
}

func (s *civControlStruct) getTransmitStatus() error {
	if err := s.st.send([]byte{254, 254, civAddress, 224, 0x1c, 0, 253}); err != nil {
		return err
	}
	return s.st.send([]byte{254, 254, civAddress, 224, 0x1c, 1, 253})
}

func (s *civControlStruct) init(st *serialStream) error {
	s.st = st

	// Querying frequency.
	if err := s.st.send([]byte{254, 254, civAddress, 224, 3, 253}); err != nil {
		return err
	}
	// Querying mode.
	if err := s.st.send([]byte{254, 254, civAddress, 224, 4, 253}); err != nil {
		return err
	}
	if err := s.getDataMode(); err != nil {
		return err
	}
	// Querying power.
	if err := s.st.send([]byte{254, 254, civAddress, 224, 0x14, 0x0a, 253}); err != nil {
		return err
	}
	if err := s.getTransmitStatus(); err != nil {
		return err
	}
	return nil
}
