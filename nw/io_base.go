package nw

import (
	"net"
	"net/http"
	"strings"
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
