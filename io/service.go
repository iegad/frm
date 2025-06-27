package io

import (
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"

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
	RECV_BUF_SIZE = 1024 * 1024 * 2 // 读缓冲区
	SEND_BUF_SIZE = 1024 * 1024 * 2 // 写缓冲区

	TCP_HEADER_SIZE = 4                       // 消息头长度
	TCP_MAX_SIZE    = uint32(1024 * 1024 * 2) // 消息体最大长度
)

// Message 消息
type Message struct {
	Context *ConnContext
	Data    []byte
}

var messagePool = utils.NewPool[Message]()

// IServiceEvent 服务事件
type IServiceEvent interface {
	OnInit(*Service) error
	OnConnected(*ConnContext) error
	OnDisconnected(*ConnContext)
	OnStopped(*Service)
	OnData(*ConnContext, []byte) error
}

// Config 服务配置
type Config struct {
	TcpHost   string `json:"tcp_host,omitempty"`
	WsHost    string `json:"ws_host,omitempty"`
	MaxConn   int32  `json:"max_conn"`
	HeadBlend uint32 `json:"-"`
	Timeout   int64  `json:"timeout"`
}

// serverInfo 服务信息
type serverInfo struct {
	State    int32  `json:"state"`
	MaxConn  int32  `json:"max_conn"`
	CurrConn int32  `json:"curr_conn"`
	Timeout  int64  `json:"timeout"`
	TcpHost  string `json:"tcp_host,omitempty"`
	WsHost   string `json:"ws_host,omitempty"`
}

func (this_ *serverInfo) String() string {
	jstr, _ := json.Marshal(this_)
	return string(jstr)
}

// Service 网络服务
type Service struct {
	currConn  int32                             // 当前连接数
	info      *serverInfo                       // 服务信息
	tcpSvr    *tcpServer                        // tcp服务
	wsSvr     *wsServer                         // websocket服务
	conns     *utils.SafeMap[int, *ConnContext] // 客户端连接池
	messageCh chan *Message                     // 消息管道
	event     IServiceEvent                     // 事件
	wg        sync.WaitGroup                    // 协程同步
}

func NewService(c *Config, event IServiceEvent) *Service {
	if c.MaxConn <= 0 {
		log.Fatal("config.max_conn is invalid")
		return nil
	}

	if len(c.TcpHost) == 0 && len(c.WsHost) == 0 {
		log.Fatal("must have one listner address")
		return nil
	}

	messageCh := make(chan *Message, c.MaxConn*10000)

	this_ := &Service{
		conns:     utils.NewSafeMap[int, *ConnContext](),
		messageCh: messageCh,
		info: &serverInfo{
			State:   ServiceState_Stopped,
			MaxConn: c.MaxConn,
			TcpHost: c.TcpHost,
			WsHost:  c.WsHost,
			Timeout: c.Timeout,
		},
		event: event,
	}

	if len(c.TcpHost) > 0 {
		this_.tcpSvr = newTcpServer(this_, c)
	}

	if len(c.WsHost) > 0 {
		this_.wsSvr = newWsServer(this_, c)
	}

	return this_
}

func (this_ *Service) CurrConn() int32 {
	return int32(this_.conns.Count())
}

func (this_ *Service) Run(nproc int) {
	if !atomic.CompareAndSwapInt32(&this_.info.State, ServiceState_Stopped, ServiceState_Running) {
		return
	}

	err := this_.event.OnInit(this_)
	if err != nil {
		return
	}

	if nproc <= 0 {
		nproc = runtime.NumCPU()
	}

	this_.wg.Add(nproc)
	for i := 0; i < nproc; i++ {
		go this_.messageLoop(&this_.wg)
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

	close(this_.messageCh)
}

func (this_ *Service) messageLoop(wg *sync.WaitGroup) {
	var state = &this_.info.State

	for msg := range this_.messageCh {
		if atomic.LoadInt32(state) == ServiceState_Running {
			this_.messageHandle(msg)
		}
		messagePool.Put(msg)
	}

	wg.Done()
}

func (this_ *Service) messageHandle(msg *Message) {
	err := this_.event.OnData(msg.Context, msg.Data)
	if err != nil {
		msg.Context.Close()
	}
}
