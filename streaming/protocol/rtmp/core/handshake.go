package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"time"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/utils/pio"
)

var (
	timeout = 3 * time.Second
)

var (
	hsClientFullKey = []byte{
		'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
		'F', 'l', 'a', 's', 'h', ' ', 'P', 'l', 'a', 'y', 'e', 'r', ' ',
		'0', '0', '1',
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
		0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
	hsServerFullKey = []byte{
		'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
		'F', 'l', 'a', 's', 'h', ' ', 'M', 'e', 'd', 'i', 'a', ' ',
		'S', 'e', 'r', 'v', 'e', 'r', ' ',
		'0', '0', '1',
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
		0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
	hsClientPartialKey = hsClientFullKey[:30]
	hsServerPartialKey = hsServerFullKey[:36]
)

func hsParse1(p []byte, peerkey []byte, key []byte) (ok bool, digest []byte) {
	var pos int
	if pos = hsFindDigest(p, peerkey, 772); pos == -1 {
		if pos = hsFindDigest(p, peerkey, 8); pos == -1 {
			return
		}
	}
	ok = true
	digest = hsMakeDigest(key, p[pos:pos+32], -1)
	return
}

func hsMakeDigest(key []byte, src []byte, gap int) (dst []byte) {
	h := hmac.New(sha256.New, key)
	if gap <= 0 {
		h.Write(src)
	} else {
		h.Write(src[:gap])
		h.Write(src[gap+32:])
	}
	return h.Sum(nil)
}

func hsCalcDigestPos(p []byte, base int) (pos int) {
	for i := 0; i < 4; i++ {
		pos += int(p[base+i])
	}
	pos = (pos % 728) + base + 4
	return
}

func hsFindDigest(p []byte, key []byte, base int) int {
	gap := hsCalcDigestPos(p, base)
	digest := hsMakeDigest(key, p, gap)
	if !bytes.Equal(p[gap:gap+32], digest) {
		return -1
	}
	return gap
}

func hsCreate01(p []byte, time uint32, ver uint32, key []byte) error {
	p[0] = 3
	p1 := p[1:]
	_, err := rand.Read(p1[8:])
	if err != nil {
		return err
	}
	pio.PutU32BE(p1[0:4], time)
	pio.PutU32BE(p1[4:8], ver)
	gap := hsCalcDigestPos(p1, 8)
	digest := hsMakeDigest(key, p1, gap)
	copy(p1[gap:], digest)
	return nil
}

func hsCreate2(p []byte, key []byte) error {
	_, err := rand.Read(p)
	if err != nil {
		return err
	}
	gap := len(p) - 32
	digest := hsMakeDigest(key, p, gap)
	copy(p[gap:], digest)
	return nil
}

const (
	C0S0Size = 1    // 5.2.2
	C1S1Size = 1536 // 5.2.3
	C2S2Size = 1536 // 5.2.4

	RTMPVersion = 3
)

type C0S0 []byte

func (b C0S0) Version() byte {
	return b[0]
}

func (b C0S0) SetVersion(version byte) {
	b[0] = version
}

type C1S1 []byte

// According to the spec the first 4 bytes denote time
func (b C1S1) Time() uint32 {
	return pio.U32BE(b[:4])
}

func (b C1S1) SetTime(time uint32) {
	pio.PutU32BE(b[:4], time)
}

// According to the spec the 4-8 bytes are zeros
func (b C1S1) Zero() uint32 {
	return pio.U32BE(b[4:8])
}

func (b C1S1) SetZero(zero uint32) {
	pio.PutU32BE(b[4:8], zero)
}

// According to the spec the 8-1536 bytes are random
func (b C1S1) Random() []byte {
	return b[8 : 1528+8]
}

func (b C1S1) SetRandom(random []byte) {
	copy(b[8:1528+8], random)
}

type C2S2 []byte

// According to the spec the 4-8 bytes are time from C1S1
func (b C2S2) Time() uint32 {
	return pio.U32BE(b[:4])
}

func (b C2S2) SetTime(time uint32) {
	pio.PutU32BE(b[:4], time)
}

// According to the spec the 4-8 bytes are time read by peer
func (b C2S2) Time2() uint32 {
	return pio.U32BE(b[4:8])
}

func (b C2S2) SetTime2(time uint32) {
	pio.PutU32BE(b[4:8], time)
}

// According to the spec the 8-1536 bytes are the random bytes from C1S1
func (b C2S2) Random() []byte {
	return b[8 : 1528+8]
}

func (b C2S2) SetRandom(random []byte) {
	copy(b[8:1528+8], random)
}

// https://www.adobe.com/content/dam/acom/en/devnet/rtmp/pdf/rtmp_specification_1.0.pdf
func (c *Conn) HandshakeServer() error {
	c012 := make([]byte, C0S0Size+C1S1Size+C2S2Size)
	c0 := C0S0(c012[:C0S0Size])
	c1 := C1S1(c012[C0S0Size : C0S0Size+C1S1Size])
	c2 := C2S2(c012[C0S0Size+C1S1Size : C0S0Size+C1S1Size+C2S2Size])
	s012 := make([]byte, C0S0Size+C1S1Size+C2S2Size)
	s0 := C0S0(s012[:C0S0Size])
	s1 := C1S1(s012[C0S0Size : C0S0Size+C1S1Size])
	s2 := C2S2(s012[C0S0Size+C1S1Size : C0S0Size+C1S1Size+C2S2Size])

	if err := c.Conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}

	// read c0
	if _, err := io.ReadFull(c.rw, c0); err != nil {
		return err
	}

	// read c1
	if _, err := io.ReadFull(c.rw, c1); err != nil {
		return err
	}

	if c0.Version() != RTMPVersion {
		logrus.Debug("bad version from client: ", c0.Version())
	}

	// This seems to be undocumented as to why we do this digest but it seems that most implementations do this. Unsure.
	if c1.Zero() != 0 {
		if ok, digest := hsParse1(c1, hsClientPartialKey, hsServerFullKey); !ok {
			return fmt.Errorf("rtmp: handshake server: C1 invalid")
		} else {
			if err := hsCreate01(c012[:C0S0Size+C1S1Size], pio.U32BE(c1[:4]), 0, hsServerPartialKey); err != nil {
				return err
			}
			if err := hsCreate2(s2, digest); err != nil {
				return err
			}
		}
	} else {
		// Set the version to 3
		s0.SetVersion(RTMPVersion)
		s1.SetTime(c1.Time())
		s1.SetZero(0)

		// Set S2 values, I am not sure why time2 exists or time for that matter.
		s2.SetTime(c1.Time())
		s2.SetTime2(0)
		s2.SetRandom(c1.Random())
	}

	if _, err := rand.Read(s1.Random()); err != nil {
		return err
	}

	if err := c.Conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}

	sentAt := time.Now()
	// write s0
	if _, err := c.rw.Write(s0); err != nil {
		return err
	}

	// write s1
	if _, err := c.rw.Write(s1); err != nil {
		return err
	}

	// write s2
	if _, err := c.rw.Write(s2); err != nil {
		return err
	}

	if err := c.rw.Flush(); err != nil {
		return err
	}

	if err := c.Conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}

	// read c2
	if _, err := c.rw.Read(c2); err != nil {
		return err
	}

	if s1.Time() != c2.Time() {
		return fmt.Errorf("invalid c2 bad time")
	}

	if !bytes.Equal(s1.Random(), c2.Random()) {
		return fmt.Errorf("invalid c2 bad random")
	}

	delay := time.Since(sentAt)
	logrus.Debug("delay is: ", delay)

	return c.Conn.SetDeadline(time.Time{})
}

func (c *Conn) HandshakeClient() error {
	c012 := make([]byte, C0S0Size+C1S1Size+C2S2Size)
	c0 := C0S0(c012[:C0S0Size])
	c1 := C1S1(c012[C0S0Size : C0S0Size+C1S1Size])
	c2 := C2S2(c012[C0S0Size+C1S1Size : C0S0Size+C1S1Size+C2S2Size])
	s012 := make([]byte, C0S0Size+C1S1Size+C2S2Size)
	s0 := C0S0(s012[:C0S0Size])
	s1 := C1S1(s012[C0S0Size : C0S0Size+C1S1Size])
	s2 := C2S2(s012[C0S0Size+C1S1Size : C0S0Size+C1S1Size+C2S2Size])

	c0.SetVersion(RTMPVersion)

	if _, err := rand.Read(c1.Random()); err != nil {
		return err
	}

	if err := c.Conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}

	sentAt := time.Now()
	// write c0
	if _, err := c.rw.Write(c0); err != nil {
		return err
	}
	// write c1
	if _, err := c.rw.Write(c1); err != nil {
		return err
	}
	if err := c.rw.Flush(); err != nil {
		return err
	}

	if err := c.Conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	// read s0
	if _, err := c.rw.Read(s0); err != nil {
		return err
	}
	// read s1
	if _, err := c.rw.Read(s1); err != nil {
		return err
	}
	// read s2
	if _, err := c.rw.Read(s2); err != nil {
		return err
	}

	if s2.Time() != c1.Time() {
		return fmt.Errorf("invalid s2 bad time")
	}

	if !bytes.Equal(c1.Random(), s2.Random()) {
		return fmt.Errorf("invalid s2 bad random")
	}

	delay := time.Since(sentAt)
	logrus.Debug("delay is: ", delay)

	c2.SetTime(s1.Time())
	c2.SetRandom(s1.Random())

	// write c2
	if _, err := c.rw.Write(c2); err != nil {
		return err
	}
	if err := c.rw.Flush(); err != nil {
		return err
	}

	return c.Conn.SetDeadline(time.Time{})
}
