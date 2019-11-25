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
	External struct {
		Etcd []struct {
			Embed                 bool     `hcl:"embed,optional"`
			Data                  string   `hcl:"data,optional"`
			InitializationTimeout string   `hcl:"initialization-timeout,optional"`
			DialTimeout           string   `hcl:"dial-timeout,optional"`
			RequestTimeout        string   `hcl:"request-timeout,optional"`
			Endpoints             []string `hcl:"endpoints,optional"`
		} `hcl:"etcd,block"`
	} `hcl:"external,block"`
}

func parseConfig(path string, logger zerolog.Logger) (internal.ClientConfig, error) {
	var cfg config
	if err := hclsimple.DecodeFile(path, nil, &cfg); err != nil {
		return internal.ClientConfig{}, fmt.Errorf("load config: %w", err)
	}

	return internal.ClientConfig{
		Transport: internal.ClientConfigTransport{
			HTTP: http.ServerConfig{
				Address: cfg.Transport.HTTP.Address,
				Port:    cfg.Transport.HTTP.Port,
			},
		},
		External: internal.ClientConfigExternal{
			Etcd: parseConfigEtcd(cfg, logger),
		},
	}, nil
}

func parseConfigEtcd(cfg config, logger zerolog.Logger) internal.ClientConfigExternalEtcd {
	if len(cfg.External.Etcd) > 2 {
		err := fmt.Errorf("invalid quantity of etcd block configuration")
		handleError(err, logger)
	}

	var result internal.ClientConfigExternalEtcd
	duration := parseTimeDuration(logger)
	for _, etcd := range cfg.External.Etcd {
		if etcd.Embed {
			result.Embed.Enable = true
			result.Embed.Data = etcd.Data
			result.Embed.InitializationTimeout = duration(etcd.InitializationTimeout)
		} else {
			result.Client.Enable = true
			result.Client.DialTimeout = duration(etcd.DialTimeout)
			result.Client.RequestTimeout = duration(etcd.RequestTimeout)
			result.Client.Endpoints = etcd.Endpoints
		}
	}

	return result
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
		logger.Error().Msg(err.Error())
		os.Exit(-1)
	}
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
