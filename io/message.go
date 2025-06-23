package io

import (
	"github.com/gox/frm/utils"
	"github.com/panjf2000/gnet/v2"
)

type message struct {
	Conn gnet.Conn
	Data []byte
}

var (
	messagePool = utils.NewPool[message]()
)
