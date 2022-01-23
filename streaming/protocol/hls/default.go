package hls

import (
	"time"

	"github.com/sirupsen/logrus"
)

type Config struct {
	MinSegmentDuration time.Duration
	Logger             logrus.FieldLogger
}

var DefaultConfig = Config{
	MinSegmentDuration: time.Second,
	Logger:             logrus.StandardLogger(),
}
