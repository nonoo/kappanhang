package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

type statusLogData struct {
	line1 string
	line2 string
	line3 string

	ptt          bool
	tune         bool
	frequency    uint
	subFrequency uint
	mode         string
	dataMode     string
	filter       string
	subMode      string
	subDataMode  string
	subFilter    string
	preamp       string
	agc          string
	vd           string
	txPower      string
	rfGain       string
	sql          string
	nr           string
	nrEnabled    bool
	s            string
	ovf          bool
	swr          string
	ts           string
	split        string
	splitMode    splitMode

	startTime time.Time
	rttStr    string

	audioMonOn    bool
	audioRecOn    bool
	audioStateStr string
}

type statusLogStruct struct {
	ticker           *time.Ticker
	stopChan         chan bool
	stopFinishedChan chan bool
	mutex            sync.Mutex

	preGenerated struct {
		rxColor          *color.Color
		retransmitsColor *color.Color
		lostColor        *color.Color
		splitColor       *color.Color

		stateStr struct {
			tx   string
			tune string
		}
		audioStateStr struct {
			off   string
			monOn string
			rec   string
		}

		ovf string
	}

	data *statusLogData
}

var statusLog statusLogStruct

func (s *statusLogStruct) reportRTTLatency(l time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.rttStr = fmt.Sprint(l.Milliseconds())
}

func (s *statusLogStruct) updateAudioStateStr() {
	if s.data.audioRecOn {
		s.data.audioStateStr = s.preGenerated.audioStateStr.rec
	} else if s.data.audioMonOn {
		s.data.audioStateStr = s.preGenerated.audioStateStr.monOn
	} else {
		s.data.audioStateStr = s.preGenerated.audioStateStr.off
	}
}

func (s *statusLogStruct) reportAudioMon(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.audioMonOn = enabled
	s.updateAudioStateStr()
}

func (s *statusLogStruct) reportAudioRec(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.audioRecOn = enabled
	s.updateAudioStateStr()
}

func (s *statusLogStruct) reportFrequency(f uint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.frequency = f
}

func (s *statusLogStruct) reportSubFrequency(f uint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.subFrequency = f
}

func (s *statusLogStruct) reportMode(mode string, dataMode bool, filter string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.mode = mode
	if dataMode {
		s.data.dataMode = "-D"
	} else {
		s.data.dataMode = ""
	}
	s.data.filter = filter
}

func (s *statusLogStruct) reportSubMode(mode string, dataMode bool, filter string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.subMode = mode
	if dataMode {
		s.data.subDataMode = "-D"
	} else {
		s.data.subDataMode = ""
	}
	s.data.subFilter = filter
}

func (s *statusLogStruct) reportPreamp(preamp int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.preamp = fmt.Sprint("PAMP", preamp)
}

func (s *statusLogStruct) reportAGC(agc string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.agc = "AGC" + agc
}

func (s *statusLogStruct) reportNREnabled(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.nrEnabled = enabled
}

func (s *statusLogStruct) reportVd(voltage float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.vd = fmt.Sprintf("%.1fV", voltage)
}

func (s *statusLogStruct) reportS(sValue string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.s = sValue
}

func (s *statusLogStruct) reportOVF(ovf bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.ovf = ovf
}

func (s *statusLogStruct) reportSWR(swr float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.swr = fmt.Sprintf("%.1f", swr)
}

func (s *statusLogStruct) reportTS(ts uint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.ts = "TS"
	if ts >= 1000 {
		if ts%1000 == 0 {
			s.data.ts += fmt.Sprintf("%.0fk", float64(ts)/1000)
		} else if ts%100 == 0 {
			s.data.ts += fmt.Sprintf("%.1fk", float64(ts)/1000)
		} else {
			s.data.ts += fmt.Sprintf("%.2fk", float64(ts)/1000)
		}
	} else {
		s.data.ts += fmt.Sprint(ts)
	}
}

func (s *statusLogStruct) reportPTT(ptt, tune bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.tune = tune
	s.data.ptt = ptt
}

func (s *statusLogStruct) reportTxPower(percent int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.txPower = fmt.Sprint(percent, "%")
}

func (s *statusLogStruct) reportRFGain(percent int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.rfGain = fmt.Sprint(percent, "%")
}

func (s *statusLogStruct) reportSQL(percent int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.sql = fmt.Sprint(percent, "%")
}

func (s *statusLogStruct) reportNR(percent int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.nr = fmt.Sprint(percent, "%")
}

func (s *statusLogStruct) reportSplit(mode splitMode, split string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.splitMode = mode
	if split == "" {
		s.data.split = ""
	} else {
		s.data.split = s.preGenerated.splitColor.Sprint(split)
	}
}

func (s *statusLogStruct) clearInternal() {
	fmt.Printf("%c[2K", 27)
}

func (s *statusLogStruct) print() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRealtimeInternal() {
		s.clearInternal()
		fmt.Println(s.data.line1)
		s.clearInternal()
		fmt.Println(s.data.line2)
		s.clearInternal()
		fmt.Printf(s.data.line3+"%c[1A%c[1A", 27, 27)
	} else {
		log.PrintStatusLog(s.data.line3)
	}
}

func (s *statusLogStruct) padLeft(str string, length int) string {
	if !s.isRealtimeInternal() {
		return str
	}

	for len(str) < length {
		str = " " + str
	}
	return str
}

func (s *statusLogStruct) padRight(str string, length int) string {
	if !s.isRealtimeInternal() {
		return str
	}

	for len(str) < length {
		str += " "
	}
	return str
}

func (s *statusLogStruct) update() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var filterStr string
	if s.data.filter != "" {
		filterStr = " " + s.data.filter
	}
	var preampStr string
	if s.data.preamp != "" {
		preampStr = " " + s.data.preamp
	}
	var agcStr string
	if s.data.agc != "" {
		agcStr = " " + s.data.agc
	}
	var nrStr string
	if s.data.nr != "" {
		nrStr = " NR"
		if s.data.nrEnabled {
			nrStr += s.data.nr
		} else {
			nrStr += "-"
		}
	}
	var rfGainStr string
	if s.data.rfGain != "" {
		rfGainStr = " rfg " + s.data.rfGain
	}
	var sqlStr string
	if s.data.sql != "" {
		sqlStr = " sql " + s.data.sql
	}
	s.data.line1 = fmt.Sprint(s.data.audioStateStr, filterStr, preampStr, agcStr, nrStr, rfGainStr, sqlStr)

	var stateStr string
	if s.data.tune {
		stateStr = s.preGenerated.stateStr.tune
	} else if s.data.ptt {
		stateStr = s.preGenerated.stateStr.tx
	} else {
		var ovfStr string
		if s.data.ovf {
			ovfStr = s.preGenerated.ovf
		}
		if len(s.data.s) <= 2 {
			stateStr = s.preGenerated.rxColor.Sprint("  " + s.padRight(s.data.s, 4) + " ")
		} else {
			stateStr = s.preGenerated.rxColor.Sprint(" " + s.padRight(s.data.s, 5) + " ")
		}
		stateStr += ovfStr
	}
	var tsStr string
	if s.data.ts != "" {
		tsStr = " " + s.data.ts
	}
	var modeStr string
	if s.data.mode != "" {
		modeStr = " " + s.data.mode + s.data.dataMode
	}
	var vdStr string
	if s.data.vd != "" {
		vdStr = " " + s.data.vd
	}
	var txPowerStr string
	if s.data.txPower != "" {
		txPowerStr = " txpwr " + s.data.txPower
	}
	var splitStr string
	if s.data.split != "" {
		splitStr = " " + s.data.split
		if s.data.splitMode == splitModeOn {
			splitStr += fmt.Sprintf("/%.6f/%s%s/%s", float64(s.data.subFrequency)/1000000,
				s.data.subMode, s.data.subDataMode, s.data.subFilter)
		}
	}
	var swrStr string
	if (s.data.tune || s.data.ptt) && s.data.swr != "" {
		swrStr = " SWR" + s.data.swr
	}
	s.data.line2 = fmt.Sprint(stateStr, " ", fmt.Sprintf("%.6f", float64(s.data.frequency)/1000000),
		tsStr, modeStr, splitStr, vdStr, txPowerStr, swrStr)

	up, down, lost, retransmits := netstat.get()
	lostStr := "0"
	if lost > 0 {
		lostStr = s.preGenerated.lostColor.Sprint(" ", lost, " ")
	}
	retransmitsStr := "0"
	if retransmits > 0 {
		retransmitsStr = s.preGenerated.retransmitsColor.Sprint(" ", retransmits, " ")
	}

	s.data.line3 = fmt.Sprint("up ", s.padLeft(fmt.Sprint(time.Since(s.data.startTime).Round(time.Second)), 6),
		" rtt ", s.padLeft(s.data.rttStr, 3), "ms up ",
		s.padLeft(netstat.formatByteCount(up), 8), "/s down ",
		s.padLeft(netstat.formatByteCount(down), 8), "/s retx ", retransmitsStr, "/1m lost ", lostStr, "/1m\r")

	if s.isRealtimeInternal() {
		t := time.Now().Format("2006-01-02T15:04:05.000Z0700")
		s.data.line1 = fmt.Sprint(t, " ", s.data.line1)
		s.data.line2 = fmt.Sprint(t, " ", s.data.line2)
		s.data.line3 = fmt.Sprint(t, " ", s.data.line3)
	}
}

func (s *statusLogStruct) loop() {
	for {
		select {
		case <-s.ticker.C:
			s.update()
			s.print()
		case <-s.stopChan:
			s.stopFinishedChan <- true
			return
		}
	}
}

func (s *statusLogStruct) isRealtimeInternal() bool {
	return statusLogInterval < time.Second
}

func (s *statusLogStruct) isRealtime() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.ticker != nil && s.isRealtimeInternal()
}

func (s *statusLogStruct) isActive() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.ticker != nil
}

func (s *statusLogStruct) startPeriodicPrint() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.initIfNeeded()

	s.data = &statusLogData{
		s:             "S0",
		startTime:     time.Now(),
		rttStr:        "?",
		audioStateStr: s.preGenerated.audioStateStr.off,
	}

	s.stopChan = make(chan bool)
	s.stopFinishedChan = make(chan bool)
	s.ticker = time.NewTicker(statusLogInterval)
	go s.loop()
}

func (s *statusLogStruct) stopPeriodicPrint() {
	if !s.isActive() {
		return
	}
	s.ticker.Stop()
	s.ticker = nil

	s.stopChan <- true
	<-s.stopFinishedChan

	if s.isRealtimeInternal() {
		s.clearInternal()
		fmt.Println()
		s.clearInternal()
		fmt.Println()
		s.clearInternal()
		fmt.Println()
	}
}

func (s *statusLogStruct) initIfNeeded() {
	if s.data != nil { // Already initialized?
		return
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) && statusLogInterval < time.Second {
		statusLogInterval = time.Second
	}

	c := color.New(color.FgHiWhite)
	c.Add(color.BgWhite)
	s.preGenerated.audioStateStr.off = c.Sprint("  MON  ")

	s.preGenerated.rxColor = color.New(color.FgHiWhite)
	s.preGenerated.rxColor.Add(color.BgGreen)
	s.preGenerated.audioStateStr.monOn = s.preGenerated.rxColor.Sprint("  MON  ")

	c = color.New(color.FgHiWhite, color.BlinkRapid)
	c.Add(color.BgRed)
	s.preGenerated.stateStr.tx = c.Sprint("  TX   ")
	s.preGenerated.stateStr.tune = c.Sprint("  TUNE ")
	s.preGenerated.audioStateStr.rec = c.Sprint("  REC  ")

	c = color.New(color.FgHiWhite)
	c.Add(color.BgRed)
	s.preGenerated.ovf = c.Sprint(" OVF ")

	s.preGenerated.retransmitsColor = color.New(color.FgHiWhite)
	s.preGenerated.retransmitsColor.Add(color.BgYellow)
	s.preGenerated.lostColor = color.New(color.FgHiWhite)
	s.preGenerated.lostColor.Add(color.BgRed)

	s.preGenerated.splitColor = color.New(color.FgHiMagenta)
}
