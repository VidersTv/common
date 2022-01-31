package parser

import (
	"io"

	"github.com/viderstv/common/errors"
	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/parser/aac"
	"github.com/viderstv/common/streaming/parser/h264"
	"github.com/viderstv/common/streaming/parser/mp3"
)

type CodecParser struct {
	aac  *aac.Parser
	mp3  *mp3.Parser
	h264 *h264.Parser
}

func NewCodecParser() *CodecParser {
	return &CodecParser{}
}

func (c *CodecParser) SampleRate() (int, error) {
	if c.aac == nil && c.mp3 == nil {
		return 0, errors.ErrNoAudio
	}
	if c.aac != nil {
		return c.aac.SampleRate(), nil
	}
	return c.mp3.SampleRate(), nil
}

func (c *CodecParser) Parse(p *av.Packet, w io.Writer) error {
	if p.IsVideo {
		f := p.Header.(av.VideoPacketHeader)
		if f.CodecID() == av.VIDEO_H264 {
			if c.h264 == nil {
				c.h264 = h264.NewParser()
			}
			return c.h264.Parse(p.Data, f.IsSeq(), w)
		}
		return errors.ErrNoSupportVideoCodec
	} else {
		f := p.Header.(av.AudioPacketHeader)
		switch f.SoundFormat() {
		case av.SOUND_AAC:
			if c.aac == nil {
				c.aac = aac.NewParser()
			}
			return c.aac.Parse(p.Data, f.AACPacketType(), w)
		case av.SOUND_MP3:
			if c.mp3 == nil {
				c.mp3 = mp3.NewParser()
			}
			return c.mp3.Parse(p.Data)
		}
		return errors.ErrNoSupportAudioCodec
	}
}
