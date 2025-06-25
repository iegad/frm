package io

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/gox/frm/log"
	"github.com/gox/frm/utils"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type wsServer struct {
	gnet.BuiltinEventEngine
	eng gnet.Engine

	owner    *Service
	wbufPool *BufferPool
	conns    *utils.SafeMap[int, *ConnContext]
	host     string
}

func newWsServer(owner *Service, c *Config) *wsServer {
	this_ := &wsServer{
		owner:    owner,
		wbufPool: NewBufferPool(),
		conns:    utils.NewSafeMap[int, *ConnContext](),
		host:     fmt.Sprintf("tcp://%s", c.WsHost),
	}

	return this_
}

func (this_ *wsServer) Proto() Protocol {
	return Protocol_Websocket
}

func (this_ *wsServer) OnBoot(eng gnet.Engine) gnet.Action {
	this_.eng = eng
	return gnet.None
}

func (this_ *wsServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	if this_.owner.info.MaxConn > 0 && atomic.LoadInt32(&this_.owner.currConn) >= this_.owner.info.MaxConn {
		return nil, gnet.Close
	}

	cctx := getConnContext()
	cctx.Init(c, this_, "", "")

	if err := this_.owner.event.OnConnected(cctx); err != nil {
		log.Error("[%d:%v] connected failed: %v", err)
		putConnContext(cctx)
		return nil, gnet.Close
	}

	atomic.AddInt32(&this_.owner.currConn, 1)
	this_.conns.Set(c.Fd(), cctx)
	c.SetContext(cctx)
	return nil, gnet.None
}

func (this_ *wsServer) OnShutdown(eng gnet.Engine) {
	this_.conns.Range(func(key int, v *ConnContext) bool {
		connContextPool.Put(v)
		return true
	})

	this_.conns.Clear()
}

func (this_ *wsServer) OnClose(c gnet.Conn, err error) gnet.Action {
	atomic.AddInt32(&this_.owner.currConn, -1)

	cctx := c.Context().(*ConnContext)
	this_.owner.event.OnDisconnected(cctx)

	cctx.SetContext(nil)
	this_.conns.Remove(cctx.Fd())
	putConnContext(cctx)
	return gnet.None
}

func (this_ *wsServer) OnTraffic(c gnet.Conn) gnet.Action {
	cctx := c.Context().(*ConnContext)

	// 升级websocket 协议
	if !cctx.upgraded {
		return this_.upgrade(cctx)
	}

	// 读取数据
	return this_.readData(cctx)
}

func (this_ *wsServer) OnTick() (time.Duration, gnet.Action) {
	tnow := time.Now().Unix()
	timeout := this_.owner.info.Timeout

	this_.conns.Range(func(fd int, cctx *ConnContext) bool {
		if tnow-cctx.lastUpdate > timeout {
			cctx.Close()
		}
		return true
	})

	return time.Second * 15, gnet.None
}

func (this_ *wsServer) Run() error {
	return gnet.Run(this_, this_.host,
		gnet.WithMulticore(true),
		gnet.WithNumEventLoop(runtime.NumCPU()*2),
		gnet.WithReuseAddr(true),
		gnet.WithReusePort(true),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
		gnet.WithSocketSendBuffer(RECV_BUF_SIZE),
		gnet.WithSocketRecvBuffer(SEND_BUF_SIZE),
		gnet.WithLogLevel(logging.DebugLevel),
		gnet.WithTicker(true),
	)
}

func (this_ *wsServer) Stop() {
	this_.eng.Stop(context.TODO())
}

func (this_ *wsServer) Write(c *ConnContext, data []byte) error {
	buf := this_.wbufPool.Get()
	err := wsutil.WriteMessage(buf, ws.StateServerSide, ws.OpText, data)
	if err != nil {
		return nil
	}

	return c.AsyncWrite(buf.Bytes(), func(c gnet.Conn, err error) error {
		if err != nil {
			log.Error("AsyncWrite failed: %v", err)
		}
		this_.wbufPool.Put(buf)
		return nil
	})
}

func (this_ *wsServer) upgrade(cctx *ConnContext) gnet.Action {
	u := ws.Upgrader{
		OnHeader: func(key, value []byte) error {
			log.Debug("Key: %v, Value: %v", string(key), string(value))
			return nil
		},
		OnRequest: func(uri []byte) error {
			log.Debug("URI: %s", string(uri))
			return nil
		},
	}

	_, err := u.Upgrade(cctx.Conn)
	if err != nil {
		log.Error("upgrade failed: %v", err)
		return gnet.Close
	}

	cctx.upgraded = true
	return gnet.None
}

func (this_ *wsServer) readData(cctx *ConnContext) gnet.Action {
	// 读取数据
	data, err := wsutil.ReadClientText(cctx.Conn)
	if err != nil {
		if err != io.EOF {
			if werr, ok := err.(*wsutil.ClosedError); ok && werr.Code != 1000 {
				log.Error("读取数据失败: %v", err)
			}
		}

		return gnet.Close
	}

	cctx.lastUpdate = time.Now().Unix()

	msg := messagePool.Get()
	msg.Conn = cctx
	msg.Data = data

	this_.owner.messageCh <- msg
	return gnet.None
}
