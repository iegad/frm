package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/gox/frm/io"
	"github.com/gox/frm/log"
)

func main() {
	server := io.NewServer(&io.Config{
		TcpHost:   ":9090",
		WsHost:    ":9091",
		MaxConn:   10000,
		HeadBlend: 0x01020304,
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Debug("收到信号: %v", sig)
		server.Stop()
	}()

	server.Run(runtime.NumCPU())
}
