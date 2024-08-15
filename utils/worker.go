package utils

import "github.com/gox/frm/log"

type WorkerHandler[T any] func(item *T)

type Worker[T any] struct {
	q       chan *T
	proc    WorkerHandler[T]
	running bool
}

func NewWorker[T any](proc WorkerHandler[T]) *Worker[T] {
	this_ := &Worker[T]{
		q:       nil,
		proc:    proc,
		running: false,
	}

	return this_
}

func (this_ *Worker[T]) Push(item *T) {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	this_.q <- item
}

func (this_ *Worker[T]) Run() {
	if this_.running {
		return
	}

	this_.q = make(chan *T, 1000)

	this_.running = true

	for this_.running {
		item, ok := <-this_.q
		if ok {
			this_.proc(item)
		}
	}
}

func (this_ *Worker[T]) Stop() {
	this_.running = false
	close(this_.q)
}
