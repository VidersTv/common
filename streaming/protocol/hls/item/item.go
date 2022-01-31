package item

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/viderstv/common/streaming/protocol/hls/buffer"
	"github.com/viderstv/common/utils"
)

type Item struct {
	name     string
	seqNum   int
	duration time.Duration
	start    time.Time

	size *int32

	data *buffer.Buffer

	writer io.WriteCloser
}

func New(name string, seqNum int) *Item {
	reader, writer := io.Pipe()

	return &Item{
		name:   name,
		seqNum: seqNum,
		size:   utils.Int32Pointer(0),
		data:   buffer.New(reader, name),
		writer: writer,
	}
}

func (i *Item) Size() int {
	return i.data.Size()
}

func (i *Item) String() string {
	return fmt.Sprintf("<id: %d, name: %s, duration: %s, size: %d, written: %d>", i.seqNum, i.name, i.duration, i.Size(), atomic.LoadInt32(i.size))
}

func (i *Item) AddWriter(writer io.WriteCloser) (string, error) {
	return i.data.AddWriter(writer)
}

func (i *Item) RemoveWriter(name string) {
	i.data.RemoveWriter(name)
}

func (i *Item) Write(data []byte) (int, error) {
	atomic.AddInt32(i.size, int32(len(data)))
	if i.start.IsZero() {
		i.start = time.Now()
	}

	return i.writer.Write(data)
}

func (i *Item) SetDuration(dur time.Duration) {
	i.duration = dur
}

func (i *Item) Name() string {
	return i.name
}

func (i *Item) SeqNum() int {
	return i.seqNum
}

func (i *Item) Start() time.Time {
	return i.start
}

func (i *Item) Duration() time.Duration {
	return i.duration
}

func (i *Item) Close() error {
	logrus.Info(i)
	return i.writer.Close()
}
