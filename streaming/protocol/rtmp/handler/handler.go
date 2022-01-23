package handler

import (
	"sync"
	"time"

	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/protocol/rtmp/cache"
)

type RtmpHandler struct {
	streams map[string]*Stream
	mtx     sync.Mutex
}

func New() *RtmpHandler {
	ret := &RtmpHandler{
		streams: map[string]*Stream{},
	}
	go ret.checkAlive()
	return ret
}

func (h *RtmpHandler) HandleReader(r av.ReadCloser) {
	info := r.Info()

	h.mtx.Lock()
	stream := h.streams[info.Key]
	if stream != nil {
		stream.Stop()
		stream = newStream()
		h.streams[info.Key] = stream
	} else {
		stream = newStream()
		h.streams[info.Key] = stream
	}

	stream.info = info

	stream.AddReader(r)
	h.mtx.Unlock()
}

func (h *RtmpHandler) HandleWriter(w av.WriteCloser) {
	info := w.Info()

	h.mtx.Lock()
	stream := h.streams[info.Key]
	if stream == nil {
		stream = newStream()
		h.streams[info.Key] = stream
	} else {
		stream.AddWriter(w)
	}
	h.mtx.Unlock()
}

func (h *RtmpHandler) StopStream(key string) {
	h.mtx.Lock()
	if stream, ok := h.streams[key]; ok {
		stream.Stop()
		stream.StopWriters()
	}
	h.mtx.Unlock()
}

func (h *RtmpHandler) checkAlive() {
	for {
		time.Sleep(time.Second * 5)

		h.mtx.Lock()
		for k, v := range h.streams {
			if v.CheckAlive() == 0 {
				delete(h.streams, k)
			}
		}
		h.mtx.Unlock()
	}
}

type Stream struct {
	info  av.Info
	cache *cache.Cache

	reader     av.ReadCloser
	writers    map[string]*WriteCloser
	writersMtx sync.Mutex
}

type WriteCloser struct {
	av.WriteCloser
	init bool
}

func newStream() *Stream {
	return &Stream{
		cache:   cache.NewCache(),
		writers: map[string]*WriteCloser{},
	}
}

func (s *Stream) GetReader() av.ReadCloser {
	return s.reader
}

func (s *Stream) AddReader(r av.ReadCloser) {
	s.reader = r
	go s.Start()
}

func (s *Stream) AddWriter(w av.WriteCloser) {
	s.writersMtx.Lock()
	s.writers[w.Info().ID] = &WriteCloser{WriteCloser: w}
	s.writersMtx.Unlock()
}

func (s *Stream) Start() {
	var p av.Packet

	for {
		err := s.reader.Read(&p)
		if err != nil {
			s.Stop()
			return
		}

		err = s.cache.Write(p)
		if err != nil {
			s.Stop()
			return
		}

		s.writersMtx.Lock()
		for k, v := range s.writers {
			if v.init {
				newPacket := p
				if err = v.Write(&newPacket); err != nil {
					delete(s.writers, k)
				}
			} else if err = s.cache.Send(v); err != nil {
				delete(s.writers, k)
			} else {
				v.init = true
			}
		}
		s.writersMtx.Unlock()
	}
}

func (s *Stream) Stop() {
	if s.reader != nil {
		_ = s.reader.Close()
	}
}

func (s *Stream) StopWriters() {
	s.writersMtx.Lock()
	defer s.writersMtx.Unlock()
	for k, v := range s.writers {
		_ = v.Close()
		delete(s.writers, k)
	}
}

func (s *Stream) CheckAlive() int {
	n := 0
	if s.reader != nil {
		if s.reader.Alive() {
			n++
		} else {
			s.Stop()
			return 0
		}
	}

	s.writersMtx.Lock()
	defer s.writersMtx.Unlock()

	for k, v := range s.writers {
		if v.WriteCloser != nil {
			if v.WriteCloser.Alive() {
				n++
				continue
			}
		}
		delete(s.writers, k)
	}

	return n
}
