package structures

type RedisRtmpEvent struct {
	Type RedisRtmpEventType `json:"type"`
	Key  string             `json:"key"`
}

type RedisRtmpEventType int32

const (
	RedisRtmpEventTypeKill RedisRtmpEventType = iota
)
