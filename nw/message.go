package nw

import (
	"sync"
)

// message 消息
type message struct {
	cctx *ConnContext // 消息的发送者
	len  int
	data []byte // 发送数据
}

func (this_ *message) Init(cctx *ConnContext, data []byte) {
	this_.cctx = cctx
	this_.setData(data)
}

func (this_ *message) Data() []byte {
	return this_.data[:this_.len]
}

func (this_ *message) setData(data []byte) {
	dlen := len(this_.data)
	slen := len(data)
	if dlen < slen {
		this_.data = make([]byte, slen*2)
	}

	this_.len = slen
	copy(this_.data, data)
}

type messagePool struct {
	pool sync.Pool
}

func newMessagePool() messagePool {
	return messagePool{
		pool: sync.Pool{
			New: func() any {
				return &message{
					cctx: &ConnContext{},
				}
			},
		},
	}
}

func (this_ *messagePool) Get(cctx *ConnContext, data []byte) *message {
	msg := this_.pool.Get().(*message)
	msg.Init(cctx, data)
	return msg
}

func (this_ *messagePool) Put(msg *message) {
	if msg != nil {
		this_.pool.Put(msg)
	}
}
