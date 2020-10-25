package main

import (
	"errors"
	"sync"
	"time"

	"github.com/nonoo/kappanhang/log"
)

type seqNum int

type seqBufEntry struct {
	seq     seqNum
	data    []byte
	addedAt time.Time
}

type seqBuf struct {
	length        time.Duration
	maxSeqNum     seqNum
	maxSeqNumDiff seqNum
	entryChan     chan seqBufEntry

	// Note that the most recently added entry is stored as the 0th entry.
	entries []seqBufEntry
	mutex   sync.RWMutex

	entryAddedChan         chan bool
	watcherCloseNeededChan chan bool
	watcherCloseDoneChan   chan bool
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
		seq:     seq,
		data:    data,
		addedAt: time.Now(),
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
	if seq1 > seq2 {
		return seq1 - seq2
	}
	seq2Overflowed := s.maxSeqNum + 1 - seq2
	return seq2Overflowed + seq1
}

const (
	left = iota
	right
	equal
)

type direction int

// Decides the direction of which seq is closer to whichSeq, considering the seq turnover at maxSeqNum.
// Example: returns left for seq=2 whichSeq=1
//          returns right for seq=0 whichSeq=1
//          returns right for seq=39 whichSeq=1 if maxSeqNum is 40
func (s *seqBuf) leftOrRightCloserToSeq(seq, whichSeq seqNum) direction {
	diff1 := s.getDiff(seq, whichSeq)
	diff2 := s.getDiff(whichSeq, seq)

	if diff1 == diff2 {
		return equal
	}

	if diff1 > diff2 {
		// This will cause an insert at the current position.
		if s.maxSeqNumDiff > 0 && diff2 > s.maxSeqNumDiff {
			return left
		}

		return right
	}

	return left
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

	if s.entries[0].seq == seq {
		return errors.New("dropping duplicate seq")
	}

	// Checking the first entry.
	if s.leftOrRightCloserToSeq(seq, s.entries[0].seq) == left {
		s.addToFront(seq, data)
		return nil
	}

	// Parsing through other entries if there are more than 1.
	for i := 1; i < len(s.entries); i++ {
		// This seqnum is already in the queue, but not as the first entry? It's not a duplicate packet.
		// It can be a beginning of a new stream for example.
		if s.entries[i].seq == seq {
			// s.addToFront(seq, data)
			return nil
		}

		if s.leftOrRightCloserToSeq(seq, s.entries[i].seq) == left {
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

func (s *seqBuf) getNextDataAvailableRemainingTime() (time.Duration, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if len(s.entries) == 0 {
		return 0, errors.New("seqbuf is empty")
	}
	lastEntryIdx := len(s.entries) - 1
	inBufDuration := time.Since(s.entries[lastEntryIdx].addedAt)
	if inBufDuration >= s.length {
		return 0, nil
	}
	return s.length - inBufDuration, nil
}

func (s *seqBuf) get() (e seqBufEntry, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.entries) == 0 {
		return e, errors.New("seqbuf is empty")
	}
	lastEntryIdx := len(s.entries) - 1
	if time.Since(s.entries[lastEntryIdx].addedAt) < s.length {
		return e, errors.New("no available entries")
	}
	e = s.entries[lastEntryIdx]
	s.entries = s.entries[:lastEntryIdx]
	return e, nil
}

func (s *seqBuf) watcher() {
	defer func() {
		s.watcherCloseDoneChan <- true
	}()

	entryAvailableTimer := time.NewTimer(s.length)
	entryAvailableTimer.Stop()
	var entryAvailableTimerRunning bool

	for {
		retry := true

		for retry {
			retry = false

			t, err := s.getNextDataAvailableRemainingTime()
			if err == nil {
				if t == 0 { // Do we have an entry available right now?
					e, err := s.get()
					if err == nil {
						if s.entryChan != nil {
							select {
							case s.entryChan <- e:
							case <-s.watcherCloseNeededChan:
								return
							}
						}
					} else {
						log.Error(err)
					}

					// We may have further available entries.
					retry = true
				} else if !entryAvailableTimerRunning {
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
func (s *seqBuf) init(length time.Duration, maxSeqNum, maxSeqNumDiff seqNum, entryChan chan seqBufEntry) {
	s.length = length
	s.maxSeqNum = maxSeqNum
	s.maxSeqNumDiff = maxSeqNumDiff
	s.entryChan = entryChan

	s.entryAddedChan = make(chan bool)
	s.watcherCloseNeededChan = make(chan bool)
	s.watcherCloseDoneChan = make(chan bool)
	go s.watcher()
}

func (s *seqBuf) deinit() {
	if s.watcherCloseNeededChan == nil { // Init has not ran?
		return
	}
	s.watcherCloseNeededChan <- true
	<-s.watcherCloseDoneChan
}
