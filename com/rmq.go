package com

import (
	"fmt"
	"sync"

	"github.com/gox/frm/log"
	"github.com/streadway/amqp"
)

type RmqConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	VHost    string `yaml:"vhost"`
}

func (this_ *RmqConfig) String() string {
	return fmt.Sprintf("amqp://%v:%v@%v%v", this_.User, this_.Password, this_.Host, this_.VHost)
}

type Rmq struct {
	conn  *amqp.Connection
	chMap sync.Map
	url   string
}

func NewRmq(config *RmqConfig) (*Rmq, error) {
	url := config.String()
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return &Rmq{
		conn:  conn,
		chMap: sync.Map{},
		url:   url,
	}, nil
}

func (this_ *Rmq) Close() {
	if !this_.conn.IsClosed() {
		this_.chMap.Range(func(key, value any) bool {
			value.(*amqp.Channel).Close()
			this_.chMap.Delete(key)
			return true
		})
		this_.conn.Close()
	}
}

func (this_ *Rmq) IsClosed() bool {
	return this_.conn.IsClosed()
}

func (this_ *Rmq) Reconnect() error {
	var err error

	this_.Close()
	this_.conn, err = amqp.Dial(this_.url)
	if err != nil {
		return err
	}

	return nil
}

func (this_ *Rmq) GetChannel(key interface{}) (*amqp.Channel, error) {
	if this_.IsClosed() {
		err := this_.Reconnect()
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}

	v, ok := this_.chMap.Load(key)
	if ok {
		return v.(*amqp.Channel), nil
	}

	ch, err := this_.conn.Channel()
	if err != nil {
		return nil, err
	}

	this_.chMap.Store(key, ch)
	return ch, nil
}

func (this_ *Rmq) Conn() *amqp.Connection {
	return this_.conn
}
