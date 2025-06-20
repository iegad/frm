package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gox/frm/io"
	"github.com/gox/frm/log"
)

func main() {
	server, err := io.NewServer(&io.Config{
		TcpAddress: ":9090",
		MaxConn:    10000,
		HeadBlend:  0x01020304,
	})
	if err != nil {
		log.Fatal(err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Debug("收到信号: %v", sig)
		server.Stop()
	}()

	server.Run()
}
