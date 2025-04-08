package nw

import (
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gox/frm/log"
)

type wsSess struct {
	connected bool
	fd        int64
	recvSeq   int64
	sendSeq   int64
	timeout   time.Duration
	conn      *websocket.Conn
	realIP    string
	userData  any
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
		connected: false,
		fd:        fd,
		recvSeq:   0,
		sendSeq:   0,
		conn:      conn,
		timeout:   timeout,
		userData:  nil,
		realIP:    ip,
	}, nil
}

func (this_ *wsSess) IsConnected() bool {
	return this_.connected
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
	err := this_.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	if err != nil {
		log.Error("WsSess[%v] write control close message failed: %v", this_.realIP, err)
	}

	err = this_.conn.Close()
	if this_.connected {
		this_.connected = false
	}

	return err
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
			return nil, err
		}
	}

	t, data, err := this_.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	if t != websocket.BinaryMessage {
		return nil, ErrWsMsgTypeInvalid
	}

	this_.recvSeq++
	return data, nil
}

func (this_ *wsSess) GetUserData() any {
	return this_.userData
}

func (this_ *wsSess) SetUserData(userData any) {
	this_.userData = userData
}

func (this_ *wsSess) GetRecvSeq() int64 {
	return this_.recvSeq
}

func (this_ *wsSess) GetSendSeq() int64 {
	return this_.sendSeq
}
