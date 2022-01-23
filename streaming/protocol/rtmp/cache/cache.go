package cache

import (
	"github.com/viderstv/common/streaming/av"
)

type Cache struct {
	gop      *GopCache
	videoSeq *SpecialCache
	audioSeq *SpecialCache
	metadata *SpecialCache
}

func NewCache() *Cache {
	return &Cache{
		gop:      NewGopCache(1),
		videoSeq: NewSpecialCache(),
		audioSeq: NewSpecialCache(),
		metadata: NewSpecialCache(),
	}
}

func (c *Cache) Write(p av.Packet) error {
	if p.IsMetadata {
		c.metadata.Write(&p)
		return nil
	} else {
		if !p.IsVideo {
			ah, ok := p.Header.(av.AudioPacketHeader)
			if ok {
				if ah.SoundFormat() == av.SOUND_AAC &&
					ah.AACPacketType() == av.AAC_SEQHDR {
					c.audioSeq.Write(&p)
					return nil
				} else {
					return nil
				}
			}

		} else {
			vh, ok := p.Header.(av.VideoPacketHeader)
			if ok {
				if vh.IsSeq() {
					c.videoSeq.Write(&p)
					return nil
				}
			} else {
				return nil
			}

		}
	}
	return c.gop.Write(&p)
}

func (c *Cache) Send(w av.WriteCloser) error {
	if err := c.metadata.Send(w); err != nil {
		return err
	}

	if err := c.videoSeq.Send(w); err != nil {
		return err
	}

	if err := c.audioSeq.Send(w); err != nil {
		return err
	}

	if err := c.gop.Send(w); err != nil {
		return err
	}

	return nil
}
