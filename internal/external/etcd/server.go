package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.etcd.io/etcd/embed"
)

// ServerConfig has the information needed to execute the server.
type ServerConfig struct {
	// Location where the server data files gonna be kept.
	Data string

	// Max time to start a server before an error is generated.
	Timeout time.Duration

	// Used to handle async errors.
	AsyncErrorHandler func(err error)

	Logger *zerolog.Logger
}

// Server is used to control the Etcd embedded server.
type Server struct {
	Config ServerConfig

	asyncErrorHandler func(error)
	instance          *embed.Etcd
	ctx               context.Context
	ctxCancel         func()
}

// Start the server.
func (s *Server) Start() error {
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	cfg := embed.NewConfig()
	cfg.Dir = s.Config.Data

	var err error
	s.instance, err = embed.StartEtcd(cfg)
	if err != nil {
		return fmt.Errorf("failed to start the server: %w", err)
	}

	select {
	case <-s.instance.Server.ReadyNotify():
		go s.listenAsyncError()
		s.Config.Logger.Info().Msg("Etcd embedded server started")
		return nil
	case <-time.After(s.Config.Timeout):
		s.Stop()
		return fmt.Errorf("server did not start after '%s': %w", s.Config.Timeout.String(), err)
	}
}

// Stop the server.
func (s *Server) Stop() {
	s.instance.Close()
	s.ctxCancel()
	<-s.instance.Server.StopNotify()
}

func (s *Server) listenAsyncError() {
	for {
		select {
		case err := <-s.instance.Err():
			s.asyncErrorHandler(err)
		case <-s.ctx.Done():
			return
		}
	}
}
