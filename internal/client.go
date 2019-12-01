package internal

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"

	"malta/internal/database"
	"malta/internal/database/sqlite3"
	"malta/internal/service"
	"malta/internal/service/node"
	transportHTTP "malta/internal/transport/http"
)

// ClientConfigDatabase representation.
type ClientConfigDatabase struct {
	SQLite3 sqlite3.ClientConfig
}

// ClientConfigTransport used to configure the internal transport state.
type ClientConfigTransport struct {
	HTTP transportHTTP.ServerConfig
}

// ClientConfigServiceNode used to configure the internal node service state.
type ClientConfigServiceNode struct {
	Client node.ClientConfig
	Health node.HealthConfig
}

// ClientConfigService used to configure the internal service state.
type ClientConfigService struct {
	Node ClientConfigServiceNode
}

// ClientConfig used to configure the internal state.
type ClientConfig struct {
	Transport ClientConfigTransport
	Service   ClientConfigService
	Database  ClientConfigDatabase
	Logger    zerolog.Logger
}

// Client is used to bootstrap the application.
type Client struct {
	Config ClientConfig

	service struct {
		node       node.Client
		nodeHealth node.Health
	}

	transport struct {
		http transportHTTP.Server
	}

	database struct {
		sqlite3 struct {
			client    sqlite3.Client
			node      sqlite3.Node
			nodeCheck sqlite3.NodeCheck
		}
	}
}

// Init internal state.
func (c *Client) Init() error {
	c.database.sqlite3.node.Client = &c.database.sqlite3.client
	c.database.sqlite3.nodeCheck.Client = &c.database.sqlite3.client
	c.database.sqlite3.client.Config = c.Config.Database.SQLite3
	c.database.sqlite3.client.Config.ClientLifecycleHook = append(
		c.database.sqlite3.client.Config.ClientLifecycleHook,
		&c.database.sqlite3.node,
		&c.database.sqlite3.nodeCheck,
	)
	if err := c.database.sqlite3.client.Init(); err != nil {
		return fmt.Errorf("failed to initialize sqlite3 client: %w", err)
	}

	c.service.node.Config = c.Config.Service.Node.Client
	c.service.node.Notification = &c.service.nodeHealth
	c.service.node.Repository = &c.database.sqlite3.node
	c.service.node.Transaction = &c.database.sqlite3.client
	c.service.node.TransactionHandler = database.TransactionHandler(c.Config.Logger)

	c.service.nodeHealth.Config = c.Config.Service.Node.Health
	c.service.nodeHealth.Config.CheckRepository = &c.database.sqlite3.nodeCheck
	c.service.nodeHealth.Config.Repository = &c.database.sqlite3.node
	c.service.nodeHealth.Config.Logger = c.Config.Logger
	c.service.nodeHealth.Config.HTTPClient = http.DefaultClient

	c.transport.http = transportHTTP.Server{Config: c.Config.Transport.HTTP}
	c.transport.http.Config.Handler.Node.Repository = &c.service.node
	c.transport.http.Config.Handler.Node.ResourceAddress = func(node service.Node) string {
		return fmt.Sprintf(
			"http://%s:%d/nodes/%d",
			c.transport.http.Config.Address,
			c.transport.http.Config.Port,
			node.ID,
		)
	}
	c.transport.http.Config.Handler.Node.ResourceID = func(r *http.Request) string {
		return chi.URLParam(r, "id")
	}
	c.transport.http.Config.Logger = c.Config.Logger
	if err := c.transport.http.Init(); err != nil {
		return fmt.Errorf("http transport initialization error: %w", err)
	}

	return nil
}

// Start the application.
func (c *Client) Start() error {
	c.Config.Logger.Info().Msg("Starting application")

	if err := c.database.sqlite3.client.Start(); err != nil {
		return fmt.Errorf("failed to start sqlite3 database: %w", err)
	}

	if err := c.service.nodeHealth.Start(); err != nil {
		return fmt.Errorf("failed to start the node health service: %w", err)
	}

	c.transport.http.Start()
	c.Config.Logger.Info().Msg("Application started")
	return nil
}

// Stop the application.
func (c *Client) Stop() error {
	var errs []error
	c.service.nodeHealth.Stop()

	c.Config.Logger.Info().Msg("Stopping application")
	if err := c.transport.http.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop the http transport: %w", err))
	}

	if err := c.database.sqlite3.client.Stop(); err != nil {
		return fmt.Errorf("failed to stop sqlite3 database: %w", err)
	}

	switch len(errs) {
	case 0:
		c.Config.Logger.Info().Msg("Application stopped")
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
