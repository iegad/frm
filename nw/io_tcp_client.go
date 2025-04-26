package nw

import (
	"bufio"
	"encoding/binary"
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
	n, err := write(this_.conn, data, this_.timeout, this_.tcpHeadBlend)
	if err != nil {
		return -1, err
	}

	this_.sendSeq++
	return n, err
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

	hbuf := make([]byte, TCP_HEADER_SIZE)
	_, err := io.ReadAtLeast(this_.reader, hbuf, TCP_HEADER_SIZE)
	if err != nil {
		if err == io.EOF || IsConnReset(err) {
			return nil, fmt.Errorf("TcpClient[%v] PASSIVE close: %v", this_.RemoteAddr(), err)
		}

		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), err)
	}

	buflen := binary.BigEndian.Uint32(hbuf) ^ this_.tcpHeadBlend
	if buflen == 0 || buflen > TCP_MAX_SIZE {
		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), ErrInvalidBufSize)
	}

	rbuf := make([]byte, buflen)
	_, err = io.ReadAtLeast(this_.reader, rbuf, int(buflen))
	if err != nil {
		if err == io.EOF || IsConnReset(err) {
			return nil, fmt.Errorf("TcpClient[%v] PASSIVE close: %v", this_.RemoteAddr(), err)
		}

		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), err)
	}

	this_.recvSeq++
	return rbuf, nil
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
