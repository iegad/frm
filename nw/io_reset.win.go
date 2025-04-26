//go:build windows
// +build windows

package nw

import (
	"errors"
	"syscall"
)

func IsConnReset(err error) bool {
	return errors.Is(err, syscall.WSAECONNRESET) || errors.Is(err, syscall.WSAECONNABORTED)
}
