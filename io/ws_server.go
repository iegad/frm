package io

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/gox/frm/log"
	"github.com/gox/frm/utils"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

var (
	wsWbufPool = utils.NewPool[bytes.Buffer]()
)

type wsServer struct {
	gnet.BuiltinEventEngine
	eng gnet.Engine

	owner *Service
	host  string
}

func newWsServer(owner *Service, c *Config) *wsServer {
	this_ := &wsServer{
		owner: owner,
		host:  fmt.Sprintf("tcp://%s", c.WsHost),
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
	if this_.owner.maxConn > 0 && atomic.LoadInt32(&this_.owner.currConn) >= this_.owner.maxConn {
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
	c.SetContext(cctx)
	return nil, gnet.None
}

func (this_ *wsServer) OnClose(c gnet.Conn, err error) gnet.Action {
	atomic.AddInt32(&this_.owner.currConn, -1)

	cctx := c.Context().(*ConnContext)
	this_.owner.event.OnDisconnected(cctx)

	putConnContext(cctx)
	c.SetContext(nil)
	return gnet.None
}

func (this_ *wsServer) OnTraffic(c gnet.Conn) gnet.Action {
	cctx := c.Context().(*ConnContext)

	if !cctx.upgraded {
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

		_, err := u.Upgrade(c)
		if err != nil {
			log.Error("upgrade failed: %v", err)
			return gnet.Close
		}

		cctx.upgraded = true
		return gnet.None
	}

	data, err := wsutil.ReadClientText(c)
	if err != nil {
		log.Error("read failed: %v", err)
		return gnet.Close
	}

	msg := messagePool.Get()
	msg.Conn = cctx
	msg.Data = data

	this_.owner.messageCh <- msg

	return gnet.None
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
	)
}

func (this_ *wsServer) Stop() {
	this_.eng.Stop(context.TODO())
}

func (this_ *wsServer) Write(c *ConnContext, data []byte) error {
	buf := wsWbufPool.Get()
	err := wsutil.WriteServerMessage(buf, ws.OpText, data)
	if err != nil {
		return nil
	}
	return c.AsyncWrite(buf.Bytes(), func(c gnet.Conn, err error) error {
		if err != nil {
			log.Error("AsyncWrite failed: %v", err)
		}

		buf.Reset()
		wsWbufPool.Put(buf)
		return nil
	})
}
