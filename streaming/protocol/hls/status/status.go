package status

import "time"

type Status struct {
	hasVideo       bool
	seqId          int64
	createdAt      time.Time
	segBeginAt     time.Time
	hasSetFirstTs  bool
	firstTimestamp int64
	lastTimestamp  int64
}

func NewStatus() *Status {
	return &Status{
		seqId:         0,
		hasSetFirstTs: false,
		segBeginAt:    time.Now(),
	}
}

func (t *Status) Update(isVideo bool, timestamp uint32) {
	if isVideo {
		t.hasVideo = true
	}
	if !t.hasSetFirstTs {
		t.hasSetFirstTs = true
		t.firstTimestamp = int64(timestamp)
	}
	t.lastTimestamp = int64(timestamp)
}

func (t *Status) ResetAndNew() {
	t.seqId++
	t.hasVideo = false
	t.createdAt = time.Now()
	t.hasSetFirstTs = false
}

func (t *Status) Duration() time.Duration {
	return time.Duration(t.lastTimestamp-t.firstTimestamp) * time.Millisecond
}
