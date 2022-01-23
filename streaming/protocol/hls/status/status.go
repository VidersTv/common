package status

import "time"

type Status struct {
	hasVideo       bool
	seqId          int64
	hasSetFirstTs  bool
	firstTimestamp int64
	lastTimestamp  int64
}

func (t *Status) Update(isVideo bool, ts uint32) {
	if isVideo {
		t.hasVideo = true
	}
	if !t.hasSetFirstTs {
		t.hasSetFirstTs = true
		t.firstTimestamp = int64(ts)
	}
	t.lastTimestamp = int64(ts)
}

func (t *Status) ResetAndNew() {
	t.seqId++
	t.hasVideo = false
	t.hasSetFirstTs = false
	t.lastTimestamp = 0
	t.firstTimestamp = 0
}

func (t *Status) Duration() time.Duration {
	return time.Duration(t.lastTimestamp-t.firstTimestamp) * time.Millisecond
}
