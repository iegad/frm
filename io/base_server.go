package io

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

// IServer 服务接口
type IServer interface {
	gnet.EventHandler

	Proto() Protocol                  // 协议
	Host() string                     // 监听地址
	Write(*ConnContext, []byte) error // 写数据
}

// baseServer 基类
type baseServer struct {
	gnet.BuiltinEventEngine
	eng gnet.Engine

	owner    *Service    // 所属服务
	wbufPool *BufferPool // 写对象池
	server   IServer     // 实际的服务
	host     string      // 监听地址
	cctxPool sync.Pool   // ConnContext 对象池
}

func newBaseServer(owner *Service, server IServer, host string) *baseServer {
	return &baseServer{
		owner:    owner,
		wbufPool: NewBufferPool(),
		server:   server,
		host:     fmt.Sprintf("tcp://%v", host),
		cctxPool: sync.Pool{
			New: func() any {
				return &ConnContext{}
			},
		},
	}
}

func (this_ *baseServer) Proto() Protocol {
	return Protocol_None
}

func (this_ *baseServer) Host() string {
	return this_.host
}

func (this_ *baseServer) OnBoot(eng gnet.Engine) gnet.Action {
	this_.eng = eng
	return gnet.None
}

func (this_ *baseServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	if atomic.LoadInt32(&this_.owner.currConn) >= this_.owner.info.MaxConn {
		return nil, gnet.Close
	}

	cctx := this_.getConnContext()
	cctx.Init(c, this_)

	if err := this_.owner.event.OnConnected(cctx); err != nil {
		log.Error("[%d:%v] connected failed: %v", err)
		this_.putConnContext(cctx)
		return nil, gnet.Close
	}

	atomic.AddInt32(&this_.owner.currConn, 1)
	this_.owner.conns.Set(cctx.Fd(), cctx)
	return nil, gnet.None
}

func (this_ *baseServer) OnClose(c gnet.Conn, err error) gnet.Action {
	atomic.AddInt32(&this_.owner.currConn, -1)
	cctx := c.Context().(*ConnContext)

	this_.owner.conns.Remove(cctx.Fd())
	this_.owner.event.OnDisconnected(cctx)
	this_.putConnContext(cctx)
	return gnet.None
}

func (this_ *baseServer) OnTick() (time.Duration, gnet.Action) {
	tnow := time.Now().Unix()
	timeout := this_.owner.info.Timeout

	this_.owner.conns.Range(func(fd int, cctx *ConnContext) bool {
		if tnow-cctx.lastUpdate > timeout {
			cctx.Close()
		}
		return true
	})

	return time.Second * 15, gnet.None
}

func (this_ *baseServer) Stop() {
	this_.eng.Stop(context.TODO())
}

func (this_ *baseServer) Write(c *ConnContext, data []byte) error {
	return this_.server.Write(c, data)
}

func (this_ *baseServer) getConnContext() *ConnContext {
	return this_.cctxPool.Get().(*ConnContext)
}

func (this_ *baseServer) putConnContext(cctx *ConnContext) {
	cctx.Reset()
	this_.cctxPool.Put(cctx)
}

func Run(server IServer) error {
	return gnet.Run(server, server.Host(),
		gnet.WithMulticore(true),
		gnet.WithReuseAddr(true),
		gnet.WithReusePort(true),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
		gnet.WithSocketSendBuffer(SEND_BUF_SIZE),
		gnet.WithSocketRecvBuffer(RECV_BUF_SIZE),
		gnet.WithLogLevel(logging.DebugLevel),
		gnet.WithTicker(true),
		gnet.WithTCPKeepAlive(time.Second*30),
		gnet.WithTCPKeepCount(2),
		gnet.WithTCPKeepInterval(time.Second*10),
		gnet.WithReadBufferCap(4096),
		gnet.WithLockOSThread(false),
	)
}
