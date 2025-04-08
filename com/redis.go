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
	const timeout = time.Second * 15

	r := redis.NewClient(&redis.Options{
		Addr:        c.Addr,
		Username:    c.Username,
		Password:    c.Password,
		DialTimeout: timeout,
		IdleTimeout: -1,
	})

	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	err := r.Ping(ctx).Err()
	if err != nil {
		r.Close()
		return nil, err
	}

	return r, nil
}
