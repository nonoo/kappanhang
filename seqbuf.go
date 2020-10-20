package main

import (
	"errors"
	"fmt"
	"time"
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

	// Note that the most recently added entry is stored as the 0th entry.
	entries []seqBufEntry
}

func (s *seqBuf) String() (out string) {
	if len(s.entries) == 0 {
		return "empty"
	}
	for _, e := range s.entries {
		if out != "" {
			out += " "
		}
		out += fmt.Sprint(e.seq)
	}
	return out
}

func (s *seqBuf) createEntry(seq seqNum, data []byte) seqBufEntry {
	return seqBufEntry{
		seq:     seq,
		data:    data,
		addedAt: time.Now(),
	}
}

func (s *seqBuf) addToFront(seq seqNum, data []byte) {
	s.entries = append([]seqBufEntry{s.createEntry(seq, data)}, s.entries...)
}

func (s *seqBuf) addToBack(seq seqNum, data []byte) {
	s.entries = append(s.entries, s.createEntry(seq, data))
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
	s.entries = append(sliceBefore, append([]seqBufEntry{s.createEntry(seq, data)}, sliceAfter...)...)
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
			s.addToFront(seq, data)
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

func (s *seqBuf) get() (data []byte, err error) {
	if len(s.entries) == 0 {
		return nil, errors.New("seqbuf is empty")
	}
	lastEntryIdx := len(s.entries) - 1
	if time.Since(s.entries[lastEntryIdx].addedAt) < s.length {
		return nil, errors.New("no available entries")
	}
	data = make([]byte, len(s.entries[lastEntryIdx].data))
	copy(data, s.entries[lastEntryIdx].data)
	s.entries = s.entries[:lastEntryIdx]
	return data, nil
}

// Setting a max. seqnum diff is optional. If it's 0 then the diff will be half of the maxSeqNum range.
func (s *seqBuf) init(length time.Duration, maxSeqNum, maxSeqNumDiff seqNum) {
	s.length = length
	s.maxSeqNum = maxSeqNum
	s.maxSeqNumDiff = maxSeqNumDiff
}
