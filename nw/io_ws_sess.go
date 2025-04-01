package nw

import (
	"errors"
	"net"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gox/frm/log"
)

type wsSess struct {
	fd       int64
	recvSeq  int64
	sendSeq  int64
	conn     *websocket.Conn
	timeout  time.Duration
	realIP   string
	userData interface{}
}

func newWsSess(conn *websocket.Conn, timeout time.Duration, realIp ...string) (*wsSess, error) {
	rawConn, err := conn.NetConn().(*net.TCPConn).SyscallConn()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	ch := make(chan int64, 1)
	rawConn.Control(func(fd uintptr) {
		ch <- int64(fd)
	})

	ip := ""
	if len(realIp) == 1 && len(realIp[0]) > 0 {
		ip = realIp[0]
	}

	fd := <-ch

	return &wsSess{
		fd:       fd,
		recvSeq:  0,
		sendSeq:  0,
		conn:     conn,
		timeout:  timeout,
		userData: nil,
		realIP:   ip,
	}, nil
}

func (this_ *wsSess) SockFd() int64 {
	return this_.fd
}

func (this_ *wsSess) LocalAddr() net.Addr {
	return this_.conn.LocalAddr()
}

func (this_ *wsSess) RemoteAddr() net.Addr {
	return this_.conn.RemoteAddr()
}

func (this_ *wsSess) RealRemoteIP() string {
	return this_.realIP
}

func (this_ *wsSess) Close() error {
	return this_.conn.Close()
}

func (this_ *wsSess) Write(data []byte) (int, error) {
	this_.sendSeq++
	err := this_.conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		return -1, err
	}

	return len(data), nil
}

func (this_ *wsSess) Read() ([]byte, error) {
	if this_.timeout > 0 {
		err := this_.conn.SetReadDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			log.Error("set conn[%v] read timeout failed: %v then will ACITVE close", this_.conn.RemoteAddr(), err)
			return nil, err
		}
	}

	t, data, err := this_.conn.ReadMessage()
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			log.Error("read from conn[%v] failed: %v then will ACITVE close", this_.conn.RemoteAddr(), err)
		} else {
			log.Error("read from conn[%v] failed: %v then will PASSIVE close", this_.conn.RemoteAddr(), err)
		}
		return nil, err
	}

	if t != websocket.BinaryMessage {
		log.Error("read from conn[%v] failed: %v then will ACITVE close", this_.conn.RemoteAddr(), ErrWsMsgTypeInvalid)
		return nil, ErrWsMsgTypeInvalid
	}

	this_.recvSeq++
	return data, nil
}

func (this_ *wsSess) GetUserData() interface{} {
	return this_.userData
}

func (this_ *wsSess) SetUserData(userData interface{}) {
	this_.userData = userData
}

func (this_ *wsSess) GetRecvSeq() int64 {
	return this_.recvSeq
}

func (this_ *wsSess) GetSendSeq() int64 {
	return this_.sendSeq
}
