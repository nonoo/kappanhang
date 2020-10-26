package main

import (
	"fmt"
	"sync"
	"time"
)

type bandwidthStruct struct {
	toRadioBytes   int
	toRadioPkts    int
	fromRadioBytes int
	fromRadioPkts  int
	lostPkts       int
	lastGet        time.Time
}

var bandwidth bandwidthStruct
var bandwidthMutex sync.Mutex

func (b *bandwidthStruct) reset() {
	bandwidthMutex.Lock()
	defer bandwidthMutex.Unlock()

	bandwidth = bandwidthStruct{}
}

// Call this function when a packet is sent or received.
func (b *bandwidthStruct) add(toRadioBytes, fromRadioBytes int) {
	bandwidthMutex.Lock()
	defer bandwidthMutex.Unlock()

	b.toRadioBytes += toRadioBytes
	if toRadioBytes > 0 {
		b.toRadioPkts++
	}
	b.fromRadioBytes += fromRadioBytes
	if fromRadioBytes > 0 {
		b.fromRadioPkts++
	}
}

// Call this function when a packet is sent or received.
func (b *bandwidthStruct) reportLoss(pkts int) {
	bandwidthMutex.Lock()
	defer bandwidthMutex.Unlock()

	b.lostPkts += pkts
}

func (b *bandwidthStruct) get() (toRadioBytesPerSec, fromRadioBytesPerSec int, lossPercent float64) {
	bandwidthMutex.Lock()
	defer bandwidthMutex.Unlock()

	secs := time.Since(b.lastGet).Seconds()
	toRadioBytesPerSec = int(float64(b.toRadioBytes) / secs)
	fromRadioBytesPerSec = int(float64(b.fromRadioBytes) / secs)
	lossPercent = (float64(b.lostPkts) / float64(b.lostPkts+b.fromRadioPkts)) * 100

	b.toRadioBytes = 0
	b.toRadioPkts = 0
	b.fromRadioBytes = 0
	b.fromRadioPkts = 0
	b.lostPkts = 0
	b.lastGet = time.Now()
	return
}

func (b *bandwidthStruct) formatByteCount(c int) string {
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
