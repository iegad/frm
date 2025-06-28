package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gox/frm/log"
	"github.com/gox/frm/nw"
)

type echoHandler struct{}

func (this_ *echoHandler) OnInit(s *nw.Service) error {
	log.Debug("server is running...")
	return nil
}

func (this_ *echoHandler) OnConnected(c *nw.ConnContext) error {
	log.Debug("[%d:%s] has connected", c.Fd(), c.RemoteAddr())
	return nil
}

func (this_ *echoHandler) OnDisconnected(c *nw.ConnContext) {
	log.Debug("[%d:%s] has disconnected", c.Fd(), c.RemoteAddr())
}

func (this_ *echoHandler) OnStopped(s *nw.Service) {
	log.Debug("server has stopped...: %v", s.CurrConn())
}

func (this_ *echoHandler) OnData(c *nw.ConnContext, data []byte) error {
	return c.Write(data)
}

func main() {
	server := nw.NewService(&nw.Config{
		TcpHost:   ":9090",
		WsHost:    ":9091",
		MaxConn:   10000,
		HeadBlend: 0x01020304,
		Timeout:   60,
	}, &echoHandler{})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Debug("收到信号: %v", sig)
		server.Stop()
	}()

	server.Run()
}
