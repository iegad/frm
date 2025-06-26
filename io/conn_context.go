package io

import (
	"time"

	"github.com/panjf2000/gnet/v2"
)

type ConnContext struct {
	c gnet.Conn

	upgraded      bool // websocket 使用
	server        IServer
	remoteAddr    string
	xRealIP       string
	xForwardedFor string
	userData      any
	lastUpdate    int64
}

func (this_ *ConnContext) Init(c gnet.Conn, server IServer, xRealIP, xForwardedFor string) {
	this_.c = c
	this_.upgraded = server.Proto() == Protocol_TCP
	this_.server = server
	this_.remoteAddr = c.RemoteAddr().String()
	this_.xRealIP = xRealIP
	this_.xForwardedFor = xForwardedFor
	this_.userData = nil
	this_.lastUpdate = time.Now().Unix()
	c.SetContext(this_)
}

func (this_ *ConnContext) Reset() {
	this_.c.SetContext(nil)
	this_.userData = nil
}

func (this_ *ConnContext) Fd() int {
	return this_.c.Fd()
}

func (this_ *ConnContext) Close() {
	this_.c.Close()
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

func (this_ *ConnContext) asyncWrite(data []byte, callback gnet.AsyncCallback) error {
	return this_.c.AsyncWrite(data, callback)
}
