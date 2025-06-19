package nw

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gox/frm/log"
)

type wsSess struct {
	connected int32
	fd        int64
	recvSeq   int64
	sendSeq   int64
	timeout   time.Duration
	conn      *websocket.Conn
	realIP    string
	ios       *IoServer
	userData  any
	wch       chan []byte
	wg        sync.WaitGroup
}

func (this_ *wsSess) init(conn *websocket.Conn, timeout time.Duration, realIp string, ios *IoServer) error {
	rawConn, err := conn.NetConn().(*net.TCPConn).SyscallConn()
	if err != nil {
		log.Error(err)
		return err
	}

	ch := make(chan int64, 1)
	rawConn.Control(func(fd uintptr) {
		ch <- int64(fd)
	})

	fd := <-ch

	this_.connected = 1
	this_.fd = fd
	this_.recvSeq = 0
	this_.sendSeq = 0
	this_.conn = conn
	this_.timeout = timeout
	this_.realIP = realIp
	this_.ios = ios

	if this_.userData != nil {
		this_.userData = nil
	}

	this_.wch = make(chan []byte, MAX_CHAN_SIZE)

	this_.wg.Add(1)
	go this_.writeProc()

	return nil
}

func (this_ *wsSess) IsConnected() bool {
	return atomic.LoadInt32(&this_.connected) == 1
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
	if atomic.CompareAndSwapInt32(&this_.connected, 1, 0) {
		err := this_.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		if err != nil {
			log.Warn("WsSess[%v] write control close message failed: %v", this_.realIP, err)
		}

		err = this_.conn.Close()
		close(this_.wch)
		this_.wch = nil
		this_.wg.Wait()
		return err
	}

	return nil
}

func (this_ *wsSess) Write(data []byte) {
	this_.wch <- data
}

func (this_ *wsSess) write(data []byte) (int, error) {
	data, err := this_.OnEncrypt(data)
	if err != nil {
		log.Error("WsSess[%v] OnEncrypt failed: %v", this_.realIP, err)
		return -1, err
	}

	this_.sendSeq++

	err = this_.conn.WriteMessage(websocket.BinaryMessage, data)
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

	data, err = this_.OnDecrypt(data)
	if err != nil {
		log.Error("WsSess[%v] OnDecrypt failed: %v", this_.realIP, err)
		return nil, err
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

func (this_ *wsSess) OnEncrypt(data []byte) ([]byte, error) {
	return this_.ios.service.OnEncrypt(data)
}

func (this_ *wsSess) OnDecrypt(data []byte) ([]byte, error) {
	return this_.ios.service.OnDecrypt(data)
}

func (this_ *wsSess) writeProc() {
	for data := range this_.wch {
		_, err := this_.write(data)
		if err != nil {
			log.Error("[%v]发送消息失败: %v", this_.RealRemoteIP(), err)
		}
	}

	this_.wg.Done()
}
