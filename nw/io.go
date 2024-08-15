package nw

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"syscall"
	"time"

	"github.com/gox/frm/log"
)

func write(conn net.Conn, data []byte, timeout time.Duration, blend uint32) (int, error) {
	if conn == nil {
		log.Fatal("conn is nil")
	}

	if timeout > 0 {
		err := conn.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			return -1, err
		}
	}

	dlen := len(data)
	wbuf := make([]byte, dlen+UINT32_SIZE)
	binary.BigEndian.PutUint32(wbuf[:UINT32_SIZE], uint32(dlen)^blend)
	copy(wbuf[UINT32_SIZE:], data)
	return conn.Write(wbuf)
}

func IsClosedErr(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.EPIPE)
}
