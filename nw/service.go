package nw

import (
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/gox/frm/log"
	"github.com/gox/frm/utils"
)

// Protocol 网络协议
type Protocol int32

const (
	Protocol_None      Protocol = 0
	Protocol_TCP       Protocol = 1
	Protocol_Websocket Protocol = 2
	Protocol_UDP       Protocol = 3
)

func (this_ Protocol) String() string {
	switch this_ {
	case Protocol_TCP:
		return "tcp"
	case Protocol_Websocket:
		return "websocket"
	case Protocol_UDP:
		return "udp"
	}

	return "none"
}

const (
	ServiceState_Stopped  int32 = 0 // 服务器状态: 停止
	ServiceState_Stopping int32 = 1 // 服务器状态: 正在停止中
	ServiceState_Running  int32 = 2 // 服务器状态: 运行
)

// 常量定义
const (
	RECV_BUF_SIZE = 1024 * 1024 * 2 // 读缓冲区 2M
	SEND_BUF_SIZE = 1024 * 1024 * 2 // 写缓冲区 2M

	TCP_HEADER_SIZE  = int(unsafe.Sizeof(uint32(0))) // 消息头长度
	MESSAGE_MAX_SIZE = uint32(1024 * 1024 * 2)       // 消息体最大长度

	DEFAULT_CHAN_SIZE = 1000000
)

// IServiceEvent 服务事件
type IServiceEvent interface {
	OnInit(*Service) error             // 初始化事件
	OnConnected(*ConnContext) error    // 客户端连接事件
	OnDisconnected(*ConnContext)       // 客户端连接断开事件
	OnStopped(*Service)                // 服务停止事件
	OnData(*ConnContext, []byte) error // 消息事件
}

// Config 服务配置
type Config struct {
	TcpHost   string `json:"tcp_host,omitempty"` // tcp 监听地址
	WsHost    string `json:"ws_host,omitempty"`  // websocket 监听地址
	HeadBlend uint32 `json:"-"`                  // tcp 消息头混合值
	MaxConn   int    `json:"max_conn"`           // 最大连接数
	Timeout   int64  `json:"timeout"`            // 客户端超时值
}

// serverInfo 服务信息
type serverInfo struct {
	State    int32  `json:"state"`              // 服务状态
	CurrConn int    `json:"curr_conn"`          // 当前连接数
	MaxConn  int    `json:"max_conn"`           // 最大连接数
	Timeout  int64  `json:"timeout"`            // 超时值
	TcpHost  string `json:"tcp_host,omitempty"` // cp 监听地址
	WsHost   string `json:"ws_host,omitempty"`  // websocket 监听地址
}

func (this_ *serverInfo) String() string {
	jstr, _ := json.Marshal(this_)
	return string(jstr)
}

// Service 网络服务
type Service struct {
	info    *serverInfo                       // 服务信息
	tcpSvr  *tcpServer                        // tcp服务
	wsSvr   *wsServer                         // websocket服务
	conns   *utils.SafeMap[int, *ConnContext] // 客户端连接池
	wkrPool []*messageWorker                  // 消息工作池
	event   IServiceEvent                     // 事件
	wg      sync.WaitGroup                    // 协程同步
}

// NewService 创建一个新的 Service
//   - c: 服务配置
//   - event: 服务事件接口
//
// 注意: 必须至少提供一个监听地址 (TcpHost 或 WsHost)
func NewService(c *Config, event IServiceEvent) *Service {
	if len(c.TcpHost) == 0 && len(c.WsHost) == 0 {
		log.Fatal("must bind a listen address")
		return nil
	}

	if c.MaxConn < 0 {
		c.MaxConn = 0
	}

	// 创建服务对象
	this_ := &Service{
		conns: utils.NewSafeMap[int, *ConnContext](),
		info: &serverInfo{
			State:   ServiceState_Stopped,
			MaxConn: c.MaxConn,
			TcpHost: c.TcpHost,
			WsHost:  c.WsHost,
			Timeout: c.Timeout,
		},
		event: event,
	}

	// 创建消息工作池
	n := runtime.NumCPU()
	for i := 0; i < n; i++ {
		this_.wkrPool = append(this_.wkrPool, newMessageWorker(this_.messageHandle))
	}

	if len(c.TcpHost) > 0 {
		this_.tcpSvr = newTcpServer(this_, c)
	}

	if len(c.WsHost) > 0 {
		this_.wsSvr = newWsServer(this_, c)
	}

	return this_
}

func (this_ *Service) String() string {
	this_.info.CurrConn = this_.conns.Count()
	return this_.info.String()
}

// TcpHost TCP 监听地址
func (this_ *Service) TcpHost() string {
	if this_.tcpSvr != nil {
		return this_.tcpSvr.host
	}

	return ""
}

// WsHost websocket 监听地址
func (this_ *Service) WsHost() string {
	if this_.wsSvr != nil {
		return this_.wsSvr.host
	}

	return ""
}

// CurrConn 当前在线人数
func (this_ *Service) CurrConn() int {
	return this_.conns.Count()
}

// Run 启动服务
func (this_ *Service) Run() {
	if !atomic.CompareAndSwapInt32(&this_.info.State, ServiceState_Stopped, ServiceState_Running) {
		return
	}

	this_.wg.Add(len(this_.wkrPool))
	for _, wkr := range this_.wkrPool {
		go wkr.run(&this_.wg)
	}

	err := this_.event.OnInit(this_)
	if err != nil {
		return
	}

	if this_.tcpSvr != nil {
		this_.wg.Add(1)
		go func() {
			err := Run(this_.tcpSvr)
			if err != nil {
				log.Error("tcp server run failed: %v", err)
			}
			this_.wg.Done()
		}()
	}

	if this_.wsSvr != nil {
		this_.wg.Add(1)
		go func() {
			err := Run(this_.wsSvr)
			if err != nil {
				log.Error("ws server run failed: %v", err)
			}
			this_.wg.Done()
		}()
	}

	this_.wg.Wait()
	this_.event.OnStopped(this_)
	atomic.StoreInt32(&this_.info.State, ServiceState_Stopped)
}

// Stop 停止服务
func (this_ *Service) Stop() {
	if !atomic.CompareAndSwapInt32(&this_.info.State, ServiceState_Running, ServiceState_Stopping) {
		return
	}

	if this_.tcpSvr != nil {
		this_.tcpSvr.Stop()
	}

	if this_.wsSvr != nil {
		this_.wsSvr.Stop()
	}

	for _, wkr := range this_.wkrPool {
		wkr.stop()
	}
}

// messageHandle 消息处理
func (this_ *Service) messageHandle(msg *message) {
	err := this_.event.OnData(msg.cctx, msg.data())
	if err != nil {
		msg.cctx.Close()
	}
	msg.release()
}

func (this_ *Service) pushMessage(msg *message) {
	this_.wkrPool[msg.cctx.Fd()%len(this_.wkrPool)].push(msg)
}
