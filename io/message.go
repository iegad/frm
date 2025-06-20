package io

import "github.com/panjf2000/gnet/v2"

type message struct {
	Conn gnet.Conn
	Data []byte
}
