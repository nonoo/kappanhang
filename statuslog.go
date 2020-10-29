package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

type statusLogStruct struct {
	ticker           *time.Ticker
	stopChan         chan bool
	stopFinishedChan chan bool
	mutex            sync.Mutex

	line1 string
	line2 string

	state struct {
		rxStr   string
		txStr   string
		tuneStr string
	}
	stateStr  string
	frequency float64
	mode      string
	filter    string

	startTime  time.Time
	rttLatency time.Duration
}

var statusLog statusLogStruct

func (s *statusLogStruct) reportRTTLatency(l time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.rttLatency = l
}

func (s *statusLogStruct) reportFrequency(f float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.frequency = f
}

func (s *statusLogStruct) reportMode(mode, filter string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.mode = mode
	s.filter = filter
}

func (s *statusLogStruct) reportPTT(ptt, tune bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if tune {
		s.stateStr = s.state.tuneStr
	} else if ptt {
		s.stateStr = s.state.txStr
	} else {
		s.stateStr = s.state.rxStr
	}
}

func (s *statusLogStruct) clearInternal() {
	fmt.Printf("%c[2K", 27)
}

func (s *statusLogStruct) clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.clearInternal()
}

func (s *statusLogStruct) print() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRealtimeInternal() {
		s.clearInternal()
		fmt.Println(s.line1)
		s.clearInternal()
		fmt.Printf(s.line2+"%c[1A", 27)
	} else {
		log.PrintStatusLog(s.line2)
	}
}

func (s *statusLogStruct) update() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var modeStr string
	if s.mode != "" {
		modeStr = " " + s.mode
	}
	var filterStr string
	if s.filter != "" {
		filterStr = " " + s.filter
	}
	s.line1 = fmt.Sprint("state ", s.stateStr, " freq: ", fmt.Sprintf("%f", s.frequency/1000000), modeStr, filterStr)

	up, down, lost, retransmits := netstat.get()
	s.line2 = fmt.Sprint("up ", time.Since(s.startTime).Round(time.Second),
		" rtt ", s.rttLatency.Milliseconds(), "ms up ",
		netstat.formatByteCount(up), "/s down ",
		netstat.formatByteCount(down), "/s retx ", retransmits, " /1m lost ", lost, " /1m\r")

	if s.isRealtimeInternal() {
		t := time.Now().Format("2006-01-02T15:04:05.000Z0700")
		s.line1 = fmt.Sprint(t, " ", s.line1)
		s.line2 = fmt.Sprint(t, " ", s.line2)
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

	if !isatty.IsTerminal(os.Stdout.Fd()) && statusLogInterval < time.Second {
		statusLogInterval = time.Second
	}

	c := color.New(color.FgHiWhite)
	c.Add(color.BgWhite)
	s.stateStr = c.Sprint("  ??  ")
	c = color.New(color.FgHiWhite)
	c.Add(color.BgGreen)
	s.state.rxStr = c.Sprint("  RX  ")
	c = color.New(color.FgHiWhite, color.BlinkRapid)
	c.Add(color.BgRed)
	s.state.txStr = c.Sprint("  TX  ")
	s.state.tuneStr = c.Sprint(" TUNE ")

	s.startTime = time.Now()
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

	fmt.Println()
	fmt.Println()
	fmt.Println()
}
