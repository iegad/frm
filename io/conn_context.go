package io

import (
	"time"

	"github.com/gox/frm/utils"
	"github.com/panjf2000/gnet/v2"
)

var connContextPool = utils.NewPool[ConnContext]()

func getConnContext() *ConnContext {
	return connContextPool.Get()
}

func putConnContext(ctx *ConnContext) {
	if ctx != nil {
		if ctx.Conn != nil {
			ctx.Conn = nil
		}
		connContextPool.Put(ctx)
	}
}

type ConnContext struct {
	gnet.Conn

	upgraded      bool
	server        iServer
	remoteAddr    string
	xRealIP       string
	xForwardedFor string
	userData      any
	lastUpdate    int64
}

func (this_ *ConnContext) Init(c gnet.Conn, server iServer, xRealIP, xForwardedFor string) {
	this_.Conn = c
	this_.server = server
	this_.remoteAddr = c.RemoteAddr().String()
	this_.xRealIP = xRealIP
	this_.xForwardedFor = xForwardedFor
	this_.lastUpdate = time.Now().Unix()
}

func (this_ *ConnContext) Protocol() Protocol {
	return this_.server.Proto()
}

func (this_ *ConnContext) RemoteAddr() string {
	return this_.remoteAddr
}

func (this_ *ConnContext) XRealIP() string {
	return this_.xRealIP
}

func (this_ *ConnContext) XForwardedFor() string {
	return this_.xForwardedFor
}

func (this_ *ConnContext) UserData() any {
	return this_.userData
}

func (this_ *ConnContext) SetUserData(ud any) {
	this_.userData = ud
}

func (this_ *ConnContext) Write(data []byte) error {
	return this_.server.Write(this_, data)
}
