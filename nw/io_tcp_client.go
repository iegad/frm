package nw

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/gox/frm/log"
)

// TcpClient TCP客户端
type TcpClient struct {
	tcpHeadBlend uint32
	fd           int64
	recvSeq      int64
	sendSeq      int64
	timeout      time.Duration
	conn         *net.TCPConn
	reader       *bufio.Reader
	userData     interface{}
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

	rawConn, err := conn.(*net.TCPConn).SyscallConn()
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
		tcpHeadBlend: blend,
		fd:           fd,
		recvSeq:      0,
		sendSeq:      0,
		timeout:      timeout,
		conn:         conn.(*net.TCPConn),
		reader:       bufio.NewReader(conn),
		userData:     nil,
	}, nil
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

func (this_ *TcpClient) Close() error {
	return this_.conn.Close()
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
	if this_.conn == nil {
		log.Fatal("Client.tcpConn is nil")
	}

	if this_.timeout > 0 {
		err := this_.conn.SetReadDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			log.Error("set conn[%v] read timeout failed: %v then will ACITVE close", this_.conn.RemoteAddr(), err)
			return nil, err
		}
	}

	hbuf := make([]byte, UINT32_SIZE)
	_, err := io.ReadAtLeast(this_.reader, hbuf, UINT32_SIZE)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			log.Error("read from conn[%v] failed: %v then will ACITVE close", this_.conn.RemoteAddr(), err)
		} else {
			log.Error("read from conn[%v] failed: %v then will PASSIVE close", this_.conn.RemoteAddr(), err)
		}
		return nil, err
	}

	buflen := binary.BigEndian.Uint32(hbuf) ^ this_.tcpHeadBlend
	if buflen == 0 || buflen > MAX_BUF_SIZE {
		log.Error("read from conn[%v] failed: %v then will ACITVE close", this_.conn.RemoteAddr(), ErrInvalidBufSize)
		return nil, ErrInvalidBufSize
	}

	rbuf := make([]byte, buflen)
	_, err = io.ReadAtLeast(this_.reader, rbuf, int(buflen))
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			log.Error("read from conn[%v] failed: %v then will ACITVE close", this_.conn.RemoteAddr(), err)
		} else {
			log.Error("read from conn[%v] failed: %v then will PASSIVE close", this_.conn.RemoteAddr(), err)
		}
		return nil, err
	}

	this_.recvSeq++
	return rbuf, nil
}

func (this_ *TcpClient) GetUserData() interface{} {
	return this_.userData
}

func (this_ *TcpClient) SetUserData(userData interface{}) {
	this_.userData = userData
}

func (this_ *TcpClient) GetRecvSeq() int64 {
	return this_.recvSeq
}

func (this_ *TcpClient) GetSendSeq() int64 {
	return this_.sendSeq
}
