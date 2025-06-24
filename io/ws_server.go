package io

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type wsServer struct {
	gnet.BuiltinEventEngine
	eng     gnet.Engine
	owner   *Service
	host    string
	writeCh chan *message
}

func newWsServer(owner *Service, c *Config) *wsServer {
	this_ := &wsServer{
		owner: owner,
		host:  fmt.Sprintf("tcp://%s", c.WsHost),
	}

	return this_
}

func (this_ *wsServer) OnBoot(eng gnet.Engine) gnet.Action {
	this_.eng = eng
	return gnet.None
}

func (this_ *wsServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	if this_.owner.maxConn > 0 && atomic.LoadInt32(&this_.owner.currConn) >= this_.owner.maxConn {
		return nil, gnet.Close
	}

	atomic.AddInt32(&this_.owner.currConn, 1)
	return nil, gnet.None
}

func (this_ *wsServer) OnClose(c gnet.Conn, err error) gnet.Action {
	// TODO: 连接断开事件

	atomic.AddInt32(&this_.owner.currConn, -1)
	logging.Debugf("[%d:%v] has disconnected", c.Fd(), c.RemoteAddr())
	return gnet.None
}

func (this_ *wsServer) OnTraffic(c gnet.Conn) gnet.Action {
	data, err := c.Next(-1)
	if err != nil {
		logging.Errorf("[%v] read failed: %v", c.RemoteAddr(), err)
		return gnet.Close
	}

	if data == nil {
		return gnet.None
	}

	if _, upgraded := c.Context().(bool); !upgraded {
		_, err = ws.Upgrade(c)
		if err != nil {
			log.Error("ws upgrade failed: %v", err)
			return gnet.Close
		}

		c.SetContext(true)
		return gnet.None
	}

	data, err = wsutil.ReadClientBinary(c)
	if err != nil {
		log.Error("read failed: %v", err)
		return gnet.Close
	}

	msg := messagePool.Get()
	msg.Conn = c.Context().(*Conn)
	msg.Data = data

	this_.owner.messageCh <- msg
	return gnet.None
}

func (this_ *wsServer) Run() error {
	this_.writeCh = make(chan *message, this_.owner.maxConn*100)

	n := runtime.NumCPU()
	this_.owner.wg.Add(n)
	for i := 0; i < n; i++ {
		go this_.writeProc(&this_.owner.wg)
	}

	return gnet.Run(this_, this_.host,
		gnet.WithMulticore(true),
		gnet.WithReuseAddr(true),
		gnet.WithReusePort(true),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
		gnet.WithSocketSendBuffer(RECV_BUF_SIZE),
		gnet.WithSocketRecvBuffer(SEND_BUF_SIZE),
		gnet.WithLogLevel(logging.DebugLevel),
	)
}

func (this_ *wsServer) Stop() {
	close(this_.writeCh)
	this_.eng.Stop(context.TODO())
}

func (this_ *wsServer) writeProc(wg *sync.WaitGroup) {
	var (
		err     error
		running = int32(ServiceState_Running)
		state   = (*int32)(&this_.owner.state)
	)

	for msg := range this_.writeCh {
		if atomic.LoadInt32(state) == running {
			err = wsutil.WriteServerBinary(msg.Conn.Conn, msg.Data)
			if err != nil {
				log.Error("[%v] write failed: %v", err)
			}
		}
		messagePool.Put(msg)
	}
	wg.Done()
}

func (this_ *wsServer) Write(c *Conn, data []byte) error {
	msg := messagePool.Get()
	msg.Conn = c
	msg.Data = data

	this_.writeCh <- msg
	return nil
}
