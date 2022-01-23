package parser

import (
	"fmt"
	"io"

	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/parser/aac"
	"github.com/viderstv/common/streaming/parser/h264"
)

var (
	errNoAudio = fmt.Errorf("demuxer no audio")
)

type CodecParser struct {
	aac  *aac.Parser
	h264 *h264.Parser
}

func NewCodecParser() *CodecParser {
	return &CodecParser{
		aac:  aac.NewParser(),
		h264: h264.NewParser(),
	}
}

func (p *CodecParser) SampleRate() (int, error) {
	if p.aac == nil {
		return 0, errNoAudio
	}
	return p.aac.SampleRate(), nil
}

func (p *CodecParser) Parse(pkt *av.Packet, w io.Writer) error {
	if pkt.IsVideo {
		f, ok := pkt.Header.(av.VideoPacketHeader)
		if ok {
			if f.CodecID() == av.VIDEO_H264 {
				return p.h264.Parse(pkt.Data, f.IsSeq(), w)
			}
		}
	} else {
		f, ok := pkt.Header.(av.AudioPacketHeader)
		if ok {
			if f.SoundFormat() == av.SOUND_AAC {
				return p.aac.Parse(pkt.Data, f.AACPacketType(), w)
			}
		}
	}

	return fmt.Errorf("invalid type")
}
