package main

import (
	"errors"
	"sync"
	"time"
)

type seqNum int

func (s *seqNum) inc(maxSeqNum seqNum) seqNum {
	if *s == maxSeqNum {
		return 0
	}
	return *s + 1
}

func (s *seqNum) dec(maxSeqNum seqNum) seqNum {
	if *s == 0 {
		return maxSeqNum
	}
	return *s - 1
}

type seqNumRange [2]seqNum

func (r *seqNumRange) getDiff(maxSeqNum seqNum) (diff int) {
	from := r[0]
	to := r[1]

	if to >= from {
		diff = int(to) - int(from)
	} else {
		r[0] = from
		r[1] = to
		diff = (int(maxSeqNum) + 1) - int(from) + int(to)
	}
	return
}

type seqBufEntry struct {
	seq  seqNum
	data []byte
}

type requestRetransmitCallbackType func(r seqNumRange) error

type seqBuf struct {
	length                    time.Duration
	maxSeqNum                 seqNum
	maxSeqNumDiff             seqNum
	requestRetransmitCallback requestRetransmitCallbackType

	// Available entries coming out from the seqbuf will be sent to entryChan.
	entryChan chan seqBufEntry

	// If this is true then the seqBuf is locked, which means no entries will be sent to entryChan.
	lockedByInvalidSeq bool
	lockedAt           time.Time

	// This is false until no packets have been sent to the entryChan.
	alreadyReturnedFirstSeq bool
	// The seqNum of the last packet sent to entryChan.
	lastReturnedSeq seqNum

	requestedRetransmit          bool
	lastRequestedRetransmitRange seqNumRange

	ignoreMissingPktsUntilEnabled bool
	ignoreMissingPktsUntilSeq     seqNum

	// Note that the most recently added entry is stored as the 0th entry.
	entries []seqBufEntry
	mutex   sync.RWMutex

	entryAddedChan         chan bool
	watcherCloseNeededChan chan bool
	watcherCloseDoneChan   chan bool

	errOutOfOrder error
}

// func (s *seqBuf) string() (out string) {
// 	if len(s.entries) == 0 {
// 		return "empty"
// 	}
// 	for _, e := range s.entries {
// 		if out != "" {
// 			out += " "
// 		}
// 		out += fmt.Sprint(e.seq)
// 	}
// 	return out
// }

func (s *seqBuf) createEntry(seq seqNum, data []byte) seqBufEntry {
	return seqBufEntry{
		seq:  seq,
		data: data,
	}
}

func (s *seqBuf) notifyWatcher() {
	select {
	case s.entryAddedChan <- true:
	default:
	}
}

func (s *seqBuf) addToFront(seq seqNum, data []byte) {
	e := s.createEntry(seq, data)
	s.entries = append([]seqBufEntry{e}, s.entries...)

	s.notifyWatcher()
}

func (s *seqBuf) addToBack(seq seqNum, data []byte) {
	e := s.createEntry(seq, data)
	s.entries = append(s.entries, e)

	s.notifyWatcher()
}

func (s *seqBuf) insert(seq seqNum, data []byte, toPos int) {
	if toPos == 0 {
		s.addToFront(seq, data)
		return
	}
	if toPos >= len(s.entries) {
		s.addToBack(seq, data)
		return
	}
	sliceBefore := s.entries[:toPos]
	sliceAfter := s.entries[toPos:]
	e := s.createEntry(seq, data)
	s.entries = append(sliceBefore, append([]seqBufEntry{e}, sliceAfter...)...)

	s.notifyWatcher()
}

func (s *seqBuf) getDiff(seq1, seq2 seqNum) seqNum {
	if seq1 >= seq2 {
		return seq1 - seq2
	}
	seq2Overflowed := s.maxSeqNum + 1 - seq2
	return seq2Overflowed + seq1
}

type seqBufCompareResult int

const (
	larger = seqBufCompareResult(iota)
	smaller
	equal
)

// Compares seq to toSeq, considering the seq turnover at maxSeqNum.
// Example: returns larger for seq=2 toSeq=1
//          returns smaller for seq=0 toSeq=1
//          returns smaller for seq=39 toSeq=1 if maxSeqNum is 40
func (s *seqBuf) compareSeq(seq, toSeq seqNum) seqBufCompareResult {
	diff1 := s.getDiff(seq, toSeq)
	diff2 := s.getDiff(toSeq, seq)

	if diff1 == diff2 {
		return equal
	}

	if diff1 > diff2 {
		// This will cause an insert at the current position.
		if s.maxSeqNumDiff > 0 && diff2 > s.maxSeqNumDiff {
			return larger
		}

		return smaller
	}

	return larger
}

func (s *seqBuf) add(seq seqNum, data []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// log.Debug("inserting ", seq)
	// defer func() {
	// 	log.Print(s.String())
	// }()

	if seq > s.maxSeqNum {
		return errors.New("seq out of range")
	}

	if len(s.entries) == 0 {
		s.addToFront(seq, data)
		return nil
	}

	if s.entries[0].seq == seq { // Dropping duplicate seq.
		return nil
	}

	// Checking the first entry.
	if s.compareSeq(seq, s.entries[0].seq) == larger {
		s.addToFront(seq, data)
		return nil
	}

	// Parsing through other entries if there are more than 1.
	for i := 1; i < len(s.entries); i++ {
		// This seqnum is already in the queue? Ignoring it.
		if s.entries[i].seq == seq {
			return nil
		}

		if s.compareSeq(seq, s.entries[i].seq) == larger {
			// log.Debug("left for ", s.entries[i].seq)
			s.insert(seq, data, i)
			return nil
		}
		// log.Debug("right for ", s.entries[i].seq)
	}

	// No place found for the item?
	s.addToBack(seq, data)
	return nil
}

func (s *seqBuf) checkLockTimeout() (timeout bool, shouldRetryIn time.Duration) {
	timeSinceLastInvalidSeq := time.Since(s.lockedAt)
	if s.length > timeSinceLastInvalidSeq {
		shouldRetryIn = s.length - timeSinceLastInvalidSeq
		return
	}

	s.lockedByInvalidSeq = false
	// log.Debug("lock timeout")

	if s.requestedRetransmit {
		s.ignoreMissingPktsUntilSeq = s.lastRequestedRetransmitRange[1]
		s.ignoreMissingPktsUntilEnabled = true
	}

	return true, 0
}

// Returns true if all entries from the requested retransmit range have been received.
func (s *seqBuf) gotRetransmitRange() bool {
	entryIdx := len(s.entries)
	rangeSeq := s.lastRequestedRetransmitRange[0]

	for {
		entryIdx--
		if entryIdx < 0 {
			return false
		}

		if s.entries[entryIdx].seq != rangeSeq {
			// log.Debug("entry idx ", entryIdx, " seq #", s.entries[entryIdx].seq, " does not match ", rangeSeq)
			// log.Debug(s.string())
			return false
		}

		if rangeSeq == s.lastRequestedRetransmitRange[1] {
			return true
		}

		rangeSeq = rangeSeq.inc(s.maxSeqNum)
	}
}

// shouldRetryIn is only filled when no entry is available, but there are entries in the seqbuf.
// err is not nil if the seqbuf is empty.
func (s *seqBuf) get() (e seqBufEntry, shouldRetryIn time.Duration, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.entries) == 0 {
		return e, 0, errors.New("seqbuf is empty")
	}

	entryCount := len(s.entries)
	lastEntryIdx := entryCount - 1
	e = s.entries[lastEntryIdx]

	if s.alreadyReturnedFirstSeq {
		if s.lockedByInvalidSeq {
			if s.requestedRetransmit && s.gotRetransmitRange() {
				s.lockedByInvalidSeq = false
				// log.Debug("lock over")
			} else {
				var timeout bool
				if timeout, shouldRetryIn = s.checkLockTimeout(); !timeout {
					return
				}
			}
		} else {
			if s.compareSeq(e.seq, seqNum(s.lastReturnedSeq)) != larger {
				// log.Debug("ignoring out of order seq ", e.seq)
				s.entries = s.entries[:lastEntryIdx]
				err = s.errOutOfOrder
				return
			}

			if s.ignoreMissingPktsUntilEnabled {
				if s.compareSeq(e.seq, s.ignoreMissingPktsUntilSeq) == larger {
					// log.Debug("ignore over ", e.seq, " ", s.ignoreMissingPktsUntilSeq)
					s.ignoreMissingPktsUntilEnabled = false
				} else {
					// log.Debug("ignoring missing pkt, seq #", e.seq, " until ", s.ignoreMissingPktsUntilSeq)
				}
			} else {
				expectedNextSeq := s.lastReturnedSeq.inc(s.maxSeqNum)

				if e.seq != expectedNextSeq {
					// log.Debug("lock on, expected seq ", expectedNextSeq, " got ", e.seq)
					s.lockedByInvalidSeq = true
					s.lockedAt = time.Now()
					s.requestedRetransmit = false
					s.ignoreMissingPktsUntilEnabled = false
					shouldRetryIn = s.length

					if s.requestRetransmitCallback != nil {
						s.lastRequestedRetransmitRange[0] = expectedNextSeq
						s.lastRequestedRetransmitRange[1] = e.seq.dec(s.maxSeqNum)
						if err = s.requestRetransmitCallback(s.lastRequestedRetransmitRange); err == nil {
							s.requestedRetransmit = true
						}
					}
					return
				}
			}
		}
	}

	s.lastReturnedSeq = e.seq
	s.alreadyReturnedFirstSeq = true

	s.entries = s.entries[:lastEntryIdx]
	return e, 0, nil
}

func (s *seqBuf) watcher() {
	defer func() {
		s.watcherCloseDoneChan <- true
	}()

	entryAvailableTimer := time.NewTimer(0)
	<-entryAvailableTimer.C
	var entryAvailableTimerRunning bool

	for {
		retry := true

		for retry {
			retry = false

			e, t, err := s.get()
			if err == nil && t == 0 {
				if s.entryChan != nil {
					select {
					case s.entryChan <- e:
					case <-s.watcherCloseNeededChan:
						return
					}
				}

				// We may have further available entries.
				retry = true
			} else {
				if err == s.errOutOfOrder {
					retry = true
				} else if !entryAvailableTimerRunning && t > 0 {
					// An entry will be available later, waiting for it.
					entryAvailableTimer.Reset(t)
					entryAvailableTimerRunning = true
				}
			}
		}

		select {
		case <-s.watcherCloseNeededChan:
			return
		case <-s.entryAddedChan:
		case <-entryAvailableTimer.C:
			entryAvailableTimerRunning = false
		}
	}
}

// Setting a max. seqnum diff is optional. If it's 0 then the diff will be half of the maxSeqNum range.
// Available entries coming out from the seqbuf will be sent to entryChan.
func (s *seqBuf) init(length time.Duration, maxSeqNum, maxSeqNumDiff seqNum, entryChan chan seqBufEntry,
	requestRetransmitCallback requestRetransmitCallbackType) {
	s.length = length
	s.maxSeqNum = maxSeqNum
	s.maxSeqNumDiff = maxSeqNumDiff
	s.entryChan = entryChan
	s.requestRetransmitCallback = requestRetransmitCallback

	s.entryAddedChan = make(chan bool)
	s.watcherCloseNeededChan = make(chan bool)
	s.watcherCloseDoneChan = make(chan bool)

	s.errOutOfOrder = errors.New("out of order pkt")

	go s.watcher()
}

func (s *seqBuf) deinit() {
	if s.watcherCloseNeededChan == nil { // Init has not ran?
		return
	}
	s.watcherCloseNeededChan <- true
	<-s.watcherCloseDoneChan
}
