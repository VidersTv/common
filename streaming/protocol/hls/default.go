package hls

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/streaming/protocol/hls/cache"
)

type Config struct {
	MinSegmentDuration time.Duration
	Logger             logrus.FieldLogger
	Cache              *cache.Cache
}

func (c Config) fill() Config {
	if c.Cache == nil {
		c.Cache = cache.New()
	}
	if c.Logger == nil {
		c.Logger = DefaultConfig.Logger
	}
	if c.MinSegmentDuration == 0 {
		c.MinSegmentDuration = DefaultConfig.MinSegmentDuration
	}

	return c
}

var DefaultConfig = Config{
	MinSegmentDuration: time.Second,
	Logger:             logrus.StandardLogger(),
}
