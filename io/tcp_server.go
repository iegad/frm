package io

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/gox/frm/log"
	"github.com/gox/frm/utils"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

var tcpBufPool = utils.NewPool[bytes.Buffer]()

type tcpServer struct {
	gnet.BuiltinEventEngine
	eng gnet.Engine

	headBlend uint32
	owner     *Service
	host      string
}

func newTcpServer(owner *Service, c *Config) *tcpServer {
	return &tcpServer{
		owner:     owner,
		headBlend: c.HeadBlend,
		host:      fmt.Sprintf("tcp://%v", c.TcpHost),
	}
}

func (this_ *tcpServer) Proto() Protocol {
	return Protocol_TCP
}

func (this_ *tcpServer) OnBoot(eng gnet.Engine) gnet.Action {
	this_.eng = eng
	return gnet.None
}

func (this_ *tcpServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
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
	cctx.upgraded = true
	return nil, gnet.None
}

func (this_ *tcpServer) OnClose(c gnet.Conn, err error) gnet.Action {
	atomic.AddInt32(&this_.owner.currConn, -1)

	cctx := c.Context().(*ConnContext)
	this_.owner.event.OnDisconnected(cctx)

	putConnContext(cctx)
	c.SetContext(nil)
	return gnet.None
}

func (this_ *tcpServer) OnTraffic(c gnet.Conn) gnet.Action {
	if c.InboundBuffered() < 4 {
		return gnet.None
	}

	hbuf, err := c.Peek(TCP_HEADER_SIZE)
	if err != nil {
		log.Error("[%d:%v]read header failed: %v", c.Fd(), c.RemoteAddr(), err)
		return gnet.Close
	}

	dlen := binary.BigEndian.Uint32(hbuf) ^ this_.headBlend
	if dlen > TCP_MAX_SIZE {
		log.Error("[%d:%v]read data over max data length", c.Fd(), c.RemoteAddr())
		return gnet.Close
	}

	mlen := int(dlen) + TCP_HEADER_SIZE
	if c.InboundBuffered() < mlen {
		return gnet.None
	}

	data, err := c.Next(mlen)
	if err != nil {
		logging.Errorf("[%v] read failed: %v", c.RemoteAddr(), err)
		return gnet.Close
	}

	msg := messagePool.Get()
	msg.Conn = c.Context().(*ConnContext)
	msg.Data = data[TCP_HEADER_SIZE:]
	this_.owner.messageCh <- msg

	return gnet.None
}

func (this_ *tcpServer) Run() error {
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

func (this_ *tcpServer) Stop() {
	this_.eng.Stop(context.TODO())
}

func (this_ *tcpServer) Write(c *ConnContext, data []byte) error {
	buf := tcpBufPool.Get()
	dlen := len(data)
	header := uint32(dlen) ^ this_.headBlend
	_ = binary.Write(buf, binary.BigEndian, header)

	buf.Write(data)

	return c.AsyncWrite(buf.Bytes(), func(c gnet.Conn, err error) error {
		if err != nil {
			log.Error("AsyncWrite failed: %v", err)
		}

		tcpBufPool.Put(buf)
		return nil
	})
}
