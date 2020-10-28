package main

import (
	"fmt"
	"sync"
	"time"
)

type netstatStruct struct {
	toRadioBytes   int
	toRadioPkts    int
	fromRadioBytes int
	fromRadioPkts  int
	lastGet        time.Time

	lostPkts             int
	lastLostReport       time.Time
	retransmits          int
	lastRetransmitReport time.Time
}

var netstat netstatStruct
var netstatMutex sync.Mutex

func (b *netstatStruct) reset() {
	netstatMutex.Lock()
	defer netstatMutex.Unlock()

	netstat = netstatStruct{}
}

// Call this function when a packet is sent or received.
func (b *netstatStruct) add(toRadioBytes, fromRadioBytes int) {
	netstatMutex.Lock()
	defer netstatMutex.Unlock()

	b.toRadioBytes += toRadioBytes
	if toRadioBytes > 0 {
		b.toRadioPkts++
	}
	b.fromRadioBytes += fromRadioBytes
	if fromRadioBytes > 0 {
		b.fromRadioPkts++
	}
}

func (b *netstatStruct) reportLoss(pkts int) {
	netstatMutex.Lock()
	defer netstatMutex.Unlock()

	b.lastLostReport = time.Now()
	b.lostPkts += pkts
}

func (b *netstatStruct) reportRetransmit(pkts int) {
	netstatMutex.Lock()
	defer netstatMutex.Unlock()

	b.lastRetransmitReport = time.Now()
	b.retransmits += pkts
}

func (b *netstatStruct) get() (toRadioBytesPerSec, fromRadioBytesPerSec int, lost int, retransmits int) {
	netstatMutex.Lock()
	defer netstatMutex.Unlock()

	secs := time.Since(b.lastGet).Seconds()
	toRadioBytesPerSec = int(float64(b.toRadioBytes) / secs)
	fromRadioBytesPerSec = int(float64(b.fromRadioBytes) / secs)

	b.toRadioBytes = 0
	b.toRadioPkts = 0
	b.fromRadioBytes = 0
	b.fromRadioPkts = 0
	b.lastGet = time.Now()

	secs = time.Since(b.lastLostReport).Seconds()
	lost = b.lostPkts
	if secs >= 60 {
		b.lostPkts = 0
		b.lastLostReport = time.Now()
	}

	secs = time.Since(b.lastRetransmitReport).Seconds()
	retransmits = b.retransmits
	if secs >= 60 {
		b.retransmits = 0
		b.lastRetransmitReport = time.Now()
	}

	return
}

func (b *netstatStruct) formatByteCount(c int) string {
	const unit = 1000
	if c < unit {
		return fmt.Sprintf("%d B", c)
	}
	div, exp := int(unit), 0
	for n := c / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(c)/float64(div), "kMGTPE"[exp])
}
