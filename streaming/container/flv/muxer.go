package flv

import (
	"os"
	"time"

	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/protocol/amf"
	"github.com/viderstv/common/utils/pio"
	"github.com/viderstv/common/utils/uid"
)

var (
	flvHeader = []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09}
)

/*
func NewFlv(handler av.Handler, info av.Info) {
	patths := strings.SplitN(info.Key, "/", 2)

	if len(patths) != 2 {
		log.Warning("invalid info")
		return
	}

	w, err := os.OpenFile(*flvFile, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		log.Error("open file error: ", err)
	}

	writer := NewFLVWriter(patths[0], patths[1], info.URL, w)

	handler.HandleWriter(writer)

	writer.Wait()
	// close flv file
	log.Debug("close flv file")
	writer.ctx.Close()
}
*/

const (
	headerLen = 11
)

type FLVWriter struct {
	Uid string
	av.RWBaser
	app, title, url string
	buf             []byte
	closed          chan struct{}
	ctx             *os.File
	closedWriter    bool
}

func NewFLVWriter(app, title, url string, ctx *os.File) *FLVWriter {
	ret := &FLVWriter{
		Uid:     uid.NewId(),
		app:     app,
		title:   title,
		url:     url,
		ctx:     ctx,
		RWBaser: av.NewRWBaser(time.Second * 10),
		closed:  make(chan struct{}),
		buf:     make([]byte, headerLen),
	}

	_, _ = ret.ctx.Write(flvHeader)
	pio.PutI32BE(ret.buf[:4], 0)
	_, _ = ret.ctx.Write(ret.buf[:4])

	return ret
}

func (w *FLVWriter) Write(p *av.Packet) error {
	w.RWBaser.SetPreTime()
	h := w.buf[:headerLen]
	typeID := av.TAG_VIDEO
	if !p.IsVideo {
		if p.IsMetadata {
			var err error
			typeID = av.TAG_SCRIPTDATAAMF0
			p.Data, err = amf.MetaDataReform(p.Data, amf.DEL)
			if err != nil {
				return err
			}
		} else {
			typeID = av.TAG_AUDIO
		}
	}
	dataLen := len(p.Data)
	timestamp := p.TimeStamp
	timestamp += w.BaseTimeStamp()
	w.RWBaser.RecTimeStamp(timestamp, uint32(typeID))

	preDataLen := dataLen + headerLen
	timestampbase := timestamp & 0xffffff
	timestampExt := timestamp >> 24 & 0xff

	pio.PutU8(h[0:1], uint8(typeID))
	pio.PutI24BE(h[1:4], int32(dataLen))
	pio.PutI24BE(h[4:7], int32(timestampbase))
	pio.PutU8(h[7:8], uint8(timestampExt))

	if _, err := w.ctx.Write(h); err != nil {
		return err
	}

	if _, err := w.ctx.Write(p.Data); err != nil {
		return err
	}

	pio.PutI32BE(h[:4], int32(preDataLen))
	if _, err := w.ctx.Write(h[:4]); err != nil {
		return err
	}

	return nil
}

func (w *FLVWriter) Wait() <-chan struct{} {
	return w.closed
}

func (w *FLVWriter) Close() error {
	if w.closedWriter {
		return nil
	}

	w.closedWriter = true
	close(w.closed)

	return w.ctx.Close()
}

func (w *FLVWriter) Info() (ret av.Info) {
	ret.ID = w.Uid
	ret.URL = w.url
	ret.Key = w.app + "/" + w.title
	return
}
