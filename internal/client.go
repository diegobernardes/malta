package internal

import (
	"fmt"

	"malta/internal/transport/http"
)

// ClientConfigTransport used to configure the internal transport state.
type ClientConfigTransport struct {
	HTTP http.ServerConfig
}

// ClientConfig used to configure the internal state.
type ClientConfig struct {
	Transport ClientConfigTransport
}

// Client is used to bootstrap the application.
type Client struct {
	Config ClientConfig

	transport struct {
		http http.Server
	}
}

// Init internal state.
func (c *Client) Init() {
	c.transport.http = http.Server{
		Config: c.Config.Transport.HTTP,
	}
	c.transport.http.Init()
}

// Start the application.
func (c *Client) Start() {
	c.transport.http.Start()
}

// Stop the application.
func (c *Client) Stop() error {
	if err := c.transport.http.Stop(); err != nil {
		return fmt.Errorf("failed to stop the http transport: %w", err)
	}
	return nil
}
