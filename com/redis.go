package com

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func NewRedisClient(c *RedisConfig) (*redis.Client, error) {
	r := redis.NewClient(&redis.Options{
		Addr:        c.Addr,
		Username:    c.Username,
		Password:    c.Password,
		DialTimeout: time.Second * 30,
		IdleTimeout: time.Minute,
	})

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*15)
	defer cancel()

	err := r.Ping(ctx).Err()
	if err != nil {
		r.Close()
		return nil, err
	}

	return r, nil
}
