package main

import "math"

const civAddress = 0xa4

type civDecoderStruct struct {
}

func (s *civDecoderStruct) decode(d []byte) {
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
	case 0x14:
		s.decodePower(payload)
	case 0x1c:
		s.decodePTT(payload)
	}
}

func (s *civDecoderStruct) decodeFreq(d []byte) {
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

func (s *civDecoderStruct) decodeMode(d []byte) {
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
		switch d[1] {
		case 0x01:
			filter = "FIL1"
		case 0x02:
			filter = "FIL2"
		case 0x03:
			filter = "FIL3"
		}
	}
	statusLog.reportMode(mode, filter)
}

func (s *civDecoderStruct) decodePower(d []byte) {
	if len(d) < 3 || d[0] != 0x0a {
		return
	}

	hex := uint16(d[1])<<8 | uint16(d[2])
	percent := int(math.Round((float64(hex) / 0x0255) * 100))

	statusLog.reportTxPower(percent)
}

func (s *civDecoderStruct) decodePTT(d []byte) {
	if len(d) < 2 {
		return
	}

	var ptt bool
	var tune bool
	switch d[0] {
	case 0:
		if d[1] == 1 {
			ptt = true
		}
	case 1:
		if d[1] == 2 {
			tune = true
		}
	}
	statusLog.reportPTT(ptt, tune)
}

func (s *civDecoderStruct) query(st *serialStream) error {
	// Querying frequency.
	if err := st.send([]byte{254, 254, civAddress, 224, 3, 253}); err != nil {
		return err
	}
	// Querying mode.
	if err := st.send([]byte{254, 254, civAddress, 224, 4, 253}); err != nil {
		return err
	}
	// Querying power.
	if err := st.send([]byte{254, 254, civAddress, 224, 0x14, 0x0a, 253}); err != nil {
		return err
	}
	// Querying PTT.
	if err := st.send([]byte{254, 254, civAddress, 224, 0x1c, 0, 253}); err != nil {
		return err
	}
	return nil
}
