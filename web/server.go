package web

import (
	"net"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	listener *net.TCPListener
	host     *net.TCPAddr
	router   *gin.Engine
	stMtx    sync.Mutex
}

func NewServer(host string, release bool) (*Server, error) {
	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, err
	}

	if release {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(cors.Default())

	return &Server{
		host:   addr,
		router: router,
	}, nil
}

func (this_ *Server) Router() *gin.Engine {
	return this_.router
}

func (this_ *Server) Run() error {
	var err error

	this_.listener, err = net.ListenTCP("tcp", this_.host)
	if err != nil {
		return err
	}

	return this_.router.RunListener(this_.listener)
}

func (this_ *Server) Close() {
	this_.stMtx.Lock()
	defer this_.stMtx.Unlock()

	if this_.listener != nil {
		this_.listener.Close()
		this_.listener = nil
	}
}
