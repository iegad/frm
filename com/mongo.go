package com

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConfig struct {
	Host     string `yaml:"host"`
	Port     int32  `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func NewMongo(c *MongoConfig) (*mongo.Client, error) {
	const timeout = time.Second * 15

	uri := fmt.Sprintf("mongodb://%v:%v", c.Host, c.Port)
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetConnectTimeout(timeout))
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	return client, nil
}
