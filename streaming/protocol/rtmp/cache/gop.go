package cache

import (
	"fmt"

	"github.com/viderstv/common/streaming/av"
)

var (
	maxGOPCap    int = 1024
	ErrGopTooBig     = fmt.Errorf("gop to big")
)

type array struct {
	index   int
	packets []*av.Packet
}

func newArray() *array {
	ret := &array{
		index:   0,
		packets: make([]*av.Packet, 0, maxGOPCap),
	}
	return ret
}

func (a *array) reset() {
	a.index = 0
	a.packets = a.packets[:0]
}

func (a *array) write(packet *av.Packet) error {
	if a.index >= maxGOPCap {
		return ErrGopTooBig
	}
	a.packets = append(a.packets, packet)
	a.index++
	return nil
}

func (a *array) send(w av.WriteCloser) error {
	for i := 0; i < a.index; i++ {
		packet := a.packets[i]
		if err := w.Write(packet); err != nil {
			return err
		}
	}

	return nil
}

type GopCache struct {
	start     bool
	num       int
	count     int
	nextindex int
	gops      []*array
}

func NewGopCache(num int) *GopCache {
	if num == 0 {
		num = 1
	}
	return &GopCache{
		count: num,
		gops:  make([]*array, num),
	}
}

func (c *GopCache) writeToArray(chunk *av.Packet, startNew bool) error {
	var ginc *array
	if startNew {
		ginc = c.gops[c.nextindex]
		if ginc == nil {
			ginc = newArray()
			c.num++
			c.gops[c.nextindex] = ginc
		} else {
			ginc.reset()
		}
		c.nextindex = (c.nextindex + 1) % c.count
	} else {
		ginc = c.gops[(c.nextindex+1)%c.count]
	}

	return ginc.write(chunk)
}

func (c *GopCache) Write(p *av.Packet) error {
	var ok bool
	if p.IsVideo {
		vh := p.Header.(av.VideoPacketHeader)
		if vh.IsKeyFrame() && !vh.IsSeq() {
			ok = true
		}
	}
	if ok || c.start {
		c.start = true
		return c.writeToArray(p, ok)
	}
	return nil
}

func (c *GopCache) sendTo(w av.WriteCloser) error {
	var err error
	pos := (c.nextindex + 1) % c.count
	for i := 0; i < c.num; i++ {
		index := (pos - c.num + 1) + i
		if index < 0 {
			index += c.count
		}
		g := c.gops[index]
		err = g.send(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *GopCache) Send(w av.WriteCloser) error {
	return c.sendTo(w)
}
