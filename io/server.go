package io

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type ServerState int32

const (
	ServerState_Stopped ServerState = 0
	ServerState_Running ServerState = 1

	RECV_BUF_SIZE = 1024 * 1024 * 2
	SEND_BUF_SIZE = 1024 * 1024 * 2
)

var (
	ErrConifgNil     = errors.New("config is invalid")
	ErrConfigAddr    = errors.New("config.address is invalid")
	ErrConfigMaxConn = errors.New("config.max_conn is invalid")
	ErrConfigBlend   = errors.New("config.head_blend is invalid")
)

type serverInfo struct {
	Running    ServerState `json:"running"`
	MaxConn    int32       `json:"max_conn"`
	CurrConn   int32       `json:"curr_conn"`
	TcpAddress string      `json:"tcp_address,omitempty"`
	WsAddress  string      `json:"ws_address,omitempty"`
}

func (this_ *serverInfo) String() string {
	jstr, _ := json.Marshal(this_)
	return string(jstr)
}

type Config struct {
	TcpAddress string `json:"tcp_address,omitempty"`
	WsAddress  string `json:"tcp_address,omitempty"`
	MaxConn    int32  `json:"max_conn"`
	HeadBlend  uint32 `json:"-"`
}

type Server struct {
	gnet.BuiltinEventEngine
	eng gnet.Engine

	state    ServerState // 运行状态
	currConn int32       // 当前连接数

	headBlend uint32

	messageC chan *message

	info *serverInfo
	wg   sync.WaitGroup
}

func NewServer(c *Config) (*Server, error) {
	if c == nil {
		return nil, ErrConifgNil
	}

	if len(c.TcpAddress) == 0 && len(c.WsAddress) == 0 {
		return nil, ErrConfigAddr
	}

	if c.MaxConn < 0 {
		return nil, ErrConfigMaxConn
	}

	if c.HeadBlend == 0 {
		return nil, ErrConfigBlend
	}

	return &Server{
		state:     ServerState_Stopped,
		currConn:  0,
		headBlend: c.HeadBlend,
		info: &serverInfo{
			Running:    ServerState_Stopped,
			MaxConn:    c.MaxConn,
			CurrConn:   0,
			TcpAddress: c.TcpAddress,
			WsAddress:  c.WsAddress,
		},
	}, nil
}

func (this_ *Server) String() string {
	this_.info.CurrConn = atomic.LoadInt32(&this_.currConn)
	return this_.info.String()
}

func (this_ *Server) OnBoot(eng gnet.Engine) gnet.Action {
	this_.eng = eng

	// TODO: 初始化事件

	return gnet.None
}

func (this_ *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	if atomic.LoadInt32(&this_.currConn) >= this_.info.MaxConn {
		return nil, gnet.Close
	}

	logging.Debugf("[%d:%v] has connected", c.Fd(), c.RemoteAddr())

	// TODO 连接事件
	return nil, gnet.None
}

func (this_ *Server) OnClose(c gnet.Conn, err error) gnet.Action {
	// TODO: 连接断开事件

	logging.Debugf("[%d:%v] has disconnected", c.Fd(), c.RemoteAddr())
	return gnet.None
}

func (this_ *Server) OnTraffic(c gnet.Conn) gnet.Action {
	data, err := c.Next(-1)
	if err != nil {
		logging.Errorf("[%v] read failed: %v", c.RemoteAddr(), err)
		return gnet.Close
	}

	this_.messageC <- &message{
		Conn: c,
		Data: data,
	}

	return gnet.None
}

func (this_ *Server) OnShutdown(eng gnet.Engine) {
	// TODO: 服务关闭事件
	logging.Infof("服务已关闭")
}

func (this_ *Server) Run() error {
	if !atomic.CompareAndSwapInt32((*int32)(&this_.state), int32(ServerState_Stopped), int32(ServerState_Running)) {
		return nil
	}

	this_.state = ServerState_Running
	this_.messageC = make(chan *message, this_.info.MaxConn*10)

	n := runtime.NumCPU()
	this_.wg.Add(n)
	for i := 0; i < n; i++ {
		go this_.messageWorkProc()
	}

	err := gnet.Run(this_, fmt.Sprintf("tcp://%s", this_.info.TcpAddress),
		gnet.WithMulticore(true),
		gnet.WithReuseAddr(true),
		gnet.WithReusePort(true),
		gnet.WithSocketSendBuffer(RECV_BUF_SIZE),
		gnet.WithSocketRecvBuffer(SEND_BUF_SIZE),
		gnet.WithLogLevel(logging.DebugLevel),
	)

	if err != nil {
		logging.Errorf("run failed: %v", err)
		return err
	}

	return nil
}

func (this_ *Server) Stop() {
	if !atomic.CompareAndSwapInt32((*int32)(&this_.state), int32(ServerState_Running), int32(ServerState_Stopped)) {
		return
	}

	this_.eng.Stop(context.TODO())

	close(this_.messageC)
	this_.messageC = nil
}

func (this_ *Server) messageWorkProc() {
	for msg := range this_.messageC {
		this_.messageHandle(msg)
	}

	this_.wg.Done()
}

func (this_ *Server) messageHandle(msg *message) {
	// TODO: 消息事件
	err := msg.Conn.AsyncWrite(msg.Data, nil)
	if err != nil {
		logging.Errorf("[%v] asyncWrite failed: %v\n", err)
	}
}
