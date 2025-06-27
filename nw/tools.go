package nw

import (
	"errors"
	"io"
	"net"
	"syscall"
)

func IsClosedErr(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.EPIPE)
}
