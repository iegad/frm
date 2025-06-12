package nw

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/gox/frm/log"
)

// tcpSess
//
//	TCP会话
type tcpSess struct {
	connected int32         // 是否处理连接
	blend     uint32        // 消息头混合值
	fd        int64         // 文件描述符
	recvSeq   int64         // 接收序列
	sendSeq   int64         // 发送序列
	timeout   time.Duration // 超时值
	conn      *net.TCPConn  // 连接对象
	reader    *bufio.Reader // 读缓冲区
	realIP    string        // 真实IP
	service   IService      // 服务实例
	userData  any           // 用户数据
}

// 创建新的TCP会话
//   - 该方法只会返回一种错误, 即 获取原始文件描述符失败
func newTcpSess(conn *net.TCPConn, timeout time.Duration, blend uint32, service IService) (*tcpSess, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return nil, err
	}

	ch := make(chan int64, 1)
	rawConn.Control(func(fd uintptr) {
		ch <- int64(fd)
	})

	fd := <-ch

	return &tcpSess{
		connected: 1,
		blend:     blend,
		fd:        fd,
		recvSeq:   0,
		sendSeq:   0,
		timeout:   timeout,
		conn:      conn,
		reader:    bufio.NewReader(conn),
		realIP:    conn.RemoteAddr().(*net.TCPAddr).IP.String(),
		service:   service,
		userData:  nil,
	}, nil
}

func (this_ *tcpSess) IsConnected() bool {
	return atomic.LoadInt32(&this_.connected) == 1
}

func (this_ *tcpSess) SockFd() int64 {
	return this_.fd
}

func (this_ *tcpSess) LocalAddr() net.Addr {
	return this_.conn.LocalAddr()
}

func (this_ *tcpSess) RemoteAddr() net.Addr {
	return this_.conn.RemoteAddr()
}

func (this_ *tcpSess) RealRemoteIP() string {
	return this_.realIP
}

func (this_ *tcpSess) Close() error {
	if atomic.CompareAndSwapInt32(&this_.connected, 1, 0) {
		return this_.conn.Close()
	}

	return nil
}

func (this_ *tcpSess) Write(data []byte) (int, error) {
	if len(data) > TCP_MAX_SIZE {
		log.Error("TcpSess[%v] Write data size[%d] exceeds max size[%d]", this_.RemoteAddr(), len(data), TCP_MAX_SIZE)
		return -1, ErrInvalidBufSize
	}

	if this_.timeout > 0 {
		err := this_.conn.SetWriteDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			return -1, err
		}
	}

	data, err := this_.service.OnEncrypt(data)
	if err != nil {
		log.Error("TcpSess[%v] Encrypt error: %v", this_.RemoteAddr(), err)
		return -1, err
	}

	n, err := write(this_.conn, data, this_.timeout, this_.blend)
	if err != nil {
		return -1, err
	}

	this_.sendSeq++
	return n, err
}

func (this_ *tcpSess) Read() ([]byte, error) {
	var err error

	if this_.timeout > 0 {
		err = this_.conn.SetReadDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			return nil, err
		}
	}

	hbuf := make([]byte, TCP_HEADER_SIZE)
	_, err = io.ReadAtLeast(this_.reader, hbuf, TCP_HEADER_SIZE)
	if err != nil {
		return nil, err
	}

	buflen := binary.BigEndian.Uint32(hbuf) ^ this_.blend
	if buflen == 0 || buflen > TCP_MAX_SIZE {
		return nil, ErrInvalidBufSize
	}

	rbuf := make([]byte, buflen)
	_, err = io.ReadAtLeast(this_.reader, rbuf, int(buflen))
	if err != nil {
		return nil, err
	}

	rbuf, err = this_.service.OnDecrypt(rbuf)
	if err != nil {
		log.Error("TcpSess[%v] Decrypt error: %v", this_.RemoteAddr(), err)
		return nil, err
	}

	this_.recvSeq++
	return rbuf, nil
}

func (this_ *tcpSess) GetUserData() any {
	return this_.userData
}

func (this_ *tcpSess) SetUserData(userData any) {
	this_.userData = userData
}

func (this_ *tcpSess) GetRecvSeq() int64 {
	return this_.recvSeq
}

func (this_ *tcpSess) GetSendSeq() int64 {
	return this_.sendSeq
}
