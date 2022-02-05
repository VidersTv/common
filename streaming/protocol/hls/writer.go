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
	tmp         *bytes.Buffer
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
	config = config.fill()
	s := &Source{
		info: info,

		RWBaser:     av.NewRWBaser(time.Second * 10),
		bWriter:     bytes.NewBuffer(make([]byte, 100*1024)),
		tmp:         bytes.NewBuffer(nil),
		currentItem: config.Cache.NewItem(),

		align: &align.Align{},
		stat:  &status.Status{},

		audioCache: cache.NewAudioCache(),
		demuxer:    flv.NewDemuxer(),
		muxer:      ts.NewMuxer(),

		segmentCache: config.Cache,
		tsParser:     parser.NewCodecParser(),

		closed: make(chan struct{}),

		packetQueue: make(chan *av.Packet, maxQueueNum),

		config: config,
	}
	go func() {
		defer s.Close()
		err := s.SendPacket()
		if err != nil {
			config.Logger.Error("send pkt: ", err)
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

func (s *Source) SendPacket() error {
	defer func() {
		s.config.Logger.Debug("hls sender stopped")
		if r := recover(); r != nil {
			s.config.Logger.Warn("hls SendPacket panic: ", r)
		}
	}()

	s.config.Logger.Debug("hls sender started")
	for p := range s.packetQueue {
		if p.IsMetadata {
			continue
		}

		err := s.demuxer.Demux(p)
		if err == flv.ErrAvcEndSEQ {
			s.config.Logger.Warn(err)
			continue
		} else {
			if err != nil {
				s.config.Logger.Warn(err)
				return err
			}
		}
		compositionTime, isSeq, err := s.parse(p)
		if err != nil {
			s.config.Logger.Warning(err)
		}

		if err != nil || isSeq {
			continue
		}

		if s.btsWriter != nil {
			s.stat.Update(p.IsVideo, p.TimeStamp)
			s.calcPtsDts(p.IsVideo, p.TimeStamp, uint32(compositionTime))
			_ = s.tsMux(p)
		}
	}

	return nil
}

func (s *Source) Info() av.Info {
	return s.info
}

func (s *Source) Close() error {
	s.once.Do(func() {
		s.config.Logger.Info("closed")
		close(s.closed)
		close(s.packetQueue)
		s.cut(true)
		s.segmentCache.Stop()
	})

	return nil
}

func (s *Source) cut(end bool) {
	_, err := s.currentItem.Write(s.btsWriter.Bytes())
	if err != nil {
		panic(err)
	}
	s.btsWriter.Reset()

	if end {
		err := s.flushAudio()
		if err != nil {
			s.config.Logger.Errorf("audio flush, err=%v", err)
		}

		s.currentItem.SetDuration(s.stat.Duration())
		_ = s.currentItem.Close()

		select {
		case <-s.closed:
			return
		default:
		}

		s.stat.ResetAndNew()
		s.currentItem = s.segmentCache.NewItem()
		s.btsWriter.Write(s.muxer.PAT())
		s.btsWriter.Write(s.muxer.PMT(av.SOUND_AAC, true))
	}
}

func (s *Source) parse(p *av.Packet) (int32, bool, error) {
	if s.btsWriter == nil {
		s.btsWriter = bytes.NewBuffer(nil)
		s.btsWriter.Write(s.muxer.PAT())
		s.btsWriter.Write(s.muxer.PMT(av.SOUND_AAC, true))
	}
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
		if vh.IsKeyFrame() && vh.IsSeq() {
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

	s.cut(p.IsVideo && vh.IsKeyFrame() && s.stat.Duration() >= s.config.MinSegmentDuration)

	return compositionTime, false, nil
}

func (s *Source) calcPtsDts(isVideo bool, ts, compositionTs uint32) {
	s.dts = uint64(ts) * align.H264DefaultHZ
	if isVideo {
		s.pts = s.dts + uint64(compositionTs)*align.H264DefaultHZ
	} else {
		sampleRate, _ := s.tsParser.SampleRate()
		s.align.Align(&s.dts, uint32(videoHZ*aacSampleLen/sampleRate))
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
		IsAudio:   true,
		Data:      buf,
		TimeStamp: uint32(pts / align.H264DefaultHZ),
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
