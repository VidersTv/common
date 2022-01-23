package rtmp

import (
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/protocol/rtmp/core"
)

type Config struct {
	Logger          logrus.FieldLogger
	AuthTimeout     time.Duration
	OnError         func(error)
	OnNewStream     func(addr net.Addr) bool
	OnStreamClose   func(info av.Info, addr net.Addr)
	AuthStream      func(info *av.Info, addr net.Addr) bool
	HandlePublisher func(info av.Info, reader av.ReadCloser)
	HandleViewer    func(info av.Info, writer av.WriteCloser)
	HandleCmdChunk  func(info av.Info, vs []interface{}, chunk *core.ChunkStream) error
}

func (c Config) fill() Config {
	if c.Logger == nil {
		c.Logger = DefaultConfig.Logger
	}
	if c.OnError == nil {
		c.OnError = DefaultConfig.OnError
	}
	if c.OnNewStream == nil {
		c.OnNewStream = DefaultConfig.OnNewStream
	}
	if c.OnStreamClose == nil {
		c.OnStreamClose = DefaultConfig.OnStreamClose
	}
	if c.AuthStream == nil {
		c.AuthStream = DefaultConfig.AuthStream
	}
	if c.HandlePublisher == nil {
		c.HandlePublisher = DefaultConfig.HandlePublisher
	}
	if c.HandleViewer == nil {
		c.HandleViewer = DefaultConfig.HandleViewer
	}
	if c.HandleCmdChunk == nil {
		c.HandleCmdChunk = DefaultConfig.HandleCmdChunk
	}
	if c.AuthTimeout <= 0 {
		c.AuthTimeout = DefaultConfig.AuthTimeout
	}

	return c
}

var DefaultConfig = Config{
	AuthTimeout:     time.Second * 5,
	Logger:          logrus.StandardLogger(),
	OnError:         func(e error) {},
	OnNewStream:     func(addr net.Addr) bool { return true },
	OnStreamClose:   func(info av.Info, addr net.Addr) {},
	AuthStream:      func(info *av.Info, addr net.Addr) bool { return true },
	HandlePublisher: func(info av.Info, reader av.ReadCloser) {},
	HandleViewer:    func(info av.Info, writer av.WriteCloser) {},
	HandleCmdChunk:  func(info av.Info, vs []interface{}, chunk *core.ChunkStream) error { return nil },
}
