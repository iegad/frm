package nw

import (
	"context"
	"fmt"
	"time"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

const HEART_BEAT_INTERVAL = 15 * time.Second // 心跳检查周期

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

	owner    *Service        // 所属服务
	server   IServer         // 实际的服务
	host     string          // 监听地址
	cctxPool connContextPool // ConnContext 对象池
	wbufPool BufferPool      // 写对象池
	msgPool  messagePool     // 消息对象池
}

// newBaseServer 构造函数
func newBaseServer(owner *Service, server IServer, host string) *baseServer {
	return &baseServer{
		owner:    owner,
		wbufPool: NewBufferPool(),
		msgPool:  newMessagePool(),
		server:   server,
		host:     fmt.Sprintf("tcp://%v", host),
		cctxPool: newConnContextPool(),
	}
}

// Proto 协议
func (this_ *baseServer) Proto() Protocol {
	return Protocol_None
}

// Host 监听地址
func (this_ *baseServer) Host() string {
	return this_.host
}

// OnBoot 启动事件
func (this_ *baseServer) OnBoot(eng gnet.Engine) gnet.Action {
	this_.eng = eng
	return gnet.None
}

// OnOpen 客户端连接事件
func (this_ *baseServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	if this_.owner.info.MaxConn > 0 && this_.owner.conns.Count() >= this_.owner.info.MaxConn {
		return nil, gnet.Close
	}

	cctx := this_.cctxPool.get()
	cctx.Init(c, this_)

	if err := this_.owner.event.OnConnected(cctx); err != nil {
		log.Error("[%d:%v] connected failed: %v", err)
		this_.cctxPool.put(cctx)
		return nil, gnet.Close
	}

	// 将连接上下文放入 conns 集中
	this_.owner.conns.Set(cctx.Fd(), cctx)
	return nil, gnet.None
}

// OnClose 客户端连接断开事件
func (this_ *baseServer) OnClose(c gnet.Conn, err error) gnet.Action {
	cctx := c.Context().(*ConnContext)

	this_.owner.conns.Remove(cctx.Fd())
	this_.owner.event.OnDisconnected(cctx)
	this_.cctxPool.put(cctx)
	return gnet.None
}

// OnTick 定时任务, 主要作用是心跳检查
func (this_ *baseServer) OnTick() (time.Duration, gnet.Action) {
	var (
		tnow    = time.Now().Unix()
		timeout = this_.owner.info.Timeout
	)

	this_.owner.conns.Range(func(fd int, cctx *ConnContext) bool {
		if tnow-cctx.lastUpdate > timeout {
			cctx.Close()
		}
		return true
	})

	return HEART_BEAT_INTERVAL, gnet.None
}

// Stop 停止服务
func (this_ *baseServer) Stop() {
	this_.eng.Stop(context.TODO())
}

// Write 向客户端发送数据
func (this_ *baseServer) Write(cctx *ConnContext, data []byte) error {
	return this_.server.Write(cctx, data)
}

// Run 启动服务
func Run(server IServer) error {
	return gnet.Run(server, server.Host(),
		gnet.WithMulticore(true),
		gnet.WithReuseAddr(true),
		gnet.WithReusePort(true),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
		gnet.WithSocketSendBuffer(SEND_BUF_SIZE),
		gnet.WithSocketRecvBuffer(RECV_BUF_SIZE),
		gnet.WithLogLevel(logging.InfoLevel),
		gnet.WithTicker(true),
		gnet.WithTCPKeepAlive(time.Second*30),
		gnet.WithTCPKeepCount(2),
		gnet.WithTCPKeepInterval(time.Second*10),
		gnet.WithReadBufferCap(4096),
		gnet.WithLockOSThread(false),
	)
}
