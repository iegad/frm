// !windows

package io

import (
	"syscall"

	"github.com/panjf2000/gnet/v2"
)

func GetSockRecvBuffer(c gnet.Conn) (int, error) {
	fd := syscall.Handle(c.Fd())

	size, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	if err != nil {
		return -1, err
	}

	return size, nil
}

func GetSockSendBuffer(c gnet.Conn) (int, error) {
	fd := syscall.Handle(c.Fd())

	size, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF)
	if err != nil {
		return -1, err
	}

	return size, nil
}
