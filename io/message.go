package io

import (
	"github.com/gox/frm/utils"
)

type Message struct {
	Context *ConnContext
	Data    []byte
}

var messagePool = utils.NewPool[Message]()
