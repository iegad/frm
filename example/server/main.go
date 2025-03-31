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

func (this_ *EchoService) OnDisconnect(sess nw.ISess) {
	log.Info("[%d]%v has disconnected", sess.SockFd(), sess.RemoteAddr().String())
}

func (this_ *EchoService) OnData(sess nw.ISess, data []byte) bool {
	log.Info("recvSeq: %v, %v", sess.GetRecvSeq(), utils.Bytes2Str(data))
	_, err := sess.Write(data)
	if err != nil {
		log.Error(err)
		return false
	}
	log.Info("sendSeq: %v", sess.GetSendSeq())
	return true
}

func (this_ *EchoService) OnStart(ios *nw.IOServer) error {
	log.Info("running ...: %v, %v", ios.TcpAddr(), ios.WsAddr())
	return nil
}

func (this_ *EchoService) OnStop(ios *nw.IOServer) {
	log.Info("stop !!!")
}

func main() {
	ios, err := nw.NewIOServer(&nw.IOSConfig{
		IP:      "",
		WsPort:  8081,
		TcpPort: 8080,
		Timeout: 300,
		Blend:   0x12345678,
	}, &EchoService{})
	if err != nil {
		log.Fatal(err)
	}

	ios.Run()
}
