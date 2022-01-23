package core

import (
	"encoding/binary"
	"fmt"

	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/utils/pool"
)

type ChunkStream struct {
	Format    uint32
	CSID      uint32
	Timestamp uint32
	Length    uint32
	TypeID    uint32
	StreamID  uint32
	timeDelta uint32
	exted     bool
	index     uint32
	remain    uint32
	got       bool
	tmpFromat uint32
	Data      []byte
}

func (c *ChunkStream) full() bool {
	return c.got
}

func (c *ChunkStream) new(pool *pool.Pool) {
	c.got = false
	c.index = 0
	c.remain = c.Length
	c.Data = pool.Get(int(c.Length))
}

func (c *ChunkStream) writeHeader(w *ReadWriter) error {
	//Chunk Basic Header
	var err error
	h := c.Format << 6
	switch {
	case c.CSID < 64:
		h |= c.CSID
		err = w.WriteUintBE(h, 1)
		if err != nil {
			return err
		}
	case c.CSID-64 < 256:
		h |= 0
		err = w.WriteUintBE(h, 1)
		if err != nil {
			return err
		}
		err = w.WriteUintLE(c.CSID-64, 1)
		if err != nil {
			return err
		}
	case c.CSID-64 < 65536:
		h |= 1
		err = w.WriteUintBE(h, 1)
		if err != nil {
			return err
		}
		err = w.WriteUintLE(c.CSID-64, 2)
		if err != nil {
			return err
		}
	}
	//Chunk Message Header
	ts := c.Timestamp
	if c.Format == 3 {
		goto END
	}
	if c.Timestamp > 0xffffff {
		ts = 0xffffff
	}
	err = w.WriteUintBE(ts, 3)
	if err != nil {
		return err
	}
	if c.Format == 2 {
		goto END
	}
	if c.Length > 0xffffff {
		return fmt.Errorf("length=%d", c.Length)
	}
	err = w.WriteUintBE(c.Length, 3)
	if err != nil {
		return err
	}
	err = w.WriteUintBE(c.TypeID, 1)
	if err != nil {
		return err
	}
	if c.Format == 1 {
		goto END
	}
	err = w.WriteUintLE(c.StreamID, 4)
	if err != nil {
		return err
	}
END:
	//Extended Timestamp
	if ts >= 0xffffff {
		err = w.WriteUintBE(c.Timestamp, 4)
		if err != nil {
			return err
		}
	}
	return w.WriteError()
}

func (c *ChunkStream) writeChunk(w *ReadWriter, chunkSize int) error {
	if c.TypeID == av.TAG_AUDIO {
		c.CSID = 4
	} else if c.TypeID == av.TAG_VIDEO ||
		c.TypeID == av.TAG_SCRIPTDATAAMF0 ||
		c.TypeID == av.TAG_SCRIPTDATAAMF3 {
		c.CSID = 6
	}

	totalLen := uint32(0)
	numChunks := (c.Length / uint32(chunkSize))
	for i := uint32(0); i <= numChunks; i++ {
		if totalLen == c.Length {
			break
		}
		if i == 0 {
			c.Format = uint32(0)
		} else {
			c.Format = uint32(3)
		}
		if err := c.writeHeader(w); err != nil {
			return err
		}
		inc := uint32(chunkSize)
		start := uint32(i) * uint32(chunkSize)
		if uint32(len(c.Data))-start <= inc {
			inc = uint32(len(c.Data)) - start
		}
		totalLen += inc
		end := start + inc
		buf := c.Data[start:end]
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}

	return nil

}

func (c *ChunkStream) readChunk(r *ReadWriter, chunkSize uint32, pool *pool.Pool) error {
	if c.remain != 0 && c.tmpFromat != 3 {
		return fmt.Errorf("invalid remain = %d", c.remain)
	}
	switch c.CSID {
	case 0:
		id, _ := r.ReadUintLE(1)
		c.CSID = id + 64
	case 1:
		id, _ := r.ReadUintLE(2)
		c.CSID = id + 64
	}

	switch c.tmpFromat {
	case 0:
		c.Format = c.tmpFromat
		c.Timestamp, _ = r.ReadUintBE(3)
		c.Length, _ = r.ReadUintBE(3)
		c.TypeID, _ = r.ReadUintBE(1)
		c.StreamID, _ = r.ReadUintLE(4)
		if c.Timestamp == 0xffffff {
			c.Timestamp, _ = r.ReadUintBE(4)
			c.exted = true
		} else {
			c.exted = false
		}
		c.new(pool)
	case 1:
		c.Format = c.tmpFromat
		timeStamp, _ := r.ReadUintBE(3)
		c.Length, _ = r.ReadUintBE(3)
		c.TypeID, _ = r.ReadUintBE(1)
		if timeStamp == 0xffffff {
			timeStamp, _ = r.ReadUintBE(4)
			c.exted = true
		} else {
			c.exted = false
		}
		c.timeDelta = timeStamp
		c.Timestamp += timeStamp
		c.new(pool)
	case 2:
		c.Format = c.tmpFromat
		timeStamp, _ := r.ReadUintBE(3)
		if timeStamp == 0xffffff {
			timeStamp, _ = r.ReadUintBE(4)
			c.exted = true
		} else {
			c.exted = false
		}
		c.timeDelta = timeStamp
		c.Timestamp += timeStamp
		c.new(pool)
	case 3:
		if c.remain == 0 {
			switch c.Format {
			case 0:
				if c.exted {
					timestamp, _ := r.ReadUintBE(4)
					c.Timestamp = timestamp
				}
			case 1, 2:
				var timedet uint32
				if c.exted {
					timedet, _ = r.ReadUintBE(4)
				} else {
					timedet = c.timeDelta
				}
				c.Timestamp += timedet
			}
			c.new(pool)
		} else {
			if c.exted {
				b, err := r.Peek(4)
				if err != nil {
					return err
				}
				tmpts := binary.BigEndian.Uint32(b)
				if tmpts == c.Timestamp {
					_, err = r.Discard(4)
					if err != nil {
						return err
					}
				}
			}
		}
	default:
		return fmt.Errorf("invalid format=%d", c.Format)
	}
	size := int(c.remain)
	if size > int(chunkSize) {
		size = int(chunkSize)
	}

	buf := c.Data[c.index : c.index+uint32(size)]
	if _, err := r.Read(buf); err != nil {
		return err
	}
	c.index += uint32(size)
	c.remain -= uint32(size)
	if c.remain == 0 {
		c.got = true
	}

	return r.readError
}
