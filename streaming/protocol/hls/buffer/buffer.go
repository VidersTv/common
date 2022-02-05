package buffer

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"time"

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
	name       string
}

func New(reader io.Reader, name string) *Buffer {
	return NewWithSize(reader, DefaultBufferSize, name)
}

func NewWithSize(reader io.Reader, size int, name string) *Buffer {
	b := &Buffer{
		reader:  bufio.NewReaderSize(reader, 8192),
		buf:     make([]byte, size),
		writers: map[string]io.WriteCloser{},
		data:    &bytes.Buffer{},
		closed:  make(chan struct{}),
		name:    name,
	}

	go b.copy()

	return b
}

func (b *Buffer) copy() {
	for {
		n, err := b.reader.Read(b.buf)
		arr := b.buf[:n]
		if len(arr) != 0 {
			_, err := b.data.Write(arr)
			if err != nil {
				panic(err)
			}
		}
		b.writersMtx.Lock()
		for k, v := range b.writers {
			var err2 error
			if len(arr) != 0 {
				_, err2 = v.Write(arr)
			}

			if err2 != nil || err != nil {
				delete(b.writers, k)
				_ = v.Close()
			}
		}
		b.writersMtx.Unlock()
		if err != nil {
			close(b.closed)
			return
		}
	}
}

func (b *Buffer) AddWriter(writer io.WriteCloser) (string, error) {
	key := uid.NewId()

	go func() {
		<-time.After(time.Millisecond * 10)

		b.writersMtx.Lock()
		defer b.writersMtx.Unlock()

		_, err := writer.Write(b.data.Bytes())
		if err != nil {
			return
		}

		select {
		case <-b.closed:
			_ = writer.Close()
		default:
		}
	}()

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

func (b *Buffer) Size() int {
	return b.data.Len()
}
