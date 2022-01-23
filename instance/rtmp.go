package instance

import "net"

type RtmpServer interface {
	Serve(listener net.Listener) error
	Shutdown() error
}
