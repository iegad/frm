package nw

import (
	"sync"
)

// message 消息
type message struct {
	cctx    *ConnContext // 消息的发送者
	len     int          // 数据长度
	buf     []byte       // 发送数据
	msgPool *messagePool // 所属的消息池
}

// release 释放消息对象到对象池中
func (this_ *message) release() {
	this_.msgPool.Put(this_)
}

// Data 获取消息数据
func (this_ *message) data() []byte {
	return this_.buf[:this_.len]
}

// messagePool 消息池
//
// messagePool 用于复用 message 对象, 减少内存分配和垃圾回收的开销
// 使用 sync.Pool 来实现对象池, 提高性能
// 注意: message 对象在使用完毕后需要调用 Put 方法归还对象到池中
type messagePool struct {
	pool sync.Pool
}

// newMessagePool 创建一个新的 messagePool
func newMessagePool() messagePool {
	return messagePool{
		pool: sync.Pool{
			New: func() any {
				return &message{}
			},
		},
	}
}

// Get 从池中获取一个 message 对象
// cctx: 消息的发送者上下文
// data: 消息数据
// 返回一个 message 对象, 该对象已经初始化, 可以直接使用
// 注意: 使用完毕后需要调用 Put 方法归还对象到池中
func (this_ *messagePool) Get(cctx *ConnContext, data []byte) *message {
	msg := this_.pool.Get().(*message)
	msg.cctx = cctx
	msg.len = len(data)
	if msg.buf == nil || cap(msg.buf) < msg.len {
		msg.buf = make([]byte, msg.len)
	}

	copy(msg.buf, data)
	return msg
}

func (this_ *messagePool) GetRef(cctx *ConnContext, data []byte) *message {
	msg := this_.pool.Get().(*message)
	msg.cctx = cctx
	msg.len = len(data)
	msg.buf = data
	return msg
}

// Put 归还 message 对象到池中
// msg: 要归还的 message 对象
// 注意: 归还的对象不能为 nil, 否则会导致 panic
// 归还后, 对象会被重置, 下次获取时会重新初始化
// 如果 msg 为 nil, 则什么都不做
func (this_ *messagePool) Put(msg *message) {
	if msg != nil {
		this_.pool.Put(msg)
	}
}
