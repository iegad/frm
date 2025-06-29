package test

import (
	"testing"
	"time"

	"github.com/gox/frm/nw"
	"github.com/stretchr/testify/assert"
)

// MockServiceEvent 模拟服务事件
type MockServiceEvent struct {
	connectedCount    int
	disconnectedCount int
	dataCount         int
	lastData          []byte
}

func (m *MockServiceEvent) OnInit(service *nw.Service) error {
	return nil
}

func (m *MockServiceEvent) OnConnected(cctx *nw.ConnContext) error {
	m.connectedCount++
	return nil
}

func (m *MockServiceEvent) OnDisconnected(cctx *nw.ConnContext) {
	m.disconnectedCount++
}

func (m *MockServiceEvent) OnStopped(service *nw.Service) {
}

func (m *MockServiceEvent) OnData(cctx *nw.ConnContext, data []byte) error {
	m.dataCount++
	m.lastData = data
	return nil
}

func TestMessagePoolOptimization(t *testing.T) {
	// 创建配置
	config := &nw.Config{
		TcpHost:   "127.0.0.1:0", // 使用0端口避免冲突
		MaxConn:   100,
		Timeout:   30,
		HeadBlend: 0x12345678,
	}

	// 创建事件处理器
	event := &MockServiceEvent{}

	// 创建服务
	service := nw.NewService(config, event)
	assert.NotNil(t, service)

	// 测试消息池功能
	t.Run("MessagePoolBasic", func(t *testing.T) {
		// 创建连接上下文
		cctx := &nw.ConnContext{}
		cctx.Init(nil, nil) // 简化测试，不传入真实连接

		// 测试获取消息
		data := []byte("test message")
		msg := service.getMessage(cctx, data)
		assert.NotNil(t, msg)
		assert.Equal(t, cctx, msg.cctx)
		assert.Equal(t, data, msg.Data())

		// 测试归还消息
		service.putMessage(msg)
		assert.Nil(t, msg.cctx) // Reset后应该为nil
		assert.Equal(t, 0, msg.len)
		assert.Nil(t, msg.data)
	})

	t.Run("MessagePoolReuse", func(t *testing.T) {
		// 测试消息对象重用
		cctx := &nw.ConnContext{}
		cctx.Init(nil, nil)

		// 获取多个消息
		msg1 := service.getMessage(cctx, []byte("message1"))
		msg2 := service.getMessage(cctx, []byte("message2"))

		// 归还消息
		service.putMessage(msg1)
		service.putMessage(msg2)

		// 再次获取，应该能重用对象
		msg3 := service.getMessage(cctx, []byte("message3"))
		assert.NotNil(t, msg3)
		assert.Equal(t, []byte("message3"), msg3.Data())
	})

	t.Run("MessageReset", func(t *testing.T) {
		msg := &nw.Message{}
		cctx := &nw.ConnContext{}
		data := []byte("test data")

		// 初始化消息
		msg.init(cctx, data)
		assert.Equal(t, cctx, msg.cctx)
		assert.Equal(t, data, msg.Data())

		// 重置消息
		msg.Reset()
		assert.Nil(t, msg.cctx)
		assert.Equal(t, 0, msg.len)
		assert.Nil(t, msg.data)
	})
}

func TestMessageDataHandling(t *testing.T) {
	msg := &nw.Message{}

	t.Run("SetData", func(t *testing.T) {
		// 测试设置数据
		data1 := []byte("hello world")
		msg.setData(data1)
		assert.Equal(t, data1, msg.Data())
		assert.Equal(t, len(data1), msg.len)

		// 测试设置更大的数据
		data2 := []byte("this is a longer message")
		msg.setData(data2)
		assert.Equal(t, data2, msg.Data())
		assert.Equal(t, len(data2), msg.len)

		// 测试设置更小的数据
		data3 := []byte("short")
		msg.setData(data3)
		assert.Equal(t, data3, msg.Data())
		assert.Equal(t, len(data3), msg.len)
	})

	t.Run("DataImmutability", func(t *testing.T) {
		// 测试数据不可变性
		originalData := []byte("original data")
		msg.setData(originalData)
		retrievedData := msg.Data()

		// 修改返回的数据不应该影响原始数据
		retrievedData[0] = 'X'
		assert.Equal(t, []byte("original data"), msg.Data())
	})
}

func TestServiceMessageFlow(t *testing.T) {
	config := &nw.Config{
		TcpHost:   "127.0.0.1:0",
		MaxConn:   10,
		Timeout:   30,
		HeadBlend: 0x12345678,
	}

	event := &MockServiceEvent{}
	service := nw.NewService(config, event)

	t.Run("MessageChannelCapacity", func(t *testing.T) {
		// 测试消息通道容量
		// 容量应该是 MaxConn * 10000
		expectedCapacity := config.MaxConn * 10000
		assert.Equal(t, expectedCapacity, cap(service.messageCh))
	})

	t.Run("ServiceInfo", func(t *testing.T) {
		// 测试服务信息
		info := service.String()
		assert.Contains(t, info, "tcp_host")
		assert.Contains(t, info, "max_conn")
		assert.Contains(t, info, "timeout")
	})

	t.Run("ConnectionCount", func(t *testing.T) {
		// 测试连接计数
		assert.Equal(t, 0, service.CurrConn())
	})
}

// 性能测试
func BenchmarkMessagePool(b *testing.B) {
	config := &nw.Config{
		TcpHost:   "127.0.0.1:0",
		MaxConn:   1000,
		Timeout:   30,
		HeadBlend: 0x12345678,
	}

	event := &MockServiceEvent{}
	service := nw.NewService(config, event)
	cctx := &nw.ConnContext{}
	cctx.Init(nil, nil)

	b.ResetTimer()
	b.Run("GetPutCycle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			msg := service.getMessage(cctx, []byte("benchmark message"))
			service.putMessage(msg)
		}
	})

	b.Run("DataAccess", func(b *testing.B) {
		msg := service.getMessage(cctx, []byte("benchmark data"))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = msg.Data()
		}
		service.putMessage(msg)
	})
} 