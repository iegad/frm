package utils

import "sync"

type SafeMap[K comparable, V any] struct {
	m   map[K]*V
	mtx sync.Mutex
}

func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		m:   make(map[K]*V),
		mtx: sync.Mutex{},
	}
}

func (this_ *SafeMap[K, V]) Set(key K, value *V) {
	this_.mtx.Lock()
	this_.m[key] = value
	this_.mtx.Unlock()
}

func (this_ *SafeMap[K, V]) Remove(key K) {
	this_.mtx.Lock()
	delete(this_.m, key)
	this_.mtx.Unlock()
}

func (this_ *SafeMap[K, V]) Get(key K) *V {
	this_.mtx.Lock()
	defer this_.mtx.Unlock()
	return this_.m[key]
}

func (this_ *SafeMap[K, V]) Clear() {
	this_.mtx.Lock()
	this_.m = map[K]*V{}
	this_.mtx.Unlock()
}

func (this_ *SafeMap[K, V]) Range(handler func(key K, v *V) bool) {
	if handler == nil {
		return
	}

	this_.mtx.Lock()
	defer this_.mtx.Unlock()

	for k, v := range this_.m {
		if !handler(k, v) {
			break
		}
	}
}
