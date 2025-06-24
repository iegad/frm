package io

import (
	"github.com/gox/frm/utils"
)

type message struct {
	Conn  *Conn
	Data  []byte
	Error error
}

var (
	messagePool = utils.NewPool[message]()
)
