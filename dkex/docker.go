package dkex

import "github.com/docker/docker/client"

func NewDocker(host string) (*client.Client, error) {
	c, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return c, nil
}
