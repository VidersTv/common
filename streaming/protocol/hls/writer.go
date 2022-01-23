package hls

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/errors"
	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/container/flv"
	"github.com/viderstv/common/streaming/container/ts"
	"github.com/viderstv/common/streaming/parser"
	"github.com/viderstv/common/streaming/protocol/hls/align"
	"github.com/viderstv/common/streaming/protocol/hls/cache"
	"github.com/viderstv/common/streaming/protocol/hls/item"
	"github.com/viderstv/common/streaming/protocol/hls/status"
)

const (
	videoHZ      = 90000
	aacSampleLen = 1024
	maxQueueNum  = 512
)

type Source struct {
	av.RWBaser

	info av.Info

	bWriter     *bytes.Buffer
	btsWriter   *bytes.Buffer
	currentItem *item.Item

	demuxer *flv.Demuxer
	muxer   *ts.Muxer

	pts, dts uint64

	stat  *status.Status
	align *align.Align

	audioCache   *cache.AudioCache
	segmentCache *cache.Cache

	tsParser *parser.CodecParser

	once   sync.Once
	closed chan struct{}

	packetQueue chan *av.Packet

	config Config
}

func New(info av.Info, config Config) av.WriteCloser {
	s := &Source{
		info: av.Info{},

		RWBaser: av.NewRWBaser(time.Second * 10),
		bWriter: bytes.NewBuffer(make([]byte, 100*1024)),

		align: &align.Align{},
		stat:  &status.Status{},

		audioCache: cache.NewAudioCache(),
		demuxer:    flv.NewDemuxer(),
		muxer:      ts.NewMuxer(),

		segmentCache: cache.New(),
		tsParser:     parser.NewCodecParser(),

		closed: make(chan struct{}),

		packetQueue: make(chan *av.Packet, maxQueueNum),

		config: config,
	}
	go func() {
		err := s.SendPacket()
		if err != nil {
			s.Close()
		}
	}()
	return s
}

func (s *Source) Running() <-chan struct{} {
	return s.closed
}

func (s *Source) GetCache() *cache.Cache {
	return s.segmentCache
}

func (s *Source) Write(p *av.Packet) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("hls source has already been closed:%v", e)
		}
	}()

	s.SetPreTime()
	select {
	case <-s.closed:
		return errors.ErrSourceClosed
	case s.packetQueue <- p:
	default:
		isKeyFrame := false
		if p.IsVideo {
			pkt, _ := p.Header.(av.VideoPacketHeader)
			isKeyFrame = pkt.IsKeyFrame()
		}
		s.config.Logger.WithFields(logrus.Fields{
			"is_video":    p.IsVideo,
			"is_audio":    p.IsAudio,
			"is_metadata": p.IsMetadata,
			"is_keyframe": isKeyFrame,
			"info":        s.Info(),
		}).Warn("dropping packet")
	}

	return
}

func (s *Source) SendPacket() (err error) {
	defer func() {
		s.config.Logger.Debugf("[%v] hls sender stop", s.info)
		if r := recover(); r != nil {
			s.config.Logger.Warn("hls SendPacket panic: ", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	s.config.Logger.Debugf("[%v] hls sender start", s.info)
	for {
		select {
		case <-s.closed:
			return errors.ErrSourceClosed
		case p := <-s.packetQueue:
			err := s.demuxer.Demux(p)
			if err == flv.ErrAvcEndSEQ {
				continue
			} else if err != nil {
				return err
			}
			compositionTime, isSeq, err := s.parse(p)
			if err != nil || isSeq {
				continue
			}
			if s.btsWriter != nil {
				s.stat.Update(p.IsVideo, p.TimeStamp)
				s.calcPtsDts(p.IsVideo, p.TimeStamp, uint32(compositionTime))
				err = s.tsMux(p)
				if err != nil {
					s.config.Logger.Errorf("ts mux, err=%v", err)
				}
			}
		}
	}
}

func (s *Source) Info() av.Info {
	return s.info
}

func (s *Source) Close() error {
	s.once.Do(func() {
		close(s.closed)
		close(s.packetQueue)
	})

	return nil
}

func (s *Source) cut(end bool) {
	if s.btsWriter == nil {
		s.btsWriter = bytes.NewBuffer(nil)
	} else {
		err := s.flushAudio()
		if err != nil {
			s.config.Logger.Errorf("audio flush, err=%v", err)
		}

		src := s.btsWriter.Bytes()
		if s.currentItem == nil {
			s.currentItem = s.segmentCache.NewItem()
		}
		s.currentItem.SetDuration(s.stat.Duration())
		_, _ = s.currentItem.Write(src)
		if end {
			_ = s.currentItem.Close()
			s.currentItem = nil
		}
	}
	if end {
		s.btsWriter.Reset()
		s.stat.ResetAndNew()
		s.btsWriter.Write(s.muxer.PAT())
		s.btsWriter.Write(s.muxer.PMT(av.SOUND_AAC, true))
	}
}

func (s *Source) parse(p *av.Packet) (int32, bool, error) {
	var (
		compositionTime int32
		ah              av.AudioPacketHeader
		vh              av.VideoPacketHeader
	)

	if p.IsVideo {
		vh = p.Header.(av.VideoPacketHeader)
		if vh.CodecID() != av.VIDEO_H264 {
			return compositionTime, false, errors.ErrNoSupportVideoCodec
		}
		compositionTime = vh.CompositionTime()
		if vh.IsSeq() {
			return compositionTime, true, s.tsParser.Parse(p, s.bWriter)
		}
	} else {
		ah = p.Header.(av.AudioPacketHeader)
		if ah.SoundFormat() != av.SOUND_AAC {
			return compositionTime, false, errors.ErrNoSupportAudioCodec
		}
		if ah.AACPacketType() == av.AAC_SEQHDR {
			return compositionTime, true, s.tsParser.Parse(p, s.bWriter)
		}
	}

	s.bWriter.Reset()
	if err := s.tsParser.Parse(p, s.bWriter); err != nil {
		return compositionTime, false, err
	}

	p.Data = s.bWriter.Bytes()

	if p.IsVideo {
		s.cut(s.stat.Duration() >= s.config.MinSegmentDuration && vh.IsKeyFrame())
	}

	return compositionTime, false, nil
}

func (s *Source) calcPtsDts(isVideo bool, ts, compositionTs uint32) {
	s.dts = uint64(ts) * align.H264_default_hz
	if isVideo {
		s.pts = s.dts + uint64(compositionTs)*align.H264_default_hz
	} else {
		sampleRate, _ := s.tsParser.SampleRate()
		s.dts = s.align.Align(s.dts, uint32(videoHZ*aacSampleLen/sampleRate))
		s.pts = s.dts
	}
}

func (s *Source) flushAudio() error {
	return s.muxAudio(1)
}

func (s *Source) muxAudio(limit byte) error {
	if s.audioCache.CacheNum() < limit {
		return nil
	}
	_, pts, buf := s.audioCache.GetFrame()
	return s.muxer.Mux(&av.Packet{
		Data:      buf,
		TimeStamp: uint32(pts / align.H264_default_hz),
	}, s.btsWriter)
}

func (s *Source) tsMux(p *av.Packet) error {
	if p.IsVideo {
		return s.muxer.Mux(p, s.btsWriter)
	} else {
		s.audioCache.Cache(p.Data, s.pts)
		return s.muxAudio(cache.AudioCacheMaxFrames)
	}
}
