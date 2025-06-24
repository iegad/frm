package io

import (
	"github.com/gox/frm/utils"
	"github.com/panjf2000/gnet/v2"
)

var connPool = utils.NewPool[Conn]()

func getConn() *Conn {
	return connPool.Get()
}

func putConn(conn *Conn) {
	if conn != nil {
		if conn.Conn != nil {
			conn.Conn = nil
		}
		connPool.Put(conn)
	}
}

type Conn struct {
	gnet.Conn

	server        iServer
	remoteAddr    string
	xRealIP       string
	xForwardedFor string
	userData      any
}

func (this_ *Conn) Init(c gnet.Conn, server iServer, xRealIP, xForwardedFor string) {
	this_.Conn = c
	this_.server = server
	this_.remoteAddr = c.RemoteAddr().String()
	this_.xRealIP = xRealIP
	this_.xForwardedFor = xForwardedFor
}

func (this_ *Conn) Protocol() Protocol {
	return this_.server.Proto()
}

func (this_ *Conn) RemoteAddr() string {
	return this_.remoteAddr
}

func (this_ *Conn) XRealIP() string {
	return this_.xRealIP
}

func (this_ *Conn) XForwardedFor() string {
	return this_.xForwardedFor
}

func (this_ *Conn) UserData() any {
	return this_.userData
}

func (this_ *Conn) SetUserData(ud any) {
	this_.userData = ud
}

func (this_ *Conn) Write(data []byte) error {
	return this_.server.Write(this_, data)
}
