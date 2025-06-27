package nw

import (
	"encoding/binary"
	"sync"
)

// Buffer 缓冲区
type Buffer struct {
	buf    []byte
	offset int
}

// NewBuffer 创建缓冲区
func NewBuffer() *Buffer {
	return &Buffer{
		buf: make([]byte, 1024),
	}
}

func NewBufferWithData(buf []byte) *Buffer {
	return &Buffer{
		buf:    buf,
		offset: len(buf),
	}
}

// Write 向缓冲区写入数据
func (this_ *Buffer) Write(data []byte) (int, error) {
	blen := len(this_.buf)
	dlen := len(data)

	if blen-this_.offset < dlen {
		buf := make([]byte, this_.offset+dlen*2)
		copy(buf, this_.buf[:this_.offset])
		this_.buf = buf
	}

	copy(this_.buf[this_.offset:], data)
	this_.offset += dlen
	return dlen, nil
}

// WriteUint32 向缓冲区写在大端序 uint32 值
func (this_ *Buffer) WriteUint32(v uint32) {
	need := this_.offset + 4
	if len(this_.buf) < need {
		newCap := len(this_.buf) * 2
		if newCap < need {
			newCap = need
		}
		buf := make([]byte, newCap)
		copy(buf, this_.buf[:this_.offset])
		this_.buf = buf
	}

	binary.BigEndian.PutUint32(this_.buf[this_.offset:], v)
	this_.offset += 4
}

// Bytes 获取缓冲区中的数据
func (this_ *Buffer) Bytes() []byte {
	return this_.buf[:this_.offset]
}

// Reset 重置缓冲区
func (this_ *Buffer) Reset() {
	this_.offset = 0
}

// BufferPool 缓冲区池
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool 创建缓冲区池
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() any {
				return NewBuffer()
			},
		},
	}
}

// Get 获取 Buffer
func (this_ *BufferPool) Get() *Buffer {
	return this_.pool.Get().(*Buffer)
}

// Put 归还 Buffer
func (this_ *BufferPool) Put(buf *Buffer) {
	if buf != nil {
		buf.Reset()
		this_.pool.Put(buf)
	}
}
