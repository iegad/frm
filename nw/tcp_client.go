package nw

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/gox/frm/log"
)

var (
	errDataTooLong = errors.New("data too long") // 数据过长错误
)

// TcpClient TCP客户端
type TcpClient struct {
	connected    int32         // 连接状态
	tcpHeadBlend uint32        // tcp 消息头混合值
	fd           int64         // 原始文件描述符
	timeout      time.Duration // 读超时
	conn         *net.TCPConn  // 连接对象
	hbuf         []byte        // tcp header buffer
	dbuf         []byte        // tcp data buffer
	userData     any           // 用户数据
}

// NewTcpClient 创建一个新的 TCP 客户端
//   - host: 服务器地址
//   - timeout: 连接超时时间
//   - blend: tcp 消息头混合值，不能为0
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
		timeout:      timeout,
		conn:         tcpConn,
		hbuf:         make([]byte, TCP_HEADER_SIZE),
		dbuf:         make([]byte, MESSAGE_MAX_SIZE),
		userData:     nil,
	}, nil
}

// IsConnected 检查客户端是否连接
func (this_ *TcpClient) IsConnected() bool {
	return atomic.LoadInt32(&this_.connected) == 1
}

// Fd 获取客户端的文件描述符
func (this_ *TcpClient) Fd() int64 {
	return this_.fd
}

// // LocalAddr 获取本地地址
func (this_ *TcpClient) LocalAddr() net.Addr {
	return this_.conn.LocalAddr()
}

// RemoteAddr 获取远程地址
func (this_ *TcpClient) RemoteAddr() net.Addr {
	return this_.conn.RemoteAddr()
}

// Close 关闭客户端连接
func (this_ *TcpClient) Close() error {
	if atomic.CompareAndSwapInt32(&this_.connected, 1, 0) {
		return this_.conn.Close()
	}

	return nil
}

// Write 向服务器发送数据
func (this_ *TcpClient) Write(data []byte) (int, error) {
	var err error

	if len(data) > int(MESSAGE_MAX_SIZE) {
		return -1, fmt.Errorf("TcpClient[%v] write data size[%d] exceeds max size[%d]", this_.RemoteAddr(), len(data), MESSAGE_MAX_SIZE)
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

	return n - TCP_HEADER_SIZE, err
}

// Read 从服务器读取数据
func (this_ *TcpClient) Read() ([]byte, error) {
	// Step 1, 设置读取超时
	if this_.timeout > 0 {
		err := this_.conn.SetReadDeadline(time.Now().Add(this_.timeout))
		if err != nil {
			if err == io.EOF || IsConnReset(err) {
				return nil, fmt.Errorf("TcpClient[%v] PASSIVE close: %v", this_.RemoteAddr(), err)
			}

			return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), err)
		}
	}

	// 读取消息头
	_, err := io.ReadAtLeast(this_.conn, this_.hbuf, TCP_HEADER_SIZE)
	if err != nil {
		if err == io.EOF || IsConnReset(err) {
			return nil, fmt.Errorf("TcpClient[%v] PASSIVE close: %v", this_.RemoteAddr(), err)
		}

		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), err)
	}

	buflen := binary.BigEndian.Uint32(this_.hbuf) ^ this_.tcpHeadBlend
	if buflen == 0 || buflen > MESSAGE_MAX_SIZE {
		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: invalid data size", this_.RemoteAddr())
	}

	// Step 2, 读取数据
	_, err = io.ReadAtLeast(this_.conn, this_.dbuf, int(buflen))
	if err != nil {
		if err == io.EOF || IsConnReset(err) {
			return nil, fmt.Errorf("TcpClient[%v] PASSIVE close: %v", this_.RemoteAddr(), err)
		}

		return nil, fmt.Errorf("TcpClient[%v] ACTIVE close: %v", this_.RemoteAddr(), err)
	}

	data := make([]byte, buflen)
	copy(data, this_.dbuf[:buflen])
	return data, nil
}

// GetUserData 获取用户自定义数据
func (this_ *TcpClient) UserData() any {
	return this_.userData
}

// SetUserData 设置用户自定义数据
func (this_ *TcpClient) SetUserData(userData any) {
	this_.userData = userData
}

// write 向连接写入数据
func write(conn net.Conn, data []byte, timeout time.Duration, blend uint32) (int, error) {
	if timeout > 0 {
		err := conn.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			return -1, err
		}
	}

	dlen := len(data)

	if dlen > int(MESSAGE_MAX_SIZE) {
		return -1, errDataTooLong
	}

	wbuf := make([]byte, dlen+TCP_HEADER_SIZE)
	binary.BigEndian.PutUint32(wbuf[:TCP_HEADER_SIZE], uint32(dlen)^blend)
	copy(wbuf[TCP_HEADER_SIZE:], data)
	return conn.Write(wbuf)
}
