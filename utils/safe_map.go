package utils

import (
	"sync"
)

type safeMap[K comparable, V any] struct {
	m   map[K]V
	mtx sync.Mutex
}

func NewSafeMap[K comparable, V any]() *safeMap[K, V] {
	return &safeMap[K, V]{
		m:   make(map[K]V),
		mtx: sync.Mutex{},
	}
}

func (this_ *safeMap[K, V]) Set(key K, value V) {
	this_.mtx.Lock()
	this_.m[key] = value
	this_.mtx.Unlock()
}

func (this_ *safeMap[K, V]) Remove(key K) {
	this_.mtx.Lock()
	delete(this_.m, key)
	this_.mtx.Unlock()
}

func (this_ *safeMap[K, V]) Has(key K) bool {
	this_.mtx.Lock()
	_, ok := this_.m[key]
	this_.mtx.Unlock()
	return ok
}

func (this_ *safeMap[K, V]) Get(key K) V {
	this_.mtx.Lock()
	res := this_.m[key]
	this_.mtx.Unlock()
	return res
}

func (this_ *safeMap[K, V]) Clear() {
	this_.mtx.Lock()
	this_.m = map[K]V{}
	this_.mtx.Unlock()
}

func (this_ *safeMap[K, V]) Range(handler func(key K, v V) bool) {
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
