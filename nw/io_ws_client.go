package nw

import (
	"net"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gox/frm/log"
)

type WsClient struct {
	connected int32
	fd        int64
	recvSeq   int64
	sendSeq   int64
	timeout   time.Duration
	conn      *websocket.Conn
	realIP    string
	userData  any
}

func NewWsClient(addr string, timeout time.Duration) (*WsClient, error) {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	rawConn, err := conn.NetConn().(*net.TCPConn).SyscallConn()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	ch := make(chan int64, 1)
	rawConn.Control(func(fd uintptr) {
		ch <- int64(fd)
	})

	fd := <-ch

	return &WsClient{
		connected: 1,
		fd:        fd,
		recvSeq:   0,
		sendSeq:   0,
		timeout:   timeout,
		conn:      conn,
		realIP:    conn.RemoteAddr().(*net.TCPAddr).IP.String(),
		userData:  nil,
	}, nil
}

func (this_ *WsClient) IsConnected() bool {
	return atomic.LoadInt32(&this_.connected) == 1
}

func (this_ *WsClient) SockFd() int64 {
	return this_.fd
}

func (this_ *WsClient) LocalAddr() net.Addr {
	return this_.conn.LocalAddr()

}

func (this_ *WsClient) RemoteAddr() net.Addr {
	return this_.conn.RemoteAddr()
}

func (this_ *WsClient) RealRemoteIP() string {
	return this_.realIP
}

func (this_ *WsClient) Close() error {
	err := this_.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	if err != nil {
		log.Error("WsClient[%v] write control close message failed: %v", this_.realIP, err)
	}

	err = this_.conn.Close()
	atomic.StoreInt32(&this_.connected, 0)
	return err
}

func (this_ *WsClient) Write(data []byte) (int, error) {
	var err error

	if this_.timeout > 0 {
		err = this_.conn.SetWriteDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			log.Error(err)
			return -1, err
		}
	}

	err = this_.conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		return -1, err
	}

	this_.sendSeq++
	return len(data), nil
}

func (this_ *WsClient) Read() ([]byte, error) {
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

func (this_ *WsClient) GetUserData() any {
	return this_.userData
}

func (this_ *WsClient) SetUserData(userData any) {
	this_.userData = userData
}

func (this_ *WsClient) GetRecvSeq() int64 {
	return this_.recvSeq
}

func (this_ *WsClient) GetSendSeq() int64 {
	return this_.sendSeq
}
