package io

import (
	"io"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
)

type wsServer struct {
	*baseServer
}

func newWsServer(owner *Service, c *Config) *wsServer {
	this_ := &wsServer{}
	this_.baseServer = newBaseServer(owner, this_, c.WsHost)
	return this_
}

func (this_ *wsServer) Proto() Protocol {
	return Protocol_Websocket
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

func (this_ *wsServer) Write(c *ConnContext, data []byte) error {
	buf := this_.wbufPool.Get()
	err := wsutil.WriteMessage(buf, ws.StateServerSide, ws.OpBinary, data)
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
	data, err := wsutil.ReadClientBinary(cctx.Conn)
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
