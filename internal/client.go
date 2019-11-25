package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"malta/internal/external/etcd"
	"malta/internal/service/node"
	"malta/internal/transport/http"
)

// ClientConfigExternalEtcdClient representation.
type ClientConfigExternalEtcdClient struct {
	Enable         bool
	DialTimeout    time.Duration
	RequestTimeout time.Duration
	Endpoints      []string
}

// ClientConfigExternalEtcdEmbed representation.
type ClientConfigExternalEtcdEmbed struct {
	Enable                bool
	Data                  string
	InitializationTimeout time.Duration
}

// ClientConfigExternalEtcd representation.
type ClientConfigExternalEtcd struct {
	Embed  ClientConfigExternalEtcdEmbed
	Client ClientConfigExternalEtcdClient
}

// ClientConfigExternal representation.
type ClientConfigExternal struct {
	Etcd ClientConfigExternalEtcd
}

// ClientConfigTransport used to configure the internal transport state.
type ClientConfigTransport struct {
	HTTP http.ServerConfig
}

// ClientConfig used to configure the internal state.
type ClientConfig struct {
	Transport ClientConfigTransport
	External  ClientConfigExternal
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

	external struct {
		etcd struct {
			server etcd.Server
			client etcd.Client
			node   etcd.Node
		}
	}
}

// Init internal state.
func (c *Client) Init() error {
	if c.Config.External.Etcd.Embed.Enable {
		c.external.etcd.server.Config = etcd.ServerConfig{
			Data:    c.Config.External.Etcd.Embed.Data,
			Timeout: c.Config.External.Etcd.Embed.InitializationTimeout,
			Logger:  &c.Config.Logger,
		}
	}

	if c.Config.External.Etcd.Client.Enable {
		c.external.etcd.client.Config = etcd.ClientConfig{
			DialTimeout:    c.Config.External.Etcd.Client.DialTimeout,
			RequestTimeout: c.Config.External.Etcd.Client.RequestTimeout,
			Endpoints:      c.Config.External.Etcd.Client.Endpoints,
		}

		c.external.etcd.node = etcd.Node{Client: &c.external.etcd.client}
		c.service.node.Repository = &c.external.etcd.node
	}

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
	if c.Config.External.Etcd.Embed.Enable {
		if err := c.external.etcd.server.Start(); err != nil {
			return fmt.Errorf("failed to start the etcd server: %w", err)
		}
	}

	if c.Config.External.Etcd.Client.Enable {
		if err := c.external.etcd.client.Start(); err != nil {
			return fmt.Errorf("failed to start the etcd client: %w", err)
		}
		c.external.etcd.node.Open()
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

	if c.Config.External.Etcd.Client.Enable {
		if err := c.external.etcd.client.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop the etcd client: %w", err))
		}
	}

	if c.Config.External.Etcd.Embed.Enable {
		c.external.etcd.server.Stop()
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
