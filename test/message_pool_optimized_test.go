package test

import (
	"testing"
	"time"

	"github.com/gox/frm/nw"
	"github.com/stretchr/testify/assert"
)

func TestOptimizedMessagePool(t *testing.T) {
	// 创建配置
	config := &nw.Config{
		TcpHost:   "127.0.0.1:0",
		WsHost:    "127.0.0.1:0",
		MaxConn:   100,
		Timeout:   30,
		HeadBlend: 0x12345678,
	}

	// 创建事件处理器
	event := &MockServiceEvent{}

	// 创建服务
	service := nw.NewService(config, event)
	assert.NotNil(t, service)

	t.Run("MultipleServerPools", func(t *testing.T) {
		// 验证TCP和WebSocket服务器都有独立的消息池
		assert.NotNil(t, service.tcpSvr)
		assert.NotNil(t, service.wsSvr)

		// 创建连接上下文
		cctx := &nw.ConnContext{}
		cctx.Init(nil, service.tcpSvr)

		// 测试TCP服务器的消息池
		data1 := []byte("tcp message")
		msg1 := service.tcpSvr.getMessage(cctx, data1)
		assert.NotNil(t, msg1)
		assert.Equal(t, cctx, msg1.cctx)
		assert.Equal(t, data1, msg1.Data())

		// 测试WebSocket服务器的消息池
		cctx2 := &nw.ConnContext{}
		cctx2.Init(nil, service.wsSvr)
		data2 := []byte("ws message")
		msg2 := service.wsSvr.getMessage(cctx2, data2)
		assert.NotNil(t, msg2)
		assert.Equal(t, cctx2, msg2.cctx)
		assert.Equal(t, data2, msg2.Data())

		// 归还消息
		service.tcpSvr.putMessage(msg1)
		service.wsSvr.putMessage(msg2)

		// 验证重置
		assert.Nil(t, msg1.cctx)
		assert.Nil(t, msg2.cctx)
	})

	t.Run("MessagePoolIsolation", func(t *testing.T) {
		// 测试不同服务器的消息池是隔离的
		cctx := &nw.ConnContext{}
		cctx.Init(nil, service.tcpSvr)

		// 从TCP池获取消息
		msg1 := service.tcpSvr.getMessage(cctx, []byte("tcp1"))
		msg2 := service.tcpSvr.getMessage(cctx, []byte("tcp2"))

		// 从WS池获取消息
		cctx2 := &nw.ConnContext{}
		cctx2.Init(nil, service.wsSvr)
		msg3 := service.wsSvr.getMessage(cctx2, []byte("ws1"))

		// 归还到各自的池
		service.tcpSvr.putMessage(msg1)
		service.tcpSvr.putMessage(msg2)
		service.wsSvr.putMessage(msg3)

		// 再次获取，应该从各自的池中获取
		msg4 := service.tcpSvr.getMessage(cctx, []byte("tcp3"))
		msg5 := service.wsSvr.getMessage(cctx2, []byte("ws2"))

		assert.NotNil(t, msg4)
		assert.NotNil(t, msg5)
		assert.Equal(t, []byte("tcp3"), msg4.Data())
		assert.Equal(t, []byte("ws2"), msg5.Data())
	})

	t.Run("MessageReset", func(t *testing.T) {
		msg := &nw.Message{}
		cctx := &nw.ConnContext{}
		data := []byte("test data")

		// 初始化消息
		msg.init(cctx, data)
		assert.Equal(t, cctx, msg.cctx)
		assert.Equal(t, data, msg.Data())
		assert.Equal(t, len(data), msg.len)

		// 重置消息
		msg.Reset()
		assert.Nil(t, msg.cctx)
		assert.Equal(t, 0, msg.len)
		assert.Nil(t, msg.data)
	})

	t.Run("BaseServerMethods", func(t *testing.T) {
		// 测试baseServer的方法
		cctx := &nw.ConnContext{}
		cctx.Init(nil, service.tcpSvr)

		// 测试getMessage
		data := []byte("base server message")
		msg := service.tcpSvr.getMessage(cctx, data)
		assert.NotNil(t, msg)
		assert.Equal(t, data, msg.Data())

		// 测试putMessage
		service.tcpSvr.putMessage(msg)
		assert.Nil(t, msg.cctx)
		assert.Equal(t, 0, msg.len)
		assert.Nil(t, msg.data)
	})
}

func TestMessageDataHandling(t *testing.T) {
	msg := &nw.Message{}

	t.Run("SetDataExpansion", func(t *testing.T) {
		// 测试数据扩展
		data1 := []byte("short")
		msg.setData(data1)
		assert.Equal(t, data1, msg.Data())
		assert.Equal(t, len(data1), msg.len)

		// 设置更大的数据，应该扩展内部缓冲区
		data2 := []byte("this is a much longer message that should trigger expansion")
		msg.setData(data2)
		assert.Equal(t, data2, msg.Data())
		assert.Equal(t, len(data2), msg.len)

		// 设置更小的数据，应该重用缓冲区
		data3 := []byte("small")
		msg.setData(data3)
		assert.Equal(t, data3, msg.Data())
		assert.Equal(t, len(data3), msg.len)
	})

	t.Run("DataCopy", func(t *testing.T) {
		// 测试数据复制
		originalData := []byte("original data")
		msg.setData(originalData)
		retrievedData := msg.Data()

		// 修改原始数据不应该影响消息数据
		originalData[0] = 'X'
		assert.Equal(t, []byte("original data"), msg.Data())

		// 修改返回的数据不应该影响消息数据
		retrievedData[0] = 'Y'
		assert.Equal(t, []byte("original data"), msg.Data())
	})
}

// 性能测试
func BenchmarkOptimizedMessagePool(b *testing.B) {
	config := &nw.Config{
		TcpHost:   "127.0.0.1:0",
		WsHost:    "127.0.0.1:0",
		MaxConn:   1000,
		Timeout:   30,
		HeadBlend: 0x12345678,
	}

	event := &MockServiceEvent{}
	service := nw.NewService(config, event)

	b.Run("TCPMessagePool", func(b *testing.B) {
		cctx := &nw.ConnContext{}
		cctx.Init(nil, service.tcpSvr)
		data := []byte("benchmark tcp message")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			msg := service.tcpSvr.getMessage(cctx, data)
			service.tcpSvr.putMessage(msg)
		}
	})

	b.Run("WSMessagePool", func(b *testing.B) {
		cctx := &nw.ConnContext{}
		cctx.Init(nil, service.wsSvr)
		data := []byte("benchmark ws message")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			msg := service.wsSvr.getMessage(cctx, data)
			service.wsSvr.putMessage(msg)
		}
	})

	b.Run("ConcurrentPools", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			cctx := &nw.ConnContext{}
			data := []byte("concurrent message")

			for pb.Next() {
				// 交替使用TCP和WS池
				if cctx.server == service.tcpSvr {
					cctx.Init(nil, service.wsSvr)
					msg := service.wsSvr.getMessage(cctx, data)
					service.wsSvr.putMessage(msg)
				} else {
					cctx.Init(nil, service.tcpSvr)
					msg := service.tcpSvr.getMessage(cctx, data)
					service.tcpSvr.putMessage(msg)
				}
			}
		})
	})
} 