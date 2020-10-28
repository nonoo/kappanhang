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

	up, down, lost, retransmits := bandwidth.get()

	l := fmt.Sprint("up ", time.Since(s.startTime).Round(time.Second),
		" rtt ", s.rttLatency.Milliseconds(), "ms up ",
		bandwidth.formatByteCount(up), "/s down ",
		bandwidth.formatByteCount(down), "/s retx ", retransmits, " /1m lost ", lost, " /1m")

	if statusLogInterval < time.Second {
		log.printLineClear()
		fmt.Print(time.Now().Format("2006-01-02T15:04:05.000Z0700"), " ", l, "\r")
	} else {
		log.Print(l)
	}
}

func (s *statusLogStruct) loop() {
	for {
		select {
		case <-s.ticker.C:
			s.print()
		case <-s.stopChan:
			s.stopFinishedChan <- true
			return
		}
	}
}

func (s *statusLogStruct) startPeriodicPrint() {
	s.startTime = time.Now()
	s.stopChan = make(chan bool)
	s.stopFinishedChan = make(chan bool)
	s.ticker = time.NewTicker(statusLogInterval)
	go s.loop()
}

func (s *statusLogStruct) stopPeriodicPrint() {
	if s.ticker == nil { // Already stopped?
		return
	}
	s.ticker.Stop()
	s.ticker = nil

	s.stopChan <- true
	<-s.stopFinishedChan
}
