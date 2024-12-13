package nw

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gox/frm/log"
)

type tcpSess struct {
	blend    uint32
	fd       int64
	recvSeq  int64
	sendSeq  int64
	timeout  time.Duration
	conn     *net.TCPConn
	reader   *bufio.Reader
	userData interface{}
	realIP   string
}

func newTcpSess(conn *net.TCPConn, timeout time.Duration, blend uint32) (*tcpSess, error) {
	if blend == 0 {
		log.Fatal("blend cannot be zero")
	}

	rawConn, err := conn.SyscallConn()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	ch := make(chan int64, 1)
	rawConn.Control(func(fd uintptr) {
		ch <- int64(fd)
	})

	fd := <-ch

	realIP := strings.Split(conn.RemoteAddr().String(), ":")[0]

	return &tcpSess{
		blend:    blend,
		fd:       fd,
		recvSeq:  0,
		sendSeq:  0,
		timeout:  timeout,
		conn:     conn,
		reader:   bufio.NewReader(conn),
		userData: nil,
		realIP:   realIP,
	}, nil
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

func (this_ *tcpSess) RemoteIP() string {
	return this_.realIP
}

func (this_ *tcpSess) Close() error {
	return this_.conn.Close()
}

func (this_ *tcpSess) Write(data []byte) (int, error) {
	n, err := write(this_.conn, data, this_.timeout, this_.blend)
	if err != nil {
		return -1, err
	}

	this_.sendSeq++
	return n, err
}

func (this_ *tcpSess) Read() ([]byte, error) {
	var err error

	if this_.conn == nil {
		log.Fatal("tcpSess.tcpConn is nil")
	}

	if this_.timeout > 0 {
		err = this_.conn.SetReadDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			log.Error("set conn[%v] read timeout failed: %v then will ACITVE close", this_.conn.RemoteAddr(), err)
			return nil, err
		}
	}

	hbuf := make([]byte, UINT32_SIZE)
	_, err = io.ReadAtLeast(this_.reader, hbuf, UINT32_SIZE)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			log.Error("read from conn[%v] failed: %v then will ACITVE close", this_.conn.RemoteAddr(), err)
		} else {
			log.Error("read from conn[%v] failed: %v then will PASSIVE close", this_.conn.RemoteAddr(), err)
		}
		return nil, err
	}

	buflen := binary.BigEndian.Uint32(hbuf) ^ this_.blend
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

func (this_ *tcpSess) GetUserData() interface{} {
	return this_.userData
}

func (this_ *tcpSess) SetUserData(userData interface{}) {
	this_.userData = userData
}

func (this_ *tcpSess) GetRecvSeq() int64 {
	return this_.recvSeq
}

func (this_ *tcpSess) GetSendSeq() int64 {
	return this_.sendSeq
}
