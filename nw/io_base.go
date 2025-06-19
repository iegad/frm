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
	TCP_HEADER_SIZE = int(unsafe.Sizeof(uint32(0)))
	TCP_MAX_SIZE    = 1024 * 1024 * 2
	MAX_CHAN_SIZE   = 100000
)

var errDataTooLong = errors.New("data too long")

// 获取 Http/Websocket 客户端的真实IP
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
	if dlen > TCP_MAX_SIZE {
		return -1, errDataTooLong
	}

	wbuf := make([]byte, dlen+TCP_HEADER_SIZE)
	binary.BigEndian.PutUint32(wbuf[:TCP_HEADER_SIZE], uint32(dlen)^blend)
	copy(wbuf[TCP_HEADER_SIZE:], data)
	return conn.Write(wbuf)
}

func IsClosedErr(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.EPIPE)
}
