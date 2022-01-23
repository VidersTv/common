package core

import (
	"bytes"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/protocol/amf"
)

var (
	ErrReq = fmt.Errorf("req error")
)

var (
	cmdConnect       = "connect"
	cmdFcpublish     = "FCPublish"
	cmdReleaseStream = "releaseStream"
	cmdCreateStream  = "createStream"
	cmdPublish       = "publish"
	cmdFCUnpublish   = "FCUnpublish"
	cmdDeleteStream  = "deleteStream"
	cmdPlay          = "play"
)

type ConnectInfo struct {
	App            string `amf:"app" json:"app"`
	Flashver       string `amf:"flashVer" json:"flashVer"`
	SwfUrl         string `amf:"swfUrl" json:"swfUrl"`
	TcUrl          string `amf:"tcUrl" json:"tcUrl"`
	Fpad           bool   `amf:"fpad" json:"fpad"`
	AudioCodecs    int    `amf:"audioCodecs" json:"audioCodecs"`
	VideoCodecs    int    `amf:"videoCodecs" json:"videoCodecs"`
	VideoFunction  int    `amf:"videoFunction" json:"videoFunction"`
	PageUrl        string `amf:"pageUrl" json:"pageUrl"`
	ObjectEncoding int    `amf:"objectEncoding" json:"objectEncoding"`
}

type ConnectResp struct {
	FMSVer       string `amf:"fmsVer"`
	Capabilities int    `amf:"capabilities"`
}

type ConnectEvent struct {
	Level          string `amf:"level"`
	Code           string `amf:"code"`
	Description    string `amf:"description"`
	ObjectEncoding int    `amf:"objectEncoding"`
}

type PublishInfo struct {
	Name string
	Type string
}

type ConnServer struct {
	done          bool
	streamID      int
	isPublisher   bool
	conn          *Conn
	transactionID int
	ConnInfo      ConnectInfo
	PublishInfo   PublishInfo
	decoder       *amf.Decoder
	encoder       *amf.Encoder
	bytesw        *bytes.Buffer
	cb            func() error
}

func NewConnServer(conn *Conn) *ConnServer {
	return &ConnServer{
		conn:     conn,
		streamID: 1,
		bytesw:   bytes.NewBuffer(nil),
		decoder:  &amf.Decoder{},
		encoder:  &amf.Encoder{},
	}
}

func (c *ConnServer) writeMsg(csid, streamID uint32, args ...interface{}) error {
	c.bytesw.Reset()
	for _, v := range args {
		if _, err := c.encoder.Encode(c.bytesw, v, amf.AMF0); err != nil {
			return err
		}
	}
	msg := c.bytesw.Bytes()
	chunk := ChunkStream{
		Format:    0,
		CSID:      csid,
		Timestamp: 0,
		TypeID:    20,
		StreamID:  streamID,
		Length:    uint32(len(msg)),
		Data:      msg,
	}
	err := c.conn.Write(&chunk)
	if err != nil {
		return err
	}
	return c.conn.Flush()
}

func (c *ConnServer) connect(vs []interface{}) error {
	for _, v := range vs {
		switch m := v.(type) {
		case string:
		case float64:
			id := int(m)
			if id != 1 {
				return ErrReq
			}
			c.transactionID = id
		case amf.Object:
			obimap := v.(amf.Object)
			if app, ok := obimap["app"]; ok {
				c.ConnInfo.App = app.(string)
			}
			if flashVer, ok := obimap["flashVer"]; ok {
				c.ConnInfo.Flashver = flashVer.(string)
			}
			if tcurl, ok := obimap["tcUrl"]; ok {
				c.ConnInfo.TcUrl = tcurl.(string)
			}
			if encoding, ok := obimap["objectEncoding"]; ok {
				c.ConnInfo.ObjectEncoding = int(encoding.(float64))
			}
		}
	}
	return nil
}

func (c *ConnServer) releaseStream(vs []interface{}) error {
	return nil
}

func (c *ConnServer) fcPublish(vs []interface{}) error {
	return nil
}

func (c *ConnServer) SetCallbackAuth(callback func() error) {
	c.cb = callback
}

func (c *ConnServer) connectResp(cur *ChunkStream) error {
	chunk := c.conn.NewWindowAckSize(2500000)
	err := c.conn.Write(&chunk)
	if err != nil {
		return err
	}
	chunk = c.conn.NewSetPeerBandwidth(2500000)
	err = c.conn.Write(&chunk)
	if err != nil {
		return err
	}
	chunk = c.conn.NewSetChunkSize(uint32(1024))
	err = c.conn.Write(&chunk)
	if err != nil {
		return err
	}

	resp := make(amf.Object)
	resp["fmsVer"] = "FMS/3,0,1,123"
	resp["capabilities"] = 31

	event := make(amf.Object)
	event["level"] = "status"
	event["code"] = "NetConnection.Connect.Success"
	event["description"] = "Connection succeeded."
	event["objectEncoding"] = c.ConnInfo.ObjectEncoding
	return c.writeMsg(cur.CSID, cur.StreamID, "_result", c.transactionID, resp, event)
}

func (c *ConnServer) createStream(vs []interface{}) error {
	for _, v := range vs {
		switch m := v.(type) {
		case string:
		case float64:
			c.transactionID = int(m)
		case amf.Object:
		}
	}
	return nil
}

func (c *ConnServer) createStreamResp(cur *ChunkStream) error {
	return c.writeMsg(cur.CSID, cur.StreamID, "_result", c.transactionID, nil, c.streamID)
}

func (c *ConnServer) publishOrPlay(vs []interface{}) error {
	for k, v := range vs {
		switch m := v.(type) {
		case string:
			if k == 2 {
				c.PublishInfo.Name = m
			} else if k == 3 {
				c.PublishInfo.Type = m
			}
		case float64:
			id := int(m)
			c.transactionID = id
		case amf.Object:
		}
	}

	return nil
}

func (c *ConnServer) publishResp(cur *ChunkStream) error {
	event := make(amf.Object)
	event["level"] = "status"
	event["code"] = "NetStream.Publish.Start"
	event["description"] = "Start publising."
	return c.writeMsg(cur.CSID, cur.StreamID, "onStatus", 0, nil, event)
}

func (c *ConnServer) playResp(cur *ChunkStream) error {
	err := c.conn.SetRecorded()
	if err != nil {
		return err
	}
	err = c.conn.SetBegin()
	if err != nil {
		return err
	}

	event := make(amf.Object)
	event["level"] = "status"
	event["code"] = "NetStream.Play.Reset"
	event["description"] = "Playing and resetting stream."
	if err := c.writeMsg(cur.CSID, cur.StreamID, "onStatus", 0, nil, event); err != nil {
		return err
	}

	event["level"] = "status"
	event["code"] = "NetStream.Play.Start"
	event["description"] = "Started playing stream."
	if err := c.writeMsg(cur.CSID, cur.StreamID, "onStatus", 0, nil, event); err != nil {
		return err
	}

	event["level"] = "status"
	event["code"] = "NetStream.Data.Start"
	event["description"] = "Started playing stream."
	if err := c.writeMsg(cur.CSID, cur.StreamID, "onStatus", 0, nil, event); err != nil {
		return err
	}

	event["level"] = "status"
	event["code"] = "NetStream.Play.PublishNotify"
	event["description"] = "Started playing notify."
	if err := c.writeMsg(cur.CSID, cur.StreamID, "onStatus", 0, nil, event); err != nil {
		return err
	}
	return c.conn.Flush()
}

func (c *ConnServer) HandleCmdMsg(chunk *ChunkStream) error {
	amfType := amf.AMF0
	if chunk.TypeID == 17 {
		chunk.Data = chunk.Data[1:]
	}
	r := bytes.NewReader(chunk.Data)
	vs, err := c.decoder.DecodeBatch(r, amf.Version(amfType))
	if err != nil && err != io.EOF {
		return err
	}
	switch cmd := vs[0].(type) {
	case string:
		switch cmd {
		case cmdConnect:
			if err = c.connect(vs[1:]); err != nil {
				return err
			}
			if err = c.connectResp(chunk); err != nil {
				return err
			}
		case cmdCreateStream:
			if err = c.createStream(vs[1:]); err != nil {
				return err
			}
			if err = c.createStreamResp(chunk); err != nil {
				return err
			}
		case cmdPublish:
			c.isPublisher = true
			if err = c.publishOrPlay(vs[1:]); err != nil {
				return err
			}
			if c.cb != nil {
				if err := c.cb(); err != nil {
					return err
				}
			}
			if err = c.publishResp(chunk); err != nil {
				return err
			}
			c.done = true
		case cmdPlay:
			if err = c.publishOrPlay(vs[1:]); err != nil {
				return err
			}
			if c.cb != nil {
				if err := c.cb(); err != nil {
					return err
				}
			}
			if err = c.playResp(chunk); err != nil {
				return err
			}
			c.done = true
			c.isPublisher = false
		case cmdFcpublish:
			return c.fcPublish(vs)
		case cmdReleaseStream:
			return c.releaseStream(vs)
		case cmdFCUnpublish:
		case cmdDeleteStream:
		default:
			logrus.Error("no support command=", vs[0].(string))
		}
	}

	return nil
}

func (c *ConnServer) ReadMsg() error {
	var chunk ChunkStream
	for {
		if err := c.conn.Read(&chunk); err != nil {
			return err
		}
		switch chunk.TypeID {
		case 20, 17:
			if err := c.HandleCmdMsg(&chunk); err != nil {
				return err
			}
		}
		if c.done {
			break
		}
	}
	return nil
}

func (c *ConnServer) IsPublisher() bool {
	return c.isPublisher
}

func (c *ConnServer) Write(chunk ChunkStream) error {
	if chunk.TypeID == av.TAG_SCRIPTDATAAMF0 ||
		chunk.TypeID == av.TAG_SCRIPTDATAAMF3 {
		var err error
		if chunk.Data, err = amf.MetaDataReform(chunk.Data, amf.DEL); err != nil {
			return err
		}
		chunk.Length = uint32(len(chunk.Data))
	}
	return c.conn.Write(&chunk)
}

func (c *ConnServer) Close() error {
	return c.conn.Close()
}

func (c *ConnServer) Flush() error {
	return c.conn.Flush()
}

func (c *ConnServer) Read(chunk *ChunkStream) (err error) {
	return c.conn.Read(chunk)
}

func (c *ConnServer) GetInfo() av.Info {
	return av.Info{
		Name: c.PublishInfo.Name,
		App:  c.ConnInfo.App,
		URL:  c.ConnInfo.TcUrl + "/" + c.PublishInfo.Name,
	}
}
