package buffer

import (
	"bytes"
	"io"
	"sync"

	"github.com/viderstv/common/utils/uid"
)

const (
	DefaultBufferSize = 4096
)

type Buffer struct {
	reader     io.Reader
	writersMtx sync.Mutex
	writers    map[string]io.WriteCloser
	data       *bytes.Buffer
	buf        []byte
	closed     chan struct{}
}

func New(reader io.Reader) *Buffer {
	return NewWithSize(reader, DefaultBufferSize)
}

func NewWithSize(reader io.Reader, size int) *Buffer {
	b := &Buffer{
		reader:  reader,
		buf:     make([]byte, size),
		writers: map[string]io.WriteCloser{},
		data:    &bytes.Buffer{},
		closed:  make(chan struct{}),
	}

	go b.copy()

	return b
}

func (b *Buffer) copy() {
	for {
		n, err := b.reader.Read(b.buf)
		arr := b.buf[:n]
		if len(arr) != 0 {
			_, _ = b.data.Write(arr)
			b.writersMtx.Lock()
			for k, v := range b.writers {
				if _, err2 := v.Write(arr); err2 != nil || err != nil {
					delete(b.writers, k)
					_ = v.Close()
				}
			}
			b.writersMtx.Unlock()
		}
		if err != nil {
			close(b.closed)
			return
		}
	}
}

func (b *Buffer) AddWriter(writer io.WriteCloser) (string, error) {
	key := uid.NewId()

	b.writersMtx.Lock()
	defer b.writersMtx.Unlock()

	_, err := writer.Write(b.data.Bytes())
	if err != nil {
		return "", err
	}

	select {
	case <-b.closed:
		_ = writer.Close()
		return "", nil
	default:
	}

	b.writers[key] = writer

	return key, nil
}

func (b *Buffer) RemoveWriter(key string) {
	b.writersMtx.Lock()
	defer b.writersMtx.Unlock()

	if v, ok := b.writers[key]; ok {
		_ = v.Close()
		delete(b.writers, key)
	}
}
