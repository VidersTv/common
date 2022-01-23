package core

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/viderstv/common/utils/pio"
	"github.com/viderstv/common/utils/pool"
)

const (
	_ = iota
	idSetChunkSize
	_
	idAck
	_
	idWindowAckSize
	idSetPeerBandwidth
)

type Conn struct {
	net.Conn
	chunkSize           uint32
	remoteChunkSize     uint32
	windowAckSize       uint32
	remoteWindowAckSize uint32
	received            uint32
	ackReceived         uint32
	rw                  *ReadWriter
	pool                *pool.Pool
	chunks              map[uint32]ChunkStream
}

func NewConn(chunk net.Conn, bufferSize int) *Conn {
	return &Conn{
		Conn:                chunk,
		chunkSize:           128,
		remoteChunkSize:     128,
		windowAckSize:       2500000,
		remoteWindowAckSize: 2500000,
		pool:                pool.NewPool(),
		rw:                  NewReadWriter(chunk, bufferSize),
		chunks:              make(map[uint32]ChunkStream),
	}
}

func (c *Conn) Read(chunk *ChunkStream) error {
	for {
		h, _ := c.rw.ReadUintBE(1)
		format := h >> 6
		csid := h & 0x3f
		cs, ok := c.chunks[csid]
		if !ok {
			cs = ChunkStream{}
			c.chunks[csid] = cs
		}
		cs.tmpFromat = format
		cs.CSID = csid
		err := cs.readChunk(c.rw, c.remoteChunkSize, c.pool)
		if err != nil {
			return err
		}
		c.chunks[csid] = cs
		if cs.full() {
			*chunk = cs
			break
		}
	}

	c.handleControlMsg(chunk)

	return c.ack(chunk.Length)
}

func (c *Conn) Write(chunk *ChunkStream) error {
	if chunk.TypeID == idSetChunkSize {
		c.chunkSize = binary.BigEndian.Uint32(chunk.Data)
	}
	return chunk.writeChunk(c.rw, int(c.chunkSize))
}

func (c *Conn) Flush() error {
	return c.rw.Flush()
}

func (c *Conn) Close() error {

	return c.Conn.Close()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

func (c *Conn) SetDeadline(t time.Time) error {
	return c.Conn.SetDeadline(t)
}

func (c *Conn) NewAck(size uint32) ChunkStream {
	return initControlMsg(idAck, 4, size)
}

func (c *Conn) NewSetChunkSize(size uint32) ChunkStream {
	return initControlMsg(idSetChunkSize, 4, size)
}

func (c *Conn) NewWindowAckSize(size uint32) ChunkStream {
	return initControlMsg(idWindowAckSize, 4, size)
}

func (c *Conn) NewSetPeerBandwidth(size uint32) ChunkStream {
	ret := initControlMsg(idSetPeerBandwidth, 5, size)
	ret.Data[4] = 2
	return ret
}

func (c *Conn) handleControlMsg(chunk *ChunkStream) {
	if chunk.TypeID == idSetChunkSize {
		c.remoteChunkSize = binary.BigEndian.Uint32(chunk.Data)
	} else if chunk.TypeID == idWindowAckSize {
		c.remoteWindowAckSize = binary.BigEndian.Uint32(chunk.Data)
	}
}

func (c *Conn) ack(size uint32) error {
	c.received += uint32(size)
	c.ackReceived += uint32(size)
	if c.received >= 0xf0000000 {
		c.received = 0
	}
	if c.ackReceived >= c.remoteWindowAckSize {
		cs := c.NewAck(c.ackReceived)
		err := cs.writeChunk(c.rw, int(c.chunkSize))
		if err != nil {
			return err
		}
		c.ackReceived = 0
	}
	return nil
}

func initControlMsg(id, size, value uint32) ChunkStream {
	ret := ChunkStream{
		Format:   0,
		CSID:     2,
		TypeID:   id,
		StreamID: 0,
		Length:   size,
		Data:     make([]byte, size),
	}
	pio.PutU32BE(ret.Data[:size], value)
	return ret
}

const (
	streamBegin      uint32 = 0
	streamIsRecorded uint32 = 4
	// pingRequest      uint32 = 6
	// pingResponse     uint32 = 7
)

/*
   +------------------------------+-------------------------
   |     Event Type ( 2- bytes )  | Event Data
   +------------------------------+-------------------------
   Pay load for the ‘User Control Message’.
*/
func (c *Conn) userControlMsg(eventType, buflen uint32) ChunkStream {
	var ret ChunkStream
	buflen += 2
	ret = ChunkStream{
		Format:   0,
		CSID:     2,
		TypeID:   4,
		StreamID: 1,
		Length:   buflen,
		Data:     make([]byte, buflen),
	}
	ret.Data[0] = byte(eventType >> 8 & 0xff)
	ret.Data[1] = byte(eventType & 0xff)
	return ret
}

func (c *Conn) SetBegin() error {
	ret := c.userControlMsg(streamBegin, 4)
	for i := 0; i < 4; i++ {
		ret.Data[2+i] = byte(1 >> uint32((3-i)*8) & 0xff)
	}
	return c.Write(&ret)
}

func (c *Conn) SetRecorded() error {
	ret := c.userControlMsg(streamIsRecorded, 4)
	for i := 0; i < 4; i++ {
		ret.Data[2+i] = byte(1 >> uint32((3-i)*8) & 0xff)
	}
	return c.Write(&ret)
}
