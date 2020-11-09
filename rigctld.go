package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

const (
	rigctldNoError        = iota
	rigctldInvalidParam   = -1
	rigctldUnsupportedCmd = -11
)

type rigctldStruct struct {
	listener net.Listener
	client   net.Conn

	clientLoopDeinitNeededChan   chan bool
	clientLoopDeinitFinishedChan chan bool

	deinitNeededChan   chan bool
	deinitFinishedChan chan bool
}

var rigctld rigctldStruct

func (s *rigctldStruct) disconnectClient() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *rigctldStruct) deinitClient() {
	if s.clientLoopDeinitNeededChan != nil {
		s.clientLoopDeinitNeededChan <- true
		<-s.clientLoopDeinitFinishedChan

		s.clientLoopDeinitNeededChan = nil
		s.clientLoopDeinitFinishedChan = nil
	}
}

func (s *rigctldStruct) send(a ...interface{}) error {
	str := fmt.Sprint(a...)
	_, err := s.client.Write([]byte(str))
	return err
}

func (s *rigctldStruct) sendReplyCode(code int) error {
	str := fmt.Sprint("RPRT ", code, "\n")
	_, err := s.client.Write([]byte(str))
	return err
}

func (s *rigctldStruct) processCmd(cmd string) (close bool, err error) {
	cmdSplit := strings.Fields(cmd)

	switch {
	case cmd == "\\chk_vfo":
		err = s.send("0\n")
	case cmd == "\\dump_state":
		err = s.send("1\n" +
			"3085\n" +
			"0\n" +
			"30000.000000 199999999.000000 0x1401dbf -1 -1 0x10000003 0x1\n" +
			"400000000.000000 470000000.000000 0x1401dbf -1 -1 0x10000003 0x1\n" +
			"0 0 0 0 0 0 0\n" +
			"1800000.000000 1999999.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"3500000.000000 3999999.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"5255000.000000 5405000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"7000000.000000 7300000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"10100000.000000 10150000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"14000000.000000 14350000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"18068000.000000 18168000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"21000000.000000 21450000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"24890000.000000 24990000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"28000000.000000 29700000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"50000000.000000 54000000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"144000000.000000 148000000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"430000000.000000 450000000.000000 0x10001bf 100 10000 0x10000003 0x1\n" +
			"0 0 0 0 0 0 0\n" +
			"0x401dbf 100\n" +
			"0x401dbf 500\n" +
			"0x401dbf 1000\n" +
			"0x401dbf 5000\n" +
			"0x401dbf 6250\n" +
			"0x401dbf 8330\n" +
			"0x401dbf 9000\n" +
			"0x401dbf 10000\n" +
			"0x401dbf 12500\n" +
			"0x401dbf 20000\n" +
			"0x401dbf 25000\n" +
			"0x401dbf 50000\n" +
			"0x401dbf 100000\n" +
			"0 0\n" +
			"0xc0c 3600\n" +
			"0xc0c 2400\n" +
			"0xc0c 1800\n" +
			"0x192 500\n" +
			"0x192 250\n" +
			"0x82 1200\n" +
			"0x110 2400\n" +
			"0x400001 6000\n" +
			"0x400001 3000\n" +
			"0x400001 9000\n" +
			"0x1020 10000\n" +
			"0x1020 7000\n" +
			"0x1020 15000\n" +
			"0 0\n" +
			"9999\n" +
			"9999\n" +
			"0\n" +
			"0\n" +
			"1 2\n" +
			"20\n" +
			"0xc90133fe\n" +
			"0xc90133fe\n" +
			"0x7f74677f3f\n" +
			"0x7000677f3f\n" +
			"0x35\n" +
			"0x35\n" +
			"vfo_ops=0x81f\n" +
			"ptt_type=0x1\n" +
			"targetable_vfo=0x0\n" +
			"done\n")
	case cmd == "q":
		err = s.sendReplyCode(rigctldNoError)
		close = true
	case cmd == "f":
		civControl.state.mutex.Lock()
		defer civControl.state.mutex.Unlock()

		err = s.send(civControl.state.freq, "\n")
	case cmdSplit[0] == "F":
		var f float64
		f, err = strconv.ParseFloat(cmdSplit[1], 0)
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		err = civControl.setMainVFOFreq(uint(f))
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		err = s.sendReplyCode(rigctldNoError)
	case cmd == "m":
		civControl.state.mutex.Lock()
		defer civControl.state.mutex.Unlock()

		var mode string
		if civControl.state.dataMode {
			mode = "PKT"
		}
		mode += civOperatingModes[civControl.state.operatingModeIdx].name

		// This can be queried with a CIV command for accurate values by the way.
		width := "3000"
		switch civControl.state.filterIdx {
		case 1:
			width = "2400"
		case 2:
			width = "1800"
		}
		err = s.send(mode, "\n", width, "\n")
	case cmdSplit[0] == "M":
		mode := cmdSplit[1]
		if mode[:3] == "PKT" {
			err = civControl.setDataMode(true)
			if err != nil {
				_ = s.sendReplyCode(rigctldInvalidParam)
				return
			}
			mode = mode[3:]
		}
		var modeCode byte
		var modeFound bool
		for _, m := range civOperatingModes {
			if m.name == mode {
				modeCode = m.code
				modeFound = true
				break
			}
		}
		if !modeFound {
			err = fmt.Errorf("unknown mode %s", mode)
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		var width int
		width, err = strconv.Atoi(cmdSplit[2])
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		var filterCode byte
		if width <= 1800 {
			filterCode = 2
		} else if width <= 2400 {
			filterCode = 1
		}
		err = civControl.setOperatingModeAndFilter(modeCode, filterCode)
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
		} else {
			_ = s.sendReplyCode(rigctldNoError)
		}
	case cmd == "t":
		civControl.state.mutex.Lock()
		defer civControl.state.mutex.Unlock()

		res := "0"
		if civControl.state.ptt {
			res = "1"
		}
		err = s.send(res, "\n")
	case cmdSplit[0] == "T":
		if cmdSplit[1] != "0" {
			err = civControl.setPTT(true)
		} else {
			err = civControl.setPTT(false)
		}
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
		} else {
			_ = s.sendReplyCode(rigctldNoError)
		}
	case cmdSplit[0] == "V":
		if cmdSplit[1] == "VFOB" {
			err = civControl.setVFO(1)
		} else {
			err = civControl.setVFO(0)
		}
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
		} else {
			_ = s.sendReplyCode(rigctldNoError)
		}
	case cmd == "s":
		civControl.state.mutex.Lock()
		defer civControl.state.mutex.Unlock()

		res := "0"
		if civControl.state.splitMode == splitModeOn {
			res = "1"
		}
		err = s.send(res, "\n")
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		if civControl.state.vfoBActive {
			res = "VFOA"
		} else {
			res = "VFOB"
		}
		err = s.send(res, "\n")
	case cmdSplit[0] == "S":
		if cmdSplit[1] == "1" {
			err = civControl.setSplit(splitModeOn)
		} else {
			err = civControl.setSplit(splitModeOff)
		}
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
		} else {
			_ = s.sendReplyCode(rigctldNoError)
		}
	case cmd == "i":
		civControl.state.mutex.Lock()
		defer civControl.state.mutex.Unlock()

		err = s.send(civControl.state.subFreq, "\n")
	case cmdSplit[0] == "I":
		var f float64
		f, err = strconv.ParseFloat(cmdSplit[1], 0)
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		err = civControl.setSubVFOFreq(uint(f))
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		err = s.sendReplyCode(rigctldNoError)
	case cmd == "x":
		civControl.state.mutex.Lock()
		defer civControl.state.mutex.Unlock()

		var mode string
		if civControl.state.subDataMode {
			mode = "PKT"
		}
		mode += civOperatingModes[civControl.state.subOperatingModeIdx].name

		// This can be queried with a CIV command for accurate values by the way.
		width := "3000"
		switch civControl.state.subFilterIdx {
		case 1:
			width = "2400"
		case 2:
			width = "1800"
		}
		err = s.send(mode, "\n", width, "\n")
	case cmdSplit[0] == "X":
		mode := cmdSplit[1]
		var dataMode byte
		if mode[:3] == "PKT" {
			dataMode = 1
			mode = mode[3:]
		}
		var modeCode byte
		var modeFound bool
		for _, m := range civOperatingModes {
			if m.name == mode {
				modeCode = m.code
				modeFound = true
				break
			}
		}
		if !modeFound {
			err = fmt.Errorf("unknown mode %s", mode)
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		var width int
		width, err = strconv.Atoi(cmdSplit[2])
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
			return
		}
		var filterCode byte
		if width <= 1800 {
			filterCode = 2
		} else if width <= 2400 {
			filterCode = 1
		}
		err = civControl.setSubVFOMode(modeCode, dataMode, filterCode)
		if err != nil {
			_ = s.sendReplyCode(rigctldInvalidParam)
		} else {
			_ = s.sendReplyCode(rigctldNoError)
		}
	case cmd == "v": // Ignore this command.
		_ = s.sendReplyCode(rigctldUnsupportedCmd)
		return
	default:
		_ = s.sendReplyCode(rigctldUnsupportedCmd)
		return false, fmt.Errorf("got unknown cmd %s", cmd)
	}
	return
}

func (s *rigctldStruct) clientLoop() {
	defer func() {
		s.client.Close()
		log.Print("client ", s.client.RemoteAddr().String(), " disconnected")

		<-s.clientLoopDeinitNeededChan
		s.clientLoopDeinitFinishedChan <- true
	}()

	log.Print("client ", s.client.RemoteAddr().String(), " connected")

	var b [128]byte
	var lineBuf bytes.Buffer
	for {
		n, err := s.client.Read(b[:])
		if err != nil {
			break
		}

		select {
		case <-s.clientLoopDeinitNeededChan:
			s.clientLoopDeinitFinishedChan <- true
			return
		default:
		}

		lineBuf.Write(b[:n])
		endIndex := bytes.Index(lineBuf.Bytes(), []byte{'\n'})
		if endIndex >= 0 {
			lineB := make([]byte, endIndex+1)
			n, err := lineBuf.Read(lineB)
			if err != nil {
				log.Error(err)
				return
			}
			if n < endIndex+1 {
				log.Error("short read")
				return
			}
			if n > 1 {
				close, err := s.processCmd(strings.TrimSpace(string(lineB[:len(lineB)-1])))
				if err != nil {
					log.Error(err)
				}
				if close {
					return
				}
			}
		}
	}
}

func (s *rigctldStruct) loop() {
	for {
		newClient, err := s.listener.Accept()

		s.disconnectClient()
		s.deinitClient()

		s.clientLoopDeinitNeededChan = make(chan bool)
		s.clientLoopDeinitFinishedChan = make(chan bool)

		if err != nil {
			if err != io.EOF {
				reportError(err)
			}
			<-s.deinitNeededChan
			s.deinitFinishedChan <- true
			return
		}

		s.client = newClient

		go s.clientLoop()
	}
}

// We only init the serial port TCP server once, with the first device name we acquire, so apps using the
// serial port TCP server won't have issues with the interface going down while the app is running.
func (s *rigctldStruct) initIfNeeded() (err error) {
	if s.listener != nil {
		return
	}

	s.listener, err = net.Listen("tcp", fmt.Sprint(":", rigctldPort))
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Print("starting internal rigctld on tcp port ", rigctldPort)

	s.deinitNeededChan = make(chan bool)
	s.deinitFinishedChan = make(chan bool)
	go s.loop()
	return
}

func (s *rigctldStruct) deinit() {
	if s.listener != nil {
		s.listener.Close()
	}

	if s.deinitNeededChan != nil {
		s.deinitNeededChan <- true
		<-s.deinitFinishedChan
	}
}
