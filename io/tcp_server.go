package io

import (
	"encoding/binary"
	"time"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type tcpServer struct {
	*baseServer

	headBlend uint32
}

func newTcpServer(owner *Service, c *Config) *tcpServer {

	this_ := &tcpServer{
		headBlend: c.HeadBlend,
	}

	this_.baseServer = newBaseServer(owner, this_, c.TcpHost)
	return this_
}

func (this_ *tcpServer) Proto() Protocol {
	return Protocol_TCP
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

	data, err := c.Next(-1)
	if err != nil {
		logging.Errorf("[%v] read failed: %v", c.RemoteAddr(), err)
		return gnet.Close
	}

	cctx := c.Context().(*ConnContext)
	cctx.lastUpdate = time.Now().Unix()

	msg := messagePool.Get()
	msg.Conn = cctx
	msg.Data = data[TCP_HEADER_SIZE:]

	this_.owner.messageCh <- msg
	return gnet.None
}

func (this_ *tcpServer) Write(c *ConnContext, data []byte) error {
	buf := this_.wbufPool.Get()
	dlen := len(data)
	header := uint32(dlen) ^ this_.headBlend
	buf.WriteUint32(header)
	buf.Write(data)

	return c.AsyncWrite(buf.Bytes(), func(c gnet.Conn, err error) error {
		if err != nil {
			log.Error("AsyncWrite failed: %v", err)
		}

		this_.wbufPool.Put(buf)
		return nil
	})
}
