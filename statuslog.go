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

	stateStr   string
	frequency  float64
	mode       string
	dataMode   string
	filter     string
	txPowerStr string

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
		retransmitsColor *color.Color
		lostColor        *color.Color

		stateStr struct {
			unknown string
			rx      string
			tx      string
			tune    string
		}
		audioStateStr struct {
			off   string
			monOn string
			rec   string
		}
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

func (s *statusLogStruct) reportFrequency(f float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.frequency = f
}

func (s *statusLogStruct) reportMode(mode, filter string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.mode = mode
	s.data.filter = filter
}

func (s *statusLogStruct) reportDataMode(dataMode, filter string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.dataMode = dataMode
	if dataMode != "" {
		s.data.filter = filter
	}
}

func (s *statusLogStruct) reportPTT(ptt, tune bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	if tune {
		s.data.stateStr = s.preGenerated.stateStr.tune
	} else if ptt {
		s.data.stateStr = s.preGenerated.stateStr.tx
	} else {
		s.data.stateStr = s.preGenerated.stateStr.rx
	}
}

func (s *statusLogStruct) reportTxPower(percent int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.data == nil {
		return
	}
	s.data.txPowerStr = fmt.Sprint(percent, "%")
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
		fmt.Printf(s.data.line2+"%c[1A", 27)
	} else {
		log.PrintStatusLog(s.data.line2)
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
func (s *statusLogStruct) update() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var modeStr string
	if s.data.mode != "" {
		modeStr = " " + s.data.mode + s.data.dataMode
	}
	var filterStr string
	if s.data.filter != "" {
		filterStr = " " + s.data.filter
	}
	var txPowerStr string
	if s.data.txPowerStr != "" {
		txPowerStr = " txpwr " + s.data.txPowerStr
	}
	s.data.line1 = fmt.Sprint("state ", s.data.stateStr, " freq: ", fmt.Sprintf("%.6f", s.data.frequency),
		modeStr, filterStr, txPowerStr, " audio ", s.data.audioStateStr)

	up, down, lost, retransmits := netstat.get()
	lostStr := "0"
	if lost > 0 {
		lostStr = s.preGenerated.lostColor.Sprint(" ", lost, " ")
	}
	retransmitsStr := "0"
	if retransmits > 0 {
		retransmitsStr = s.preGenerated.retransmitsColor.Sprint(" ", retransmits, " ")
	}

	s.data.line2 = fmt.Sprint("up ", s.padLeft(fmt.Sprint(time.Since(s.data.startTime).Round(time.Second)), 6),
		" rtt ", s.padLeft(s.data.rttStr, 3), "ms up ",
		s.padLeft(netstat.formatByteCount(up), 8), "/s down ",
		s.padLeft(netstat.formatByteCount(down), 8), "/s retx ", retransmitsStr, "/1m lost ", lostStr, "/1m\r")

	if s.isRealtimeInternal() {
		t := time.Now().Format("2006-01-02T15:04:05.000Z0700")
		s.data.line1 = fmt.Sprint(t, " ", s.data.line1)
		s.data.line2 = fmt.Sprint(t, " ", s.data.line2)
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
		stateStr:      s.preGenerated.stateStr.unknown,
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
	s.preGenerated.stateStr.unknown = c.Sprint("  ??  ")
	s.preGenerated.audioStateStr.off = c.Sprint("  OFF  ")

	c = color.New(color.FgHiWhite)
	c.Add(color.BgGreen)
	s.preGenerated.stateStr.rx = c.Sprint("  RX  ")
	s.preGenerated.audioStateStr.monOn = c.Sprint("  MON  ")

	c = color.New(color.FgHiWhite, color.BlinkRapid)
	c.Add(color.BgRed)
	s.preGenerated.stateStr.tx = c.Sprint("  TX  ")
	s.preGenerated.stateStr.tune = c.Sprint(" TUNE ")
	s.preGenerated.audioStateStr.rec = c.Sprint("  REC  ")

	s.preGenerated.retransmitsColor = color.New(color.FgHiWhite)
	s.preGenerated.retransmitsColor.Add(color.BgYellow)
	s.preGenerated.lostColor = color.New(color.FgHiWhite)
	s.preGenerated.lostColor.Add(color.BgRed)
}
