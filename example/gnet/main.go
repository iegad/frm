package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/gox/frm/io"
	"github.com/gox/frm/log"
)

type echoHandler struct{}

func (this_ *echoHandler) OnInit(s *io.Service) error {
	log.Debug("server is running...")
	return nil
}

func (this_ *echoHandler) OnConnected(c *io.ConnContext) error {
	log.Debug("[%d:%s] has connected", c.Fd(), c.RemoteAddr())
	return nil
}

func (this_ *echoHandler) OnDisconnected(c *io.ConnContext) {
	log.Debug("[%d:%s] has disconnected", c.Fd(), c.RemoteAddr())
}

func (this_ *echoHandler) OnStopped(s *io.Service) {
	log.Debug("server has stopped...")
}

func (this_ *echoHandler) OnData(c *io.ConnContext, data []byte) error {
	return c.Write(data)
}

func main() {
	server := io.NewService(&io.Config{
		TcpHost:   ":9090",
		WsHost:    ":9091",
		MaxConn:   10000,
		HeadBlend: 0x01020304,
	}, &echoHandler{})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Debug("收到信号: %v", sig)
		server.Stop()
	}()

	server.Run(runtime.NumCPU())
}
