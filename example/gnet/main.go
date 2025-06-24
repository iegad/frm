package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/gox/frm/io"
	"github.com/gox/frm/log"
	"github.com/panjf2000/gnet/v2"
)

type echoHandler struct{}

func (this_ *echoHandler) OnInit(s *io.Server) error {
	log.Debug("server is running...")
	return nil
}

func (this_ *echoHandler) OnConnected(c gnet.Conn) error {
	log.Debug("[%d:%s] has connected", c.Fd(), c.RemoteAddr())
	return nil
}

func (this_ *echoHandler) OnDisconnected(c gnet.Conn) {
	log.Debug("[%d:%s] has disconnected", c.Fd(), c.RemoteAddr())
}

func (this_ *echoHandler) OnStopped(s *io.Server) {
	log.Debug("server has stopped...")
}

func (this_ *echoHandler) OnData(c gnet.Conn, data []byte) error {
	return c.AsyncWrite(data, nil)
}

func main() {
	server := io.NewServer(&io.Config{
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
