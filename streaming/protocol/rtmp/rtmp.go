package rtmp

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/errors"
	"github.com/viderstv/common/instance"
	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/protocol/amf"
	"github.com/viderstv/common/streaming/protocol/rtmp/core"
	"github.com/viderstv/common/utils/uid"
)

const (
	maxQueueNum  = 1024
	writeTimeout = 10 * time.Second
)

type Server struct {
	once     sync.Once
	ln       net.Listener
	wg       sync.WaitGroup
	shutdown chan struct{}
	config   Config
}

func New(config Config) instance.RtmpServer {
	return &Server{
		config:   config.fill(),
		shutdown: make(chan struct{}),
	}
}

// Shutsdown the server and waits for all connections to close.
func (s *Server) Shutdown() error {
	s.once.Do(func() {
		close(s.shutdown)
	})
	s.wg.Wait()
	return s.ln.Close()
}

func (s *Server) Serve(ln net.Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Error("rtmp serve panic: ", r)
			err = fmt.Errorf("%v", r)
		}
	}()

	s.ln = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return nil
			default:
				return err
			}
		}
		select {
		case <-s.shutdown:
			// if we shutdown ths server then stop accepting new connections.
			_ = conn.Close()
			continue
		default:
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			s.config.Logger.Error("panic in handleConn: ", err)
		}
		_ = conn.Close()
	}()

	info := av.Info{}
	connId := uid.NewId()
	addr := conn.RemoteAddr()
	if !s.config.OnNewStream(addr) {
		return
	}

	defer func() {
		s.config.OnStreamClose(info, addr)
	}()

	coreConn := core.NewConn(conn, 4*1024)
	connServer := core.NewConnServer(coreConn)

	if err := coreConn.HandshakeServer(); err != nil {
		return
	}

	mtx := sync.Mutex{}
	authed := make(chan struct{})

	connServer.SetCallbackAuth(func() error {
		mtx.Lock()
		select {
		case <-authed:
			return errors.ErrAlreadyAuthed
		default:
			close(authed)
		}
		mtx.Unlock()

		info = connServer.GetInfo()
		info.ID = connId
		info.Key = connId
		info.Publisher = connServer.IsPublisher()

		if !s.config.AuthStream(&info, addr) {
			return errors.ErrInvalidStreamKey
		}

		return nil
	})

	go func() {
		select {
		case <-time.After(s.config.AuthTimeout):
			mtx.Lock()
			close(authed)
			mtx.Unlock()
			_ = conn.Close()
		case <-authed:
		}
	}()

	if err := connServer.ReadMsg(); err != nil {
		return
	}

	decoder := &amf.Decoder{}

	if connServer.IsPublisher() {
		s.wg.Add(1)
		defer s.wg.Done()
		publisher := NewVirReader(connServer, s.config.Logger, info, func(chunk *core.ChunkStream) error {
			amfType := amf.AMF0
			if chunk.TypeID == 17 {
				chunk.Data = chunk.Data[1:]
			}

			vs, err := decoder.DecodeBatch(bytes.NewReader(chunk.Data), amf.Version(amfType))
			if err != nil && err != io.EOF {
				return err
			}
			return s.config.HandleCmdChunk(info, vs, chunk)
		})
		s.config.HandlePublisher(info, publisher)
	} else {
		viewer := NewVirWriter(connServer, s.config.Logger, info, func(chunk *core.ChunkStream) error {
			amfType := amf.AMF0
			if chunk.TypeID == 17 {
				chunk.Data = chunk.Data[1:]
			}

			vs, err := decoder.DecodeBatch(bytes.NewReader(chunk.Data), amf.Version(amfType))
			if err != nil && err != io.EOF {
				return err
			}
			return s.config.HandleCmdChunk(info, vs, chunk)
		})
		s.config.HandleViewer(info, viewer)
	}
}
