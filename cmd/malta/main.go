package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/hashicorp/hcl/v2/hclsimple"
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
		config, err := parseConfig(*appServerFlag)
		handleError(err)

		doneChan := make(chan struct{})
		config.Transport.HTTP.AsyncErrorHandler = func(err error) {
			fmt.Println("async http error: %w", err)
			close(doneChan)
		}

		client := internal.Client{Config: config}
		client.Init()
		client.Start()
		wait(doneChan)
		err = client.Stop()
		handleError(err)
	}
}

func parseConfig(path string) (internal.ClientConfig, error) {
	type config struct {
		Transport struct {
			HTTP struct {
				Address string `hcl:"address"`
				Port    uint   `hcl:"port"`
			} `hcl:"http,block"`
		} `hcl:"transport,block"`
	}

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
	}, nil
}

func wait(doneChan chan struct{}) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	select {
	case <-signalChan:
	case <-doneChan:
	}
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
}
