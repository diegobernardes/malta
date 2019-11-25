package etcd

import (
	"fmt"
	"time"

	"go.etcd.io/etcd/clientv3"
)

// ClientConfig used to initialize the client.
type ClientConfig struct {
	DialTimeout    time.Duration
	RequestTimeout time.Duration
	Endpoints      []string
}

// Client used to interact with the Etcd database.
type Client struct {
	Config ClientConfig

	instance *clientv3.Client
}

// Start the client.
func (c *Client) Start() error {
	var err error
	c.instance, err = clientv3.New(clientv3.Config{
		DialTimeout: c.Config.DialTimeout,
		Endpoints:   c.Config.Endpoints,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to etcd: %w", err)
	}
	return nil
}

// Stop the client.
func (c *Client) Stop() error {
	if err := c.instance.Close(); err != nil {
		return fmt.Errorf("failed to close the connection with etcd: %w", err)
	}
	return nil
}
