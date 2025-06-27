package nw

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/gox/frm/log"
)

// TcpClient TCP客户端
type TcpClient struct {
	connected    int32         // 连接状态
	tcpHeadBlend uint32        // tcp 消息头混合值
	fd           int64         // 原始文件描述符
	recvSeq      int64         // 接收序列
	sendSeq      int64         // 发送序列
	timeout      time.Duration // 读超时
	conn         *net.TCPConn  // 连接对象
	reader       *bufio.Reader // 读缓冲区
	realIP       string        // 真实IP
	hbuf         []byte        // tcp header buffer
	dbuf         []byte        // tcp data buffer
	userData     any           // 用户数据
}

func NewTcpClient(host string, timeout time.Duration, blend uint32) (*TcpClient, error) {
	if blend == 0 {
		log.Fatal("blend cannot be zero")
	}

	var (
		conn net.Conn
		err  error
	)

	if timeout > 0 {
		conn, err = net.DialTimeout("tcp", host, timeout)
	} else {
		conn, err = net.Dial("tcp", host)
	}

	if err != nil {
		return nil, err
	}

	tcpConn := conn.(*net.TCPConn)
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	ch := make(chan int64, 1)
	rawConn.Control(func(fd uintptr) {
		ch <- int64(fd)
	})

	fd := <-ch

	return &TcpClient{
		connected:    1,
		tcpHeadBlend: blend,
		fd:           fd,
		recvSeq:      0,
		sendSeq:      0,
		timeout:      timeout,
		conn:         tcpConn,
		reader:       bufio.NewReader(tcpConn),
		realIP:       tcpConn.RemoteAddr().(*net.TCPAddr).IP.String(),
		hbuf:         make([]byte, TCP_HEADER_SIZE),
		dbuf:         make([]byte, TCP_MAX_SIZE),
		userData:     nil,
	}, nil
}

func (this_ *TcpClient) IsConnected() bool {
	return atomic.LoadInt32(&this_.connected) == 1
}

func (this_ *TcpClient) SockFd() int64 {
	return this_.fd
}

func (this_ *TcpClient) LocalAddr() net.Addr {
	return this_.conn.LocalAddr()
}

func (this_ *TcpClient) RemoteAddr() net.Addr {
	return this_.conn.RemoteAddr()
}

func (this_ *TcpClient) RealRemoteIP() string {
	return this_.realIP
}

func (this_ *TcpClient) Close() error {
	if atomic.CompareAndSwapInt32(&this_.connected, 1, 0) {
		return this_.conn.Close()
	}

	return nil
}

func (this_ *TcpClient) Write(data []byte) (int, error) {
	var err error

	if len(data) > int(TCP_MAX_SIZE) {
		return -1, fmt.Errorf("TcpClient[%v] write data size[%d] exceeds max size[%d]", this_.RemoteAddr(), len(data), TCP_MAX_SIZE)
	}

	if !this_.IsConnected() {
		return -1, fmt.Errorf("TcpClient[%v] is not connected", this_.RemoteAddr())
	}

	if this_.timeout > 0 {
		err = this_.conn.SetWriteDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			return -1, fmt.Errorf("TcpClient[%v] write deadline error: %v", this_.RemoteAddr(), err)
		}
	}

	n, err := write(this_.conn, data, this_.timeout, this_.tcpHeadBlend)
	if err != nil {
		return -1, err
	}

	this_.sendSeq++
	return n - TCP_HEADER_SIZE, err
}

func (this_ *TcpClient) Read() ([]byte, error) {
	if this_.timeout > 0 {
		err := this_.conn.SetReadDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			if err == io.EOF || IsConnReset(err) {
				return nil, fmt.Errorf("TcpClient[%v] PASSIVE close: %v", this_.RemoteAddr(), err)
			}

			return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), err)
		}
	}

	_, err := io.ReadAtLeast(this_.reader, this_.hbuf, TCP_HEADER_SIZE)
	if err != nil {
		if err == io.EOF || IsConnReset(err) {
			return nil, fmt.Errorf("TcpClient[%v] PASSIVE close: %v", this_.RemoteAddr(), err)
		}

		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), err)
	}

	buflen := binary.BigEndian.Uint32(this_.hbuf) ^ this_.tcpHeadBlend
	if buflen == 0 || buflen > TCP_MAX_SIZE {
		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: invalid data size", this_.RemoteAddr())
	}

	_, err = io.ReadAtLeast(this_.reader, this_.dbuf, int(buflen))
	if err != nil {
		if err == io.EOF || IsConnReset(err) {
			return nil, fmt.Errorf("TcpClient[%v] PASSIVE close: %v", this_.RemoteAddr(), err)
		}

		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), err)
	}

	this_.recvSeq++
	return this_.dbuf[:buflen], nil
}

func (this_ *TcpClient) GetUserData() any {
	return this_.userData
}

func (this_ *TcpClient) SetUserData(userData any) {
	this_.userData = userData
}

func (this_ *TcpClient) GetRecvSeq() int64 {
	return this_.recvSeq
}

func (this_ *TcpClient) GetSendSeq() int64 {
	return this_.sendSeq
}

func write(conn net.Conn, data []byte, timeout time.Duration, blend uint32) (int, error) {
	if timeout > 0 {
		err := conn.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			return -1, err
		}
	}

	dlen := len(data)

	if dlen > int(TCP_MAX_SIZE) {
		return -1, errors.New("data too long")
	}

	wbuf := make([]byte, dlen+TCP_HEADER_SIZE)
	binary.BigEndian.PutUint32(wbuf[:TCP_HEADER_SIZE], uint32(dlen)^blend)
	copy(wbuf[TCP_HEADER_SIZE:], data)
	return conn.Write(wbuf)
}
