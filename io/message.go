package io

import (
	"github.com/gox/frm/utils"
)

type Message struct {
	Conn *ConnContext
	Data []byte
}

var (
	messagePool = utils.NewPool[Message]()
)
