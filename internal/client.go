package internal

import (
	"fmt"

	"malta/internal/service/node"
	"malta/internal/transport/http"

	"github.com/rs/zerolog"
)

// ClientConfigTransport used to configure the internal transport state.
type ClientConfigTransport struct {
	HTTP http.ServerConfig
}

// ClientConfig used to configure the internal state.
type ClientConfig struct {
	Transport ClientConfigTransport
	Logger    zerolog.Logger
}

// Client is used to bootstrap the application.
type Client struct {
	Config ClientConfig

	service struct {
		node node.Client
	}

	transport struct {
		http http.Server
	}
}

// Init internal state.
func (c *Client) Init() error {
	c.transport.http = http.Server{Config: c.Config.Transport.HTTP}
	c.transport.http.Config.Handler.Node.Repository = &c.service.node
	c.transport.http.Config.Logger = c.Config.Logger

	if err := c.transport.http.Init(); err != nil {
		return fmt.Errorf("http transport initialization error: %w", err)
	}
	return nil
}

// Start the application.
func (c *Client) Start() {
	c.Config.Logger.Info().Msg("Starting application")
	c.transport.http.Start()
}

// Stop the application.
func (c *Client) Stop() error {
	c.Config.Logger.Info().Msg("Stopping application")
	if err := c.transport.http.Stop(); err != nil {
		return fmt.Errorf("failed to stop the http transport: %w", err)
	}
	return nil
}
