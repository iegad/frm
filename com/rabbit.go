package com

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	VHost    string `yaml:"vhost"`
}

func (this_ *RabbitConfig) Uri() string {
	return fmt.Sprintf("amqp://%v:%v@%v%v", this_.User, this_.Password, this_.Host, this_.VHost)
}

type Rabbit struct {
	conn   *amqp.Connection
	uri    string
	chMap  sync.Map
	conmtx sync.Mutex
}

func NewRabbit(config *RabbitConfig) (*Rabbit, error) {
	url := config.Uri()
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return &Rabbit{
		conn:  conn,
		chMap: sync.Map{},
		uri:   url,
	}, nil
}

func (this_ *Rabbit) Uri() string {
	return this_.uri
}

func (this_ *Rabbit) Close() {
	this_.conmtx.Lock()

	this_.chMap.Range(func(key, value any) bool {
		value.(*amqp.Channel).Close()
		return true
	})

	this_.chMap.Clear()
	this_.conn.Close()
	this_.conn = nil
	this_.conmtx.Unlock()
}

func (this_ *Rabbit) IsClosed() bool {
	return this_.conn == nil
}

func (this_ *Rabbit) Connect() error {
	this_.conmtx.Lock()
	defer this_.conmtx.Unlock()

	if this_.conn != nil {
		return nil
	}

	conn, err := amqp.Dial(this_.uri)
	if err != nil {
		return err
	}

	this_.conn = conn
	return nil
}

func (this_ *Rabbit) GetChannel(key any) (*amqp.Channel, error) {
	if this_.IsClosed() {
		return nil, amqp.ErrClosed
	}

	v, ok := this_.chMap.Load(key)
	if ok {
		return v.(*amqp.Channel), nil
	}

	ch, err := this_.conn.Channel()
	if err != nil {
		this_.Close()
		return nil, err
	}

	this_.chMap.Store(key, ch)
	return ch, nil
}

func (this_ *Rabbit) GetQueue(chKey, qName string, durable, autoDelete, exclusive bool) (*amqp.Queue, error) {
	ch, err := this_.GetChannel(chKey)
	if err != nil {
		return nil, err
	}

	que, err := ch.QueueDeclare(qName, durable, autoDelete, exclusive, false, nil)
	if err != nil {
		if err == amqp.ErrClosed {
			this_.Close()
		}
		return nil, err
	}

	return &que, nil
}

func (this_ *Rabbit) DeleteQueue(chKey, qName string) error {
	ch, err := this_.GetChannel(chKey)
	if err != nil {
		return err
	}

	_, err = ch.QueueDelete(qName, false, false, false)
	if err != nil {
		if err == amqp.ErrClosed {
			this_.Close()
		}
	}

	return err
}
