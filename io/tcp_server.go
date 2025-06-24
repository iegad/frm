package io

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type ServerState int32

type tcpServer struct {
	gnet.BuiltinEventEngine
	eng gnet.Engine

	headBlend uint32
	owner     *Server
	host      string
}

func newTcpServer(owner *Server, c *Config) *tcpServer {
	return &tcpServer{
		owner:     owner,
		headBlend: c.HeadBlend,
		host:      fmt.Sprintf("tcp://%v", c.TcpHost),
	}
}

func (this_ *tcpServer) OnBoot(eng gnet.Engine) gnet.Action {
	this_.eng = eng
	return gnet.None
}

func (this_ *tcpServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	if atomic.LoadInt32(&this_.owner.currConn) >= this_.owner.maxConn {
		return nil, gnet.Close
	}

	atomic.AddInt32(&this_.owner.currConn, 1)

	if err := this_.owner.event.OnConnected(c); err != nil {
		log.Error("[%d:%v] connected failed: %v", err)
		return nil, gnet.Close
	}

	return nil, gnet.None
}

func (this_ *tcpServer) OnClose(c gnet.Conn, err error) gnet.Action {

	atomic.AddInt32(&this_.owner.currConn, -1)
	this_.owner.event.OnDisconnected(c)
	return gnet.None
}

func (this_ *tcpServer) OnTraffic(c gnet.Conn) gnet.Action {
	data, err := c.Next(-1)
	if err != nil {
		logging.Errorf("[%v] read failed: %v", c.RemoteAddr(), err)
		return gnet.Close
	}

	msg := messagePool.Get()
	msg.Conn = c
	msg.Data = data
	this_.owner.messageCh <- msg

	return gnet.None
}

func (this_ *tcpServer) Run() error {
	err := gnet.Run(this_, this_.host,
		gnet.WithMulticore(true),
		gnet.WithReuseAddr(true),
		gnet.WithReusePort(true),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
		gnet.WithSocketSendBuffer(RECV_BUF_SIZE),
		gnet.WithSocketRecvBuffer(SEND_BUF_SIZE),
		gnet.WithLogLevel(logging.DebugLevel),
	)

	if err != nil {
		logging.Errorf("run failed: %v", err)
		return err
	}

	return nil
}

func (this_ *tcpServer) Stop() {
	this_.eng.Stop(context.TODO())
}

func (this_ *tcpServer) Write(c gnet.Conn, data []byte) error {
	return c.AsyncWrite(data, nil)
}
