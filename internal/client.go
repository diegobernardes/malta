package internal

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"malta/internal/database"
	"malta/internal/database/sqlite3"
	"malta/internal/service/node"
	"malta/internal/transport/http"
)

// ClientConfigDatabase representation.
type ClientConfigDatabase struct {
	SQLite3 sqlite3.ClientConfig
}

// ClientConfigTransport used to configure the internal transport state.
type ClientConfigTransport struct {
	HTTP http.ServerConfig
}

// ClientConfig used to configure the internal state.
type ClientConfig struct {
	Transport ClientConfigTransport
	Database  ClientConfigDatabase
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

	database struct {
		sqlite3 struct {
			client sqlite3.Client
			node   sqlite3.Node
		}
	}
}

// Init internal state.
func (c *Client) Init() error {
	c.database.sqlite3.node.Client = &c.database.sqlite3.client
	c.database.sqlite3.client.Config = c.Config.Database.SQLite3
	c.database.sqlite3.client.Config.ClientLifecycleHook = append(
		c.database.sqlite3.client.Config.ClientLifecycleHook,
		&c.database.sqlite3.node,
	)
	if err := c.database.sqlite3.client.Init(); err != nil {
		return fmt.Errorf("failed to initialize sqlite3 client: %w", err)
	}

	c.service.node.Repository = &c.database.sqlite3.node
	c.service.node.Transaction = &c.database.sqlite3.client
	c.service.node.TransactionHandler = database.TransactionHandler(c.Config.Logger)

	c.transport.http = http.Server{Config: c.Config.Transport.HTTP}
	c.transport.http.Config.Handler.Node.Repository = &c.service.node
	c.transport.http.Config.Logger = c.Config.Logger
	if err := c.transport.http.Init(); err != nil {
		return fmt.Errorf("http transport initialization error: %w", err)
	}

	return nil
}

// Start the application.
func (c *Client) Start() error {
	if err := c.database.sqlite3.client.Start(); err != nil {
		return fmt.Errorf("failed to start sqlite3 database: %w", err)
	}

	c.Config.Logger.Info().Msg("Starting application")
	c.transport.http.Start()
	return nil
}

// Stop the application.
func (c *Client) Stop() error {
	var errs []error

	c.Config.Logger.Info().Msg("Stopping application")
	if err := c.transport.http.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop the http transport: %w", err))
	}

	if err := c.database.sqlite3.client.Stop(); err != nil {
		return fmt.Errorf("failed to stop sqlite3 database: %w", err)
	}

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		values := make([]string, len(errs))
		for i, err := range errs {
			values[i] = err.Error()
		}
		return fmt.Errorf("multiple errors detected during stop: (%s)", strings.Join(values, " | "))
	}
}
