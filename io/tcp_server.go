package io

import (
	"context"
	"encoding/binary"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

var (
	wbufPool = sync.Pool{
		New: func() any {
			b := make([]byte, 4096)
			return &b
		},
	}
)

func getWbuf(n int) []byte {
	bufPtr := wbufPool.Get().(*[]byte)
	buf := *bufPtr
	if len(buf) < n {
		buf = make([]byte, n)
	}
	return buf
}

func putWbuf(buf []byte) {
	if buf != nil {
		wbufPool.Put(&buf)
	}
}

type tcpServer struct {
	gnet.BuiltinEventEngine
	eng gnet.Engine

	headBlend uint32
	owner     *Service
	host      string
	writeCh   chan *message
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

	atomic.AddInt32(&this_.owner.currConn, 1)

	conn := getConn()
	conn.Init(c, this_, "", "")

	if err := this_.owner.event.OnConnected(conn); err != nil {
		log.Error("[%d:%v] connected failed: %v", err)
		putConn(conn)
		return nil, gnet.Close
	}

	c.SetContext(conn)
	return nil, gnet.None
}

func (this_ *tcpServer) OnClose(c gnet.Conn, err error) gnet.Action {
	atomic.AddInt32(&this_.owner.currConn, -1)

	conn := c.Context().(*Conn)
	this_.owner.event.OnDisconnected(conn)
	putConn(conn)
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
	msg.Conn = c.Context().(*Conn)
	msg.Data = data[TCP_HEADER_SIZE:]
	this_.owner.messageCh <- msg

	return gnet.None
}

func (this_ *tcpServer) Run() error {
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

func (this_ *tcpServer) Stop() {
	close(this_.writeCh)
	this_.eng.Stop(context.TODO())
}

func (this_ *tcpServer) writeProc(wg *sync.WaitGroup) {
	var (
		err     error
		running = int32(ServiceState_Running)
		state   = (*int32)(&this_.owner.state)
	)

	for msg := range this_.writeCh {
		if atomic.LoadInt32(state) == running {
			_, err = msg.Conn.Conn.Write(msg.Data)
			if err != nil {
				log.Error("[%v] write failed: %v", err)
			}
			putWbuf(msg.Data)
		}
		messagePool.Put(msg)
	}
	wg.Done()
}

func (this_ *tcpServer) Write(c *Conn, data []byte) error {
	dlen := len(data)
	buf := getWbuf(dlen + TCP_HEADER_SIZE)
	binary.BigEndian.PutUint32(buf[:TCP_HEADER_SIZE], uint32(dlen)^this_.headBlend)
	copy(buf[TCP_HEADER_SIZE:], data)
	msg := messagePool.Get()
	msg.Conn = c
	msg.Data = buf
	this_.writeCh <- msg
	return nil
}
