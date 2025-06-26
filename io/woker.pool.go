package io

import (
	"sync"

	"github.com/gox/frm/utils"
)

type WorkerPool struct {
	count int
	wkrs  []*utils.Worker[Message]
	wg    *sync.WaitGroup
}

func NewWorkerPool(n int, handler utils.WorkerHandler[Message], wg *sync.WaitGroup) *WorkerPool {
	wkrs := []*utils.Worker[Message]{}

	for i := 0; i < n; i++ {
		wkrs = append(wkrs, utils.NewWorker(handler))
	}

	return &WorkerPool{
		count: n,
		wkrs:  wkrs,
		wg:    wg,
	}
}

func (this_ *WorkerPool) Run() {
	this_.wg.Add(this_.count)
	for _, wkr := range this_.wkrs {
		go func(wg *sync.WaitGroup) {
			wkr.Run()
			wg.Done()
		}(this_.wg)
	}
}

func (this_ *WorkerPool) Stop() {
	for _, wkr := range this_.wkrs {
		wkr.Stop()
	}
}

func (this_ *WorkerPool) Push(flag int, msg *Message) {
	this_.wkrs[flag%this_.count].Push(msg)
}
