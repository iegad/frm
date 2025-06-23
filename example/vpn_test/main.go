package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gox/frm/log"
	"github.com/gox/frm/web"
)

type response struct {
	ClientIP      string `json:"client_ip"`
	RemoteIP      string `json:"remote_ip"`
	XForwardForIP string `json:"X-Forwarded-For"`
	XRealIP       string `json:"X-Real-IP"`
}

func testVpn(c *gin.Context) {
	log.Debug("client ip: %v", c.ClientIP())
	log.Debug("remote ip: %v", c.RemoteIP())

	xForwardedFor := c.Request.Header.Get("X-Forwarded-For")
	log.Debug("X-Forwarded-For: %v", xForwardedFor)

	xRealIp := c.Request.Header.Get("X-Real-IP")
	log.Debug("xRealIp: %v", xRealIp)

	web.Response(c, 0, "", &response{
		ClientIP:      c.ClientIP(),
		RemoteIP:      c.RemoteIP(),
		XForwardForIP: xForwardedFor,
		XRealIP:       xRealIp,
	})
}

func main() {
	s, err := web.NewServer(":5090", true, true)
	if err != nil {
		log.Error("server: %v", err)
		return
	}

	s.Router().GET("/test_vpn", testVpn)
	s.Run()
}
