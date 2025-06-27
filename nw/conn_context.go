package nw

import (
	"time"

	"github.com/panjf2000/gnet/v2"
)

type ConnContext struct {
	c             gnet.Conn // 原始连接
	upgraded      bool      // websocket 使用
	lastUpdate    int64     // 最后接收消息时间
	server        IServer   // 所属服务
	remoteAddr    string    // 远端地址
	userData      any       // 用户数据
	xRealIP       string
	xForwardedFor string
}

func (this_ *ConnContext) Init(c gnet.Conn, server IServer) {
	this_.c = c
	this_.upgraded = server.Proto() == Protocol_TCP
	this_.server = server
	this_.remoteAddr = c.RemoteAddr().String()
	this_.xRealIP = ""
	this_.xForwardedFor = ""
	this_.userData = nil
	this_.lastUpdate = time.Now().Unix()
	c.SetContext(this_)
}

func (this_ *ConnContext) Reset() {
	this_.c.SetContext(nil)
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
	if len(this_.xForwardedFor) > 0 {
		return this_.xForwardedFor
	}

	if len(this_.xRealIP) > 0 {
		return this_.xRealIP
	}

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
