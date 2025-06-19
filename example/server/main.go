package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gox/frm/log"
	"github.com/gox/frm/nw"
)

type EchoService struct {
}

func (this_ *EchoService) OnConnected(sess nw.ISess) error {
	log.Info("[%d]%v has connected", sess.SockFd(), sess.RemoteAddr().String())
	return nil
}

func (this_ *EchoService) OnDisconnected(sess nw.ISess) {
	log.Info("[%d]%v has disconnected", sess.SockFd(), sess.RemoteAddr().String())
}

func (this_ *EchoService) OnData(sess nw.ISess, data []byte) bool {
	sess.Write(data)
	return true
}

func (this_ *EchoService) OnStarted(ios *nw.IoServer) error {
	log.Info("running ...: %v, %v", ios.TcpAddr(), ios.WsAddr())
	return nil
}

func (this_ *EchoService) OnStopped(ios *nw.IoServer) {
	log.Info("stop !!!")
}

func (this_ *EchoService) OnDecrypt(data []byte) ([]byte, error) {
	n := len(data)
	buf := make([]byte, n)
	copy(buf, data)
	return buf, nil
}

func (this_ *EchoService) OnEncrypt(data []byte) ([]byte, error) {
	return data, nil
}

func main() {
	ios, err := nw.NewIOServer(&nw.IOSConfig{
		IP:      "0.0.0.0",
		WsPort:  9091,
		TcpPort: 9090,
		Timeout: 300,
		Blend:   0x12345678,
	}, &EchoService{})
	if err != nil {
		log.Fatal(err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Debug("收到信号: %v", sig)
		ios.Stop()
	}()

	err = ios.Run()
	if err != nil {
		log.Error(err)
	}
}
