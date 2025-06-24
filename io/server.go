package io

import (
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
)

// 常量定义
const (
	ServerState_Stopped  ServerState = 0 // 服务器状态: 停止
	ServerState_Stopping ServerState = 1 // 服务器状态: 正在停止中
	ServerState_Running  ServerState = 2 // 服务器状态: 运行

	RECV_BUF_SIZE = 1024 * 1024 * 2
	SEND_BUF_SIZE = 1024 * 1024 * 2
)

type IServerEvent interface {
	OnInit(*Server) error
	OnConnected(gnet.Conn) error
	OnDisconnected(gnet.Conn)
	OnStopped(*Server)
	OnData(gnet.Conn, []byte) error
}

type Config struct {
	TcpHost   string `json:"tcp_host,omitempty"`
	WsHost    string `json:"ws_host,omitempty"`
	MaxConn   int32  `json:"max_conn"`
	HeadBlend uint32 `json:"-"`
}

type serverInfo struct {
	State    ServerState `json:"state"`
	MaxConn  int32       `json:"max_conn"`
	CurrConn int32       `json:"curr_conn"`
	TcpHost  string      `json:"tcp_host,omitempty"`
	WsHost   string      `json:"ws_host,omitempty"`
}

func (this_ *serverInfo) String() string {
	jstr, _ := json.Marshal(this_)
	return string(jstr)
}

type Server struct {
	state     ServerState // use int32 for atomic operations
	maxConn   int32
	currConn  int32
	tcpSvr    *tcpServer
	wsSvr     *wsServer
	messageCh chan *message
	info      *serverInfo
	event     IServerEvent
	wg        sync.WaitGroup
}

func NewServer(c *Config, event IServerEvent) *Server {
	messageCh := make(chan *message, c.MaxConn*100)

	this_ := &Server{
		messageCh: messageCh,
		info: &serverInfo{
			State:   ServerState_Stopped,
			MaxConn: c.MaxConn,
			TcpHost: c.TcpHost,
			WsHost:  c.WsHost,
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

func (this_ *Server) Run(nproc int) {
	if !atomic.CompareAndSwapInt32((*int32)(&this_.state), int32(ServerState_Stopped), int32(ServerState_Running)) {
		return
	}

	err := this_.event.OnInit(this_)
	if err != nil {
		log.Fatal("server inited failed: %v", err)
	}

	if nproc <= 0 {
		nproc = runtime.NumCPU()
	}

	this_.wg.Add(nproc)
	for i := 0; i < nproc; i++ {
		go this_.messageProc(&this_.wg)
	}

	if this_.tcpSvr != nil {
		this_.wg.Add(1)
		go func() {
			err := this_.tcpSvr.Run()
			if err != nil {
				log.Error("tcp server run failed: %v", err)
			}
			this_.wg.Done()
		}()
	}

	if this_.wsSvr != nil {
		this_.wg.Add(1)
		go func() {
			err := this_.wsSvr.Run()
			if err != nil {
				log.Error("ws server run failed: %v", err)
			}
			this_.wg.Done()
		}()
	}

	this_.wg.Wait()
	this_.state = ServerState_Stopped
	this_.event.OnStopped(this_)
}

func (this_ *Server) Stop() {
	if !atomic.CompareAndSwapInt32((*int32)(&this_.state), int32(ServerState_Running), int32(ServerState_Stopping)) {
		return
	}

	this_.info.State = ServerState_Stopping
	if this_.tcpSvr != nil {
		this_.tcpSvr.Stop()
	}

	if this_.wsSvr != nil {
		this_.wsSvr.Stop()
	}

	close(this_.messageCh)
}

func (this_ *Server) messageProc(wg *sync.WaitGroup) {
	var (
		running = int32(ServerState_Running)
		state   = (*int32)(&this_.state)
	)

	for msg := range this_.messageCh {
		if atomic.LoadInt32(state) == running {
			this_.messageHandle(msg)
		}
		messagePool.Put(msg)
	}

	wg.Done()
}

func (this_ *Server) messageHandle(msg *message) {
	err := this_.event.OnData(msg.Conn, msg.Data)
	if err != nil {
		msg.Conn.Close()
	}
}
