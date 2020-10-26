package main

import (
	"fmt"
	"sync"
	"time"
)

type bandwidthStruct struct {
	toRadioBytes   int
	fromRadioBytes int
	lastGet        time.Time
}

var bandwidth bandwidthStruct
var bandwidthMutex sync.Mutex

func (b *bandwidthStruct) reset() {
	bandwidthMutex.Lock()
	defer bandwidthMutex.Unlock()

	bandwidth = bandwidthStruct{}
}

func (b *bandwidthStruct) add(toRadioBytes, fromRadioBytes int) {
	bandwidthMutex.Lock()
	defer bandwidthMutex.Unlock()

	b.toRadioBytes += toRadioBytes
	b.fromRadioBytes += fromRadioBytes
}

func (b *bandwidthStruct) get() (toRadioBytesPerSec, fromRadioBytesPerSec int) {
	bandwidthMutex.Lock()
	defer bandwidthMutex.Unlock()

	secs := time.Since(b.lastGet).Seconds()
	toRadioBytesPerSec = int(float64(b.toRadioBytes) / secs)
	fromRadioBytesPerSec = int(float64(b.fromRadioBytes) / secs)

	b.toRadioBytes = 0
	b.fromRadioBytes = 0
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
