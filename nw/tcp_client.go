package nw

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type AsyncTCPClient struct {
	conn      *net.TCPConn
	writeCh   chan []byte
	readCh    chan []byte
	closeCh   chan struct{}
	wg        sync.WaitGroup
	timeout   time.Duration
	closeOnce sync.Once
}

func NewAsyncTCPClient(addr string, timeout time.Duration) (*AsyncTCPClient, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	tcpConn := conn.(*net.TCPConn)

	client := &AsyncTCPClient{
		conn:    tcpConn,
		writeCh: make(chan []byte, 1024),
		readCh:  make(chan []byte, 1024),
		closeCh: make(chan struct{}),
		timeout: timeout,
	}
	client.wg.Add(2)
	go client.readLoop()
	go client.writeLoop()
	return client, nil
}

// Async write: 投递消息到写队列
func (c *AsyncTCPClient) Write(msg []byte) error {
	if len(msg) > 0xFFFFFFF { // 256MB 限制
		return errors.New("message too large")
	}
	buf := make([]byte, 4+len(msg))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(msg)))
	copy(buf[4:], msg)
	select {
	case c.writeCh <- buf:
		return nil
	case <-c.closeCh:
		return errors.New("client closed")
	}
}

// Async read: 业务层从 readCh 取消息
func (c *AsyncTCPClient) Read() ([]byte, error) {
	select {
	case msg, ok := <-c.readCh:
		if !ok {
			return nil, io.EOF
		}
		return msg, nil
	case <-c.closeCh:
		return nil, io.EOF
	}
}

// 关闭客户端
func (c *AsyncTCPClient) Close() error {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		c.conn.Close()
	})
	c.wg.Wait()
	close(c.readCh)
	return nil
}

// 后台读协程
func (c *AsyncTCPClient) readLoop() {
	defer c.wg.Done()
	header := make([]byte, 4)
	for {
		if c.timeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.timeout))
		}
		_, err := io.ReadFull(c.conn, header)
		if err != nil {
			break
		}
		bodyLen := binary.BigEndian.Uint32(header)
		if bodyLen == 0 || bodyLen > 0xFFFFFFF {
			break
		}
		body := make([]byte, bodyLen)
		_, err = io.ReadFull(c.conn, body)
		if err != nil {
			break
		}
		select {
		case c.readCh <- body:
		case <-c.closeCh:
			return
		}
	}
}

// 后台写协程
func (c *AsyncTCPClient) writeLoop() {
	defer c.wg.Done()
	for {
		select {
		case buf, ok := <-c.writeCh:
			if !ok {
				return
			}
			if c.timeout > 0 {
				c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
			}
			_, err := c.conn.Write(buf)
			if err != nil {
				return
			}
		case <-c.closeCh:
			return
		}
	}
}
