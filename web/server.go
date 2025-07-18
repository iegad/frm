package web

import (
	"net"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	listener *net.TCPListener
	host     *net.TCPAddr
	router   *gin.Engine
}

// 创建web服务
func NewServer(host string, release, allowCors bool) (*Server, error) {
	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, err
	}

	if release {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	if allowCors {
		// Gin示例
		router.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Token", "X-Code", "X-Staff-ID"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * 60 * 60,
		}))
	}

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

func (this_ *Server) Stop() {
	this_.listener.Close()
}
