package io

import (
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gox/frm/log"
)

const (
	ServerState_Stopped  ServerState = 0
	ServerState_Stopping ServerState = 1
	ServerState_Running  ServerState = 2

	RECV_BUF_SIZE = 1024 * 1024 * 2
	SEND_BUF_SIZE = 1024 * 1024 * 2
)

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
	wg        sync.WaitGroup
}

func NewServer(c *Config) *Server {
	messageCh := make(chan *message, c.MaxConn*100)

	this_ := &Server{
		messageCh: messageCh,
		info: &serverInfo{
			State:   ServerState_Stopped,
			MaxConn: c.MaxConn,
			TcpHost: c.TcpHost,
			WsHost:  c.WsHost,
		},
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
	// TODO
}
