package dkex

import (
	"time"

	"github.com/docker/docker/client"
)

func NewDocker(host string) (*client.Client, error) {
	c, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation(), client.WithTimeout(time.Second*5))
	if err != nil {
		return nil, err
	}
	return c, nil
}
