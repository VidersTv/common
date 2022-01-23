package cache

import (
	"bytes"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/protocol/amf"
)

const (
	SetDataFrame string = "@setDataFrame"
	OnMetaData   string = "onMetaData"
)

func init() {
	b := bytes.NewBuffer(nil)
	encoder := &amf.Encoder{}
	if _, err := encoder.Encode(b, SetDataFrame, amf.AMF0); err != nil {
		logrus.Fatal(err)
	}
}

type SpecialCache struct {
	full bool
	p    *av.Packet
}

func NewSpecialCache() *SpecialCache {
	return &SpecialCache{}
}

func (c *SpecialCache) Write(p *av.Packet) {
	c.p = p
	c.full = true
}

func (c *SpecialCache) Send(w av.WriteCloser) error {
	if !c.full {
		return nil
	}
	return w.Write(c.p)
}
