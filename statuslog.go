package main

import (
	"fmt"
	"sync"
	"time"
)

type statusLogStruct struct {
	ticker           *time.Ticker
	stopChan         chan bool
	stopFinishedChan chan bool
	mutex            sync.Mutex

	line string

	startTime  time.Time
	rttLatency time.Duration
}

var statusLog statusLogStruct

func (s *statusLogStruct) reportRTTLatency(l time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.rttLatency = l
}

func (s *statusLogStruct) print() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRealtimeInternal() {
		log.printLineClear()
		fmt.Print(s.line)
	} else {
		log.Print(s.line)
	}
}

func (s *statusLogStruct) update() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	up, down, lost, retransmits := netstat.get()

	s.line = fmt.Sprint("up ", time.Since(s.startTime).Round(time.Second),
		" rtt ", s.rttLatency.Milliseconds(), "ms up ",
		netstat.formatByteCount(up), "/s down ",
		netstat.formatByteCount(down), "/s retx ", retransmits, " /1m lost ", lost, " /1m")

	if s.isRealtimeInternal() {
		s.line = fmt.Sprint(time.Now().Format("2006-01-02T15:04:05.000Z0700"), " ", s.line, "\r")
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
}
