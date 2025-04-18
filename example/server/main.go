package main

import (
	"github.com/gox/frm/log"
	"github.com/gox/frm/nw"
	"github.com/gox/frm/utils"
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
	log.Info("recvSeq: %v, %v", sess.GetRecvSeq(), *utils.Bytes2Str(data))
	_, err := sess.Write(data)
	if err != nil {
		log.Error(err)
		return false
	}
	log.Info("sendSeq: %v", sess.GetSendSeq())
	return true
}

func (this_ *EchoService) OnStarted(ios *nw.IoServer) error {
	log.Info("running ...: %v, %v", ios.TcpAddr(), ios.WsAddr())
	return nil
}

func (this_ *EchoService) OnStopped(ios *nw.IoServer) {
	log.Info("stop !!!")
}

func main() {
	ios, err := nw.NewIOServer(&nw.IOSConfig{
		IP:      "127.0.0.1",
		WsPort:  8081,
		TcpPort: 8080,
		Timeout: 300,
		Blend:   0x12345678,
	}, &EchoService{})
	if err != nil {
		log.Fatal(err)
	}

	err = ios.Run()
	if err != nil {
		log.Error(err)
	}
}
