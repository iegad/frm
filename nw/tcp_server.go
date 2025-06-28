package nw

import (
	"encoding/binary"
	"time"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
)

// tcpServer TCP服务器
//
// TCP 协议格式：HEADER[sizeof(uint32)] + DATA
type tcpServer struct {
	baseServer
	headBlend uint32
	msgPool   messagePool // 消息对象池
}

// NewTcpServer 创建一个新的 TCP 服务器
//
//   - owner: 所属服务
//   - c: 配置
//
// 返回一个新的 tcpServer 实例
func newTcpServer(owner *Service, c *Config) *tcpServer {
	this_ := &tcpServer{
		headBlend: c.HeadBlend,
		msgPool:   newMessagePool(),
	}

	this_.baseServer = *newBaseServer(owner, this_, c.TcpHost)
	return this_
}

// Proto 返回服务器的协议类型
func (this_ *tcpServer) Proto() Protocol {
	return Protocol_TCP
}

// OnTraffic 处理客户端数据
func (this_ *tcpServer) OnTraffic(c gnet.Conn) gnet.Action {
	// 获取客户端缓冲区大小
	n := c.InboundBuffered()

	if n < TCP_HEADER_SIZE {
		return gnet.None
	}

	// 查看数据头
	data, _ := c.Peek(TCP_HEADER_SIZE)
	dlen := binary.BigEndian.Uint32(data) ^ this_.headBlend
	if dlen > MESSAGE_MAX_SIZE {
		log.Error("[%d:%v]read data over max data length", c.Fd(), c.RemoteAddr())
		return gnet.Close
	}

	mlen := int(dlen) + TCP_HEADER_SIZE
	if n < mlen {
		return gnet.None
	}

	// 查看消息体数据
	data, _ = c.Peek(mlen)
	// 消费数据
	c.Discard(mlen)

	cctx := c.Context().(*ConnContext)
	cctx.lastUpdate = time.Now().Unix()

	msg := this_.msgPool.Get(cctx, data[TCP_HEADER_SIZE:])
	this_.owner.messageCh <- msg
	return gnet.None
}

func (this_ *tcpServer) Write(cctx *ConnContext, data []byte) error {
	buf := this_.wbufPool.Get()
	dlen := len(data)
	header := uint32(dlen) ^ this_.headBlend
	buf.WriteUint32BE(header)
	buf.Write(data)

	return cctx.c.AsyncWrite(buf.Bytes(), func(c gnet.Conn, err error) error {
		if err != nil {
			log.Error("AsyncWrite failed: %v", err)
		}

		this_.wbufPool.Put(buf)
		return nil
	})
}

func (this_ *tcpServer) PutMessage(msg *message) {
	this_.msgPool.Put(msg)
}
