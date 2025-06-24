package io

import (
	"github.com/gox/frm/utils"
)

type message struct {
	Conn  *ConnContext
	Data  []byte
	Error error
}

var (
	messagePool = utils.NewPool[message]()
)
