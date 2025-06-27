//go:build unix

package nw

import (
	"errors"
	"syscall"

	"github.com/panjf2000/gnet/v2"
	"golang.org/x/sys/unix"
)

func IsConnReset(err error) bool {
	return errors.Is(err, syscall.ECONNRESET)
}

func GetSockRecvBuffer(c gnet.Conn) (int, error) {
	fd := c.Fd()
	size, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF)
	if err != nil {
		return -1, err
	}

	return size, nil
}

func GetSockSendBuffer(c gnet.Conn) (int, error) {
	fd := c.Fd()

	size, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF)
	if err != nil {
		return -1, err
	}

	return size, nil
}
