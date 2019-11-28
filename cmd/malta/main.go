package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/rs/zerolog"
	"gopkg.in/alecthomas/kingpin.v2"

	"malta/internal"
	"malta/internal/database/sqlite3"
	"malta/internal/service/node"
	"malta/internal/transport/http"
)

func main() {
	app := kingpin.New("malta", "Distributed processing engine.")
	appServer := app.Command("server", "Start the server.")
	appServerFlag := appServer.Flag("config", "Config path").
		Short('c').
		Default("malta.hcl").
		String()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case appServer.FullCommand():
		logger := logger()

		config, err := parseConfig(*appServerFlag, logger)
		handleError(err, logger)
		config.Logger = logger

		doneChan := make(chan struct{})
		config.Transport.HTTP.AsyncErrorHandler = func(err error) {
			logger.Err(err).Msg("async http error")
			close(doneChan)
		}

		client := internal.Client{Config: config}
		if err := client.Init(); err != nil {
			err = fmt.Errorf("client initialization error: %w", err)
			handleError(err, logger)
		}

		if err := client.Start(); err != nil {
			err = fmt.Errorf("client start error: %w", err)
			handleError(err, logger)
		}

		wait(doneChan)
		err = client.Stop()
		handleError(err, logger)
	}
}

type config struct {
	Transport struct {
		HTTP struct {
			Address string `hcl:"address"`
			Port    uint   `hcl:"port"`
		} `hcl:"http,block"`
	} `hcl:"transport,block"`
	Service struct {
		Node struct {
			Health struct {
				Concurrency int    `hcl:"concurrency"`
				Interval    string `hcl:"interval"`
			} `hcl:"health,block"`
		} `hcl:"node,block"`
	} `hcl:"service,block"`
	Database struct {
		SQLite3 struct {
			File               string `hcl:"file,optional"`
			MaxOpenConnections int    `hcl:"max-open-connections,optional"`
			MaxIdleConnections int    `hcl:"max-idle-connections,optional"`
			ConnectionLifetime string `hcl:"connection-lifetime,optional"`
		} `hcl:"sqlite3,block"`
	} `hcl:"database,block"`
}

func wait(doneChan chan struct{}) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	select {
	case <-signalChan:
	case <-doneChan:
	}
}

func logger() zerolog.Logger {
	out := zerolog.NewConsoleWriter()
	return zerolog.New(out).With().Caller().Timestamp().Logger()
}

func handleError(err error, logger zerolog.Logger) {
	if err != nil {
		logger.Fatal().Msg(err.Error())
	}
}

func parseConfig(path string, logger zerolog.Logger) (internal.ClientConfig, error) {
	var cfg config
	if err := hclsimple.DecodeFile(path, nil, &cfg); err != nil {
		return internal.ClientConfig{}, fmt.Errorf("load config: %w", err)
	}

	duration := parseTimeDuration(logger)
	return internal.ClientConfig{
		Transport: internal.ClientConfigTransport{
			HTTP: http.ServerConfig{
				Address: cfg.Transport.HTTP.Address,
				Port:    cfg.Transport.HTTP.Port,
			},
		},
		Service: internal.ClientConfigService{
			Node: internal.ClientConfigServiceNode{
				Health: node.HealthConfig{
					Interval:    duration(cfg.Service.Node.Health.Interval),
					Concurrency: cfg.Service.Node.Health.Concurrency,
				},
			},
		},
		Database: internal.ClientConfigDatabase{
			SQLite3: sqlite3.ClientConfig{
				DatabaseFile:       cfg.Database.SQLite3.File,
				MaxOpenConnections: cfg.Database.SQLite3.MaxOpenConnections,
				MaxIdleConnections: cfg.Database.SQLite3.MaxIdleConnections,
				ConnectionLifetime: duration(cfg.Database.SQLite3.ConnectionLifetime),
			},
		},
	}, nil
}

func parseTimeDuration(logger zerolog.Logger) func(string) time.Duration {
	return func(value string) time.Duration {
		if value == "" {
			return 0
		}
		duration, err := time.ParseDuration(value)
		if err != nil {
			err = fmt.Errorf("failed to parse '%s' into a time duration: %w", value, err)
			handleError(err, logger)
		}
		return duration
	}
}
