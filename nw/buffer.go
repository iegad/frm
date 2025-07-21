package nw

import (
	"encoding/binary"
	"io"
	"sync"
)

// Buffer 缓冲区，实现了 io.Reader 和 io.Writer 接口
type Buffer struct {
	buf    []byte
	offset int
	length int // 实际数据长度
}

// NewBuffer 创建默认大小的缓冲区
func NewBuffer() *Buffer {
	return &Buffer{
		buf: make([]byte, 2048),
	}
}

// Len 返回缓冲区中数据的长度
func (this_ *Buffer) Len() int {
	return this_.length
}

// Cap 返回缓冲区的容量
func (this_ *Buffer) Cap() int {
	return len(this_.buf)
}

// Bytes 获取缓冲区中的数据（只读）
func (this_ *Buffer) Bytes() []byte {
	return this_.buf[:this_.length]
}

// Buf 获取底层缓冲区（可写，谨慎使用）
func (this_ *Buffer) Buf() []byte {
	return this_.buf
}

// Grow 确保缓冲区有足够的空间写入n字节
func (this_ *Buffer) Grow(n int) {
	if this_.length+n > len(this_.buf) {
		this_.grow(n)
	}
}

// grow 扩容缓冲区
func (this_ *Buffer) grow(n int) {
	newCap := len(this_.buf) * 2
	if newCap < this_.length+n {
		newCap = this_.length + n
	}

	newBuf := make([]byte, newCap)
	copy(newBuf, this_.buf[:this_.length])
	this_.buf = newBuf
}

// Write 实现 io.Writer 接口，向缓冲区写入数据
func (this_ *Buffer) Write(data []byte) (int, error) {
	dlen := len(data)
	if dlen == 0 {
		return 0, nil
	}

	this_.Grow(dlen)
	copy(this_.buf[this_.length:], data)
	this_.length += dlen
	return dlen, nil
}

// WriteUint32BE 写入大端序 uint32 值
func (this_ *Buffer) WriteUint32BE(v uint32) {
	this_.Grow(4)
	binary.BigEndian.PutUint32(this_.buf[this_.length:], v)
	this_.length += 4
}

// Read 实现 io.Reader 接口，从缓冲区读取数据
func (this_ *Buffer) Read(data []byte) (int, error) {
	if this_.offset >= this_.length {
		return 0, io.EOF
	}

	n := copy(data, this_.buf[this_.offset:this_.length])
	this_.offset += n
	return n, nil
}

// ReadUint32BE 读取大端序 uint32 值
func (this_ *Buffer) ReadUint32BE() (uint32, error) {
	if this_.offset+4 > this_.length {
		return 0, io.EOF
	}

	v := binary.BigEndian.Uint32(this_.buf[this_.offset:])
	this_.offset += 4
	return v, nil
}

// Reset 重置缓冲区（清空数据，重置读写位置）
func (this_ *Buffer) Reset() {
	this_.offset = 0
	this_.length = 0
}

// ResetRead 重置读取位置（保持数据不变）
func (this_ *Buffer) ResetRead() {
	this_.offset = 0
}

// String 返回缓冲区内容的字符串表示
func (this_ *Buffer) String() string {
	return string(this_.buf[:this_.length])
}

// Truncate 截断缓冲区到指定长度
func (this_ *Buffer) Truncate(n int) {
	if n < 0 || n > this_.length {
		return
	}
	this_.length = n
	if this_.offset > this_.length {
		this_.offset = this_.length
	}
}

// Next 返回下一个n字节的数据，并推进读取位置
func (this_ *Buffer) Next(n int) []byte {
	if n <= 0 {
		return nil
	}

	if this_.offset+n > this_.length {
		n = this_.length - this_.offset
	}

	data := this_.buf[this_.offset : this_.offset+n]
	this_.offset += n
	return data
}

// BufferPool 缓冲区池
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool 创建缓冲区池
func NewBufferPool() BufferPool {
	return BufferPool{
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
