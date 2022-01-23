package errors

import "fmt"

var (
	ErrNoSupportVideoCodec = fmt.Errorf("unsupported video codec")
	ErrNoSupportAudioCodec = fmt.Errorf("unsupported audio codec")
	ErrSourceClosed        = fmt.Errorf("the source is closed")
)
