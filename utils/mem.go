package utils

// TODO: 对与 bzero, memcpy 需要改为汇编实现

import (
	"sync"
	"unsafe"
)

func Bzero[T any](obj *T) {
	data := unsafe.Slice((*byte)(unsafe.Pointer(obj)), unsafe.Sizeof(*obj))
	for i := 0; i < len(data); i++ {
		data[i] = 0
	}
}

func Memcpy[T any](dst, src *T) {
	d := unsafe.Slice((*byte)(unsafe.Pointer(dst)), unsafe.Sizeof(*dst))
	s := unsafe.Slice((*byte)(unsafe.Pointer(src)), unsafe.Sizeof(*src))
	copy(d, s)
}

func Memcmp[T any](a, b *T) bool {
	pa := *(*[]byte)(unsafe.Pointer(a))
	pb := *(*[]byte)(unsafe.Pointer(b))

	if len(pa) != len(pb) {
		return false
	}

	return *Bytes2Str(pa) == *Bytes2Str(pb)
}

func Clone[T any](obj *T) T {
	res := new(T)
	Memcpy(res, obj)
	return *res
}

type Pool[T any] struct {
	pool sync.Pool
}

func NewPool[T any]() *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return new(T)
			},
		},
	}
}

func (this_ *Pool[T]) Get() *T {
	obj := this_.pool.Get().(*T)
	return obj
}

func (this_ *Pool[T]) Put(v *T) {
	if v != nil {
		Bzero(v)
		this_.pool.Put(v)
	}
}
