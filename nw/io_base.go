package nw

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	DEFAULT_TIMEOUT int32 = 300
	MAX_TIMEOUT     int32 = 600
	UINT32_SIZE           = int(unsafe.Sizeof(uint32(0)))
	MAX_BUF_SIZE          = 1024 * 1024 * 2
)

func GetHttpRequestRealIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")

	if len(ip) > 0 {
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	ip = r.Header.Get("X-Real-IP")
	if len(ip) > 0 {
		return ip
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "unknown"
	}

	return ip
}

func write(conn net.Conn, data []byte, timeout time.Duration, blend uint32) (int, error) {
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
