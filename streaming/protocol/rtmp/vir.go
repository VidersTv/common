package rtmp

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/container/flv"
	"github.com/viderstv/common/streaming/protocol/rtmp/core"
)

type Stats struct {
	VideoDataInBytes uint64
	AudioDataInBytes uint64
}

type VirWriter struct {
	av.RWBaser

	info av.Info

	logger logrus.FieldLogger

	closed chan struct{}
	once   sync.Once

	conn        VirConnWriter
	packetQueue chan *av.Packet

	handleCmdMsg func(chunk *core.ChunkStream) error
}

type VirConnReader interface {
	Read(*core.ChunkStream) error
	Close() error
}

type VirConnWriter interface {
	Read(*core.ChunkStream) error
	Write(core.ChunkStream) error
	Flush() error
	Close() error
}

func NewVirWriter(conn VirConnWriter, logger logrus.FieldLogger, info av.Info, handleCmdMsg func(chunk *core.ChunkStream) error) *VirWriter {
	ret := &VirWriter{
		info:         info,
		closed:       make(chan struct{}),
		conn:         conn,
		logger:       logger,
		RWBaser:      av.NewRWBaser(writeTimeout),
		packetQueue:  make(chan *av.Packet, maxQueueNum),
		handleCmdMsg: handleCmdMsg,
	}

	go ret.Check()
	go func() {
		if err := ret.SendPacket(); err != nil {
			logger.Warnf("rtmp send packet, err=%v", err)
		}
	}()
	return ret
}

func (v *VirWriter) ToAvHandler() av.WriteCloser {
	return v
}

func (v *VirWriter) Running() <-chan struct{} {
	return v.closed
}

func (v *VirWriter) Check() {
	c := core.ChunkStream{}
	for {
		if err := v.conn.Read(&c); err != nil {
			_ = v.Close()
			return
		}
		if v.handleCmdMsg != nil && (c.TypeID == 20 || c.TypeID == 17) {
			if err := v.handleCmdMsg(&c); err != nil {
				_ = v.Close()
				return
			}
		}
	}
}

func (v *VirWriter) Write(p *av.Packet) (err error) {
	err = nil

	select {
	case <-v.closed:
		return fmt.Errorf("VirWriter closed")
	default:
	}
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("VirWriter has already been closed: %v", e)
		}
	}()

	select {
	case v.packetQueue <- p:
	default:
		<-v.packetQueue
		v.packetQueue <- p
		isKeyFrame := false
		if p.IsVideo {
			pkt, _ := p.Header.(av.VideoPacketHeader)
			isKeyFrame = pkt.IsKeyFrame()
		}
		v.logger.WithFields(logrus.Fields{
			"is_video":    p.IsVideo,
			"is_audio":    p.IsAudio,
			"is_metadata": p.IsMetadata,
			"is_keyframe": isKeyFrame,
			"info":        v.Info(),
		}).Warn("dropping packet")
	}

	return
}

func (v *VirWriter) SendPacket() error {
	cs := core.ChunkStream{}

	for p := range v.packetQueue {
		cs.Data = p.Data
		cs.Length = uint32(len(p.Data))
		cs.StreamID = p.StreamID
		cs.Timestamp = p.TimeStamp + v.BaseTimeStamp()

		if p.IsVideo {
			cs.TypeID = av.TAG_VIDEO
		} else {
			if p.IsMetadata {
				cs.TypeID = av.TAG_SCRIPTDATAAMF0
			} else {
				cs.TypeID = av.TAG_AUDIO
			}
		}

		v.SetPreTime()
		v.RecTimeStamp(cs.Timestamp, cs.TypeID)
		if err := v.conn.Write(cs); err != nil {
			_ = v.Close()
			return err
		}
		if err := v.conn.Flush(); err != nil {
			_ = v.Close()
			return err
		}
	}

	return nil
}

func (v *VirWriter) Info() av.Info {
	return v.info
}

func (v *VirWriter) Close() error {
	v.once.Do(func() {
		close(v.closed)
		close(v.packetQueue)
	})

	return v.conn.Close()
}

type VirReader struct {
	av.RWBaser

	logger logrus.FieldLogger

	info av.Info

	demuxer *flv.Demuxer
	conn    VirConnReader

	once   sync.Once
	closed chan struct{}

	handleCmdMsg func(chunk *core.ChunkStream) error

	stats Stats
}

func NewVirReader(conn VirConnReader, logger logrus.FieldLogger, info av.Info, handleCmdMsg func(chunk *core.ChunkStream) error) *VirReader {
	return &VirReader{
		conn:         conn,
		info:         info,
		logger:       logger,
		RWBaser:      av.NewRWBaser(writeTimeout),
		demuxer:      flv.NewDemuxer(),
		closed:       make(chan struct{}),
		handleCmdMsg: handleCmdMsg,
	}
}

func (v *VirReader) Stats() Stats {
	return v.stats
}

func (v *VirReader) SaveStatics(length uint64, isVideoFlag bool) {
	if isVideoFlag {
		v.stats.VideoDataInBytes += length
	} else {
		v.stats.AudioDataInBytes += length
	}
}

func (v *VirReader) Read(p *av.Packet) (err error) {
	defer func() {
		if r := recover(); r != nil {
			v.logger.Warn("rtmp read packet panic: ", r)
		}
	}()

	v.SetPreTime()
	cs := core.ChunkStream{}
	for {
		err = v.conn.Read(&cs)
		if err != nil {
			return err
		}

		if v.handleCmdMsg != nil && (cs.TypeID == 20 || cs.TypeID == 17) {
			if err := v.handleCmdMsg(&cs); err != nil {
				_ = v.Close()
				return err
			}
		}
		if cs.TypeID == av.TAG_AUDIO ||
			cs.TypeID == av.TAG_VIDEO ||
			cs.TypeID == av.TAG_SCRIPTDATAAMF0 ||
			cs.TypeID == av.TAG_SCRIPTDATAAMF3 {
			break
		}
	}

	p.IsAudio = cs.TypeID == av.TAG_AUDIO
	p.IsVideo = cs.TypeID == av.TAG_VIDEO
	p.IsMetadata = cs.TypeID == av.TAG_SCRIPTDATAAMF0 || cs.TypeID == av.TAG_SCRIPTDATAAMF3
	p.StreamID = cs.StreamID
	p.Data = cs.Data
	p.TimeStamp = cs.Timestamp

	v.SaveStatics(uint64(len(p.Data)), p.IsVideo)

	return v.demuxer.DemuxH(p)
}

func (v *VirReader) Info() av.Info {
	return v.info
}

func (v *VirReader) Close() error {
	v.once.Do(func() {
		close(v.closed)
	})

	return v.conn.Close()
}

func (v *VirReader) Running() <-chan struct{} {
	return v.closed
}

func (v *VirReader) ToAvHandler() av.ReadCloser {
	return v
}
