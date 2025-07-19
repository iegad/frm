package nw

import (
	"time"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
)

// ConnContext 连接上下文
type ConnContext struct {
	c             gnet.Conn // 原始连接
	fd            int       // 文件描述符
	upgraded      bool      // websocket 使用
	lastUpdate    int64     // 最后接收消息时间
	server        IServer   // 所属服务
	remoteAddr    string    // 远端地址
	userData      any       // 用户数据
	xRealIP       string    // ws 中 X-Real-IP
	xForwardedFor string    // ws 中 X-Forwareded-For
}

// Init 初始化
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

// Reset 重置
func (this_ *ConnContext) Reset() {
	this_.c.SetContext(nil)
	this_.fd = 0
}

// Fd 获取 socket 文件描述符
func (this_ *ConnContext) Fd() int {
	if this_.fd == 0 {
		this_.fd = this_.c.Fd()
	}

	return this_.fd
}

// Close 关闭连接
func (this_ *ConnContext) Close() {
	err := this_.c.Close()
	if err != nil {
		log.Error("[%d:%v] Close error: %v", this_.Fd(), this_.remoteAddr, err)
	}
}

// Protocol 客户端的连接协议
func (this_ *ConnContext) Protocol() Protocol {
	return this_.server.Proto()
}

// RemoteAddr 客户端对端地址
func (this_ *ConnContext) RemoteAddr() string {
	if len(this_.xForwardedFor) > 0 {
		return this_.xForwardedFor
	}

	if len(this_.xRealIP) > 0 {
		return this_.xRealIP
	}

	return this_.remoteAddr
}

// XRealIP 只有在websocket协议中有效
func (this_ *ConnContext) XRealIP() string {
	return this_.xRealIP
}

// XForwardedFor 只有在websocket协议中有效
func (this_ *ConnContext) XForwardedFor() string {
	return this_.xForwardedFor
}

// UserData 用户自定义数据
func (this_ *ConnContext) UserData() any {
	return this_.userData
}

func (this_ *ConnContext) SetUserData(ud any) {
	this_.userData = ud
}

// Write 发送数据
func (this_ *ConnContext) Write(data []byte) error {
	return this_.server.Write(this_, data)
}
