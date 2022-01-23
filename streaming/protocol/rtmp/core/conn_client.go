package core

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	neturl "net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/streaming/av"
	"github.com/viderstv/common/streaming/protocol/amf"
)

var (
	respResult     = "_result"
	onStatus       = "onStatus"
	publishStart   = "NetStream.Publish.Start"
	connectSuccess = "NetConnection.Connect.Success"
	publishLive    = "live"
)

var (
	ErrFail = fmt.Errorf("respone err")
)

type ConnClient struct {
	transID int
	tcurl   string

	app  string
	name string
	url  string

	query      string
	curcmdName string
	streamid   uint32
	conn       *Conn
	encoder    *amf.Encoder
	decoder    *amf.Decoder
	bytesw     *bytes.Buffer

	tls *tls.Config
}

func NewConnClient() *ConnClient {
	return &ConnClient{
		transID: 1,
		bytesw:  bytes.NewBuffer(nil),
		encoder: &amf.Encoder{},
		decoder: &amf.Decoder{},
		tls:     &tls.Config{},
	}
}

func NewConnClientWithTls(tls *tls.Config) *ConnClient {
	return &ConnClient{
		transID: 1,
		bytesw:  bytes.NewBuffer(nil),
		encoder: &amf.Encoder{},
		decoder: &amf.Decoder{},
		tls:     tls.Clone(),
	}
}

func (c *ConnClient) DecodeBatch(r io.Reader, ver amf.Version) (ret []interface{}, err error) {
	return c.decoder.DecodeBatch(r, ver)
}

func (c *ConnClient) readRespMsg() error {
	var err error
	var rc ChunkStream
	for {
		if err = c.conn.Read(&rc); err != nil {
			return err
		}
		if err != nil && err != io.EOF {
			return err
		}
		switch rc.TypeID {
		case 20, 17:
			r := bytes.NewReader(rc.Data)
			vs, _ := c.decoder.DecodeBatch(r, amf.AMF0)

			logrus.Debugf("readRespMsg: vs=%v", vs)
			for k, v := range vs {
				switch t := v.(type) {
				case string:
					switch c.curcmdName {
					case cmdConnect, cmdCreateStream:
						if t != respResult {
							return fmt.Errorf(t)
						}

					case cmdPublish:
						if t != onStatus {
							return ErrFail
						}
					}
				case float64:
					switch c.curcmdName {
					case cmdConnect, cmdCreateStream:
						id := int(v.(float64))

						if k == 1 {
							if id != c.transID {
								return ErrFail
							}
						} else if k == 3 {
							c.streamid = uint32(id)
						}
					case cmdPublish:
						if int(v.(float64)) != 0 {
							return ErrFail
						}
					}
				case amf.Object:
					objmap := v.(amf.Object)
					switch c.curcmdName {
					case cmdConnect:
						code, ok := objmap["code"]
						if ok && code.(string) != connectSuccess {
							return ErrFail
						}
					case cmdPublish:
						code, ok := objmap["code"]
						if ok && code.(string) != publishStart {
							return ErrFail
						}
					}
				}
			}

			return nil
		}
	}
}

func (c *ConnClient) writeMsg(args ...interface{}) error {
	c.bytesw.Reset()
	for _, v := range args {
		if _, err := c.encoder.Encode(c.bytesw, v, amf.AMF0); err != nil {
			return err
		}
	}

	msg := c.bytesw.Bytes()
	chunk := ChunkStream{
		Format:    0,
		CSID:      3,
		Timestamp: 0,
		TypeID:    20,
		StreamID:  c.streamid,
		Length:    uint32(len(msg)),
		Data:      msg,
	}

	if err := c.conn.Write(&chunk); err != nil {
		return err
	}

	return c.conn.Flush()
}

func (c *ConnClient) writeConnectMsg() error {
	event := make(amf.Object)
	event["app"] = c.app
	event["type"] = "nonprivate"
	event["flashVer"] = "FMS.3.1"
	event["tcUrl"] = c.tcurl
	c.curcmdName = cmdConnect

	logrus.Debugf("writeConnectMsg: c.transID=%d, event=%v", c.transID, event)
	if err := c.writeMsg(cmdConnect, c.transID, event); err != nil {
		return err
	}
	return c.readRespMsg()
}

func (c *ConnClient) writeCreateStreamMsg() error {
	c.transID++
	c.curcmdName = cmdCreateStream

	logrus.Debugf("writeCreateStreamMsg: c.transID=%d", c.transID)
	if err := c.writeMsg(cmdCreateStream, c.transID, nil); err != nil {
		return err
	}

	for {
		err := c.readRespMsg()
		if err == nil {
			return err
		}

		if err == ErrFail {
			logrus.Debugf("writeCreateStreamMsg readRespMsg err=%v", err)
			return err
		}
	}

}

func (c *ConnClient) writePublishMsg() error {
	c.transID++
	c.curcmdName = cmdPublish
	if err := c.writeMsg(cmdPublish, c.transID, nil, c.name, publishLive); err != nil {
		return err
	}
	return c.readRespMsg()
}

func (c *ConnClient) writePlayMsg() error {
	c.transID++
	c.curcmdName = cmdPlay

	if err := c.writeMsg(cmdPlay, 0, nil, c.name); err != nil {
		return err
	}
	return c.readRespMsg()
}

func (c *ConnClient) Start(url string, method string) error {
	u, err := neturl.Parse(url)
	if err != nil {
		return err
	}

	c.url = url

	path := strings.TrimLeft(u.Path, "/")
	ps := strings.SplitN(path, "/", 2)

	if len(ps) != 2 {
		return fmt.Errorf("u path err: %s", path)
	}

	c.app = ps[0]
	c.name = ps[1]

	c.query = u.RawQuery
	c.tcurl = "rtmp://" + u.Host + "/" + c.app

	var conn net.Conn
	if u.Scheme == "rtmp" {
		conn, err = net.Dial("tcp", u.Host)
		if err != nil {
			return err
		}
	} else if u.Scheme == "rtmps" {
		conn, err = tls.Dial("tcp", u.Host, c.tls)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	c.conn = NewConn(conn, 4*1024)

	if err := c.conn.HandshakeClient(); err != nil {
		return err
	}

	if err := c.writeConnectMsg(); err != nil {
		return err
	}

	if err := c.writeCreateStreamMsg(); err != nil {
		return err
	}

	if method == av.PUBLISH {
		if err := c.writePublishMsg(); err != nil {
			return err
		}
	} else if method == av.PLAY {
		if err := c.writePlayMsg(); err != nil {
			return err
		}
	}

	return nil
}

func (c *ConnClient) Write(chunk ChunkStream) error {
	if chunk.TypeID == av.TAG_SCRIPTDATAAMF0 ||
		chunk.TypeID == av.TAG_SCRIPTDATAAMF3 {
		var err error
		if chunk.Data, err = amf.MetaDataReform(chunk.Data, amf.ADD); err != nil {
			return err
		}
		chunk.Length = uint32(len(chunk.Data))
	}
	return c.conn.Write(&chunk)
}

func (c *ConnClient) Flush() error {
	return c.conn.Flush()
}

func (c *ConnClient) Read(chunk *ChunkStream) (err error) {
	return c.conn.Read(chunk)
}

func (c *ConnClient) GetInfo() av.Info {
	return av.Info{
		Name: c.name,
		App:  c.app,
		URL:  c.url,
	}
}

func (c *ConnClient) GetStreamId() uint32 {
	return c.streamid
}

func (c *ConnClient) Close() error {
	return c.conn.Close()
}
