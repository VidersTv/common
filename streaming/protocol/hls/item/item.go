package item

import (
	"fmt"
	"io"
	"time"

	"github.com/viderstv/common/streaming/protocol/hls/buffer"
)

type Item struct {
	name     string
	seqNum   int
	duration time.Duration
	start    time.Time

	data *buffer.Buffer

	writer io.WriteCloser
}

func New(name string, seqNum int) *Item {
	reader, writer := io.Pipe()

	return &Item{
		name:   name,
		seqNum: seqNum,
		data:   buffer.New(reader),
		writer: writer,
	}
}

func (i *Item) String() string {
	return fmt.Sprintf("<id: %d, name: %s>", i.seqNum, i.name)
}

func (i *Item) AddWriter(writer io.WriteCloser) (string, error) {
	return i.data.AddWriter(writer)
}

func (i *Item) Write(data []byte) (int, error) {
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

func (i *Item) Duration() time.Duration {
	return i.duration
}

func (i *Item) Close() error {
	return i.writer.Close()
}
