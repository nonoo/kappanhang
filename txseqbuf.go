package main

import "time"

const txSeqBufLength = 300 * time.Millisecond

type txSeqBufStruct struct {
	entries []seqBufEntry
}

func (s *txSeqBufStruct) add(seq seqNum, p []byte) {
	s.entries = append(s.entries, seqBufEntry{
		seq:     seq,
		data:    p,
		addedAt: time.Now(),
	})
	s.purgeOldEntries()
}

func (s *txSeqBufStruct) purgeOldEntries() {
	for len(s.entries) > 0 && time.Since(s.entries[0].addedAt) > txSeqBufLength*2 {
		s.entries = s.entries[1:]
	}
}

func (s *txSeqBufStruct) get(seq seqNum) (d []byte) {
	if len(s.entries) == 0 {
		return nil
	}

	// Searching from backwards, as we expect most queries for latest entries.
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].seq == seq {
			d = s.entries[i].data
			break
		}
	}
	s.purgeOldEntries()
	return d
}
