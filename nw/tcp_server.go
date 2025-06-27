package nw

import (
	"encoding/binary"
	"time"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
)

type tcpServer struct {
	baseServer
	headBlend uint32
}

func newTcpServer(owner *Service, c *Config) *tcpServer {
	this_ := &tcpServer{
		headBlend: c.HeadBlend,
	}

	this_.baseServer = *newBaseServer(owner, this_, c.TcpHost)
	return this_
}

func (this_ *tcpServer) Proto() Protocol {
	return Protocol_TCP
}

func (this_ *tcpServer) OnTraffic(c gnet.Conn) gnet.Action {
	n := c.InboundBuffered()

	if n < TCP_HEADER_SIZE {
		return gnet.None
	}

	data, _ := c.Peek(TCP_HEADER_SIZE)
	dlen := binary.BigEndian.Uint32(data) ^ this_.headBlend
	if dlen > TCP_MAX_SIZE {
		log.Error("[%d:%v]read data over max data length", c.Fd(), c.RemoteAddr())
		return gnet.Close
	}

	mlen := int(dlen) + TCP_HEADER_SIZE
	if n < mlen {
		return gnet.None
	}

	data, _ = c.Peek(mlen)
	c.Discard(mlen)

	cctx := c.Context().(*ConnContext)
	cctx.lastUpdate = time.Now().Unix()

	buf := make([]byte, dlen)
	copy(buf, data[TCP_HEADER_SIZE:])
	msg := messagePool.Get()
	msg.Context = cctx
	msg.Data = buf

	this_.owner.messageCh <- msg
	return gnet.None
}

func (this_ *tcpServer) Write(cctx *ConnContext, data []byte) error {
	buf := this_.wbufPool.Get()
	dlen := len(data)
	header := uint32(dlen) ^ this_.headBlend
	buf.WriteUint32(header)
	buf.Write(data)

	return cctx.asyncWrite(buf.Bytes(), func(c gnet.Conn, err error) error {
		if err != nil {
			log.Error("AsyncWrite failed: %v", err)
		}

		this_.wbufPool.Put(buf)
		return nil
	})
}
