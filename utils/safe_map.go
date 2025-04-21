package utils

import (
	"sync"
)

type SafeMap[K comparable, V any] struct {
	m   map[K]V
	mtx sync.RWMutex
}

func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		m: map[K]V{},
	}
}

func (this_ *SafeMap[K, V]) Set(key K, value V) {
	this_.mtx.Lock()
	this_.m[key] = value
	this_.mtx.Unlock()
}

func (this_ *SafeMap[K, V]) Remove(key K) {
	this_.mtx.Lock()
	delete(this_.m, key)
	this_.mtx.Unlock()
}

func (this_ *SafeMap[K, V]) Has(key K) bool {
	this_.mtx.RLock()
	_, ok := this_.m[key]
	this_.mtx.RUnlock()
	return ok
}

func (this_ *SafeMap[K, V]) Get(key K) V {
	this_.mtx.RLock()
	res := this_.m[key]
	this_.mtx.RUnlock()
	return res
}

func (this_ *SafeMap[K, V]) Clear() {
	this_.mtx.Lock()
	this_.m = map[K]V{}
	this_.mtx.Unlock()
}

// 存在返回 false, 否则返回 true
func (this_ *SafeMap[K, V]) SetNx(key K, value V) bool {
	res := false
	this_.mtx.Lock()
	_, ok := this_.m[key]
	if !ok {
		this_.m[key] = value
		res = true
	}
	this_.mtx.Unlock()
	return res
}

func (this_ *SafeMap[K, V]) Range(handler func(key K, v V) bool) {
	if handler != nil {
		this_.mtx.RLock()
		for k, v := range this_.m {
			if !handler(k, v) {
				break
			}
		}
		this_.mtx.RUnlock()
	}
}

func (this_ *SafeMap[K, V]) Count() int {
	count := 0
	this_.mtx.RLock()
	count = len(this_.m)
	this_.mtx.RUnlock()
	return count
}
