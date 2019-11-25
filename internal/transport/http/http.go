package http

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"

	"malta/internal/transport/http/handler"
	"malta/internal/transport/http/shared"
)

// ServerConfig used to configure the server internal state.
type ServerConfig struct {
	Address string
	Port    uint
	Handler struct {
		Node handler.Node
	}
	AsyncErrorHandler func(error)
	Logger            zerolog.Logger
}

// Server implements the HTTP server.
type Server struct {
	Config   ServerConfig
	instance *http.Server
}

// Init internal state.
func (s *Server) Init() error {
	s.instance = &http.Server{
		Addr: fmt.Sprintf("%s:%d", s.Config.Address, s.Config.Port),
	}

	writer := shared.Writer{Logger: s.Config.Handler.Node.Writer.Logger}
	s.Config.Handler.Node.Writer = writer

	if err := s.Config.Handler.Node.Init(); err != nil {
		return fmt.Errorf("node handler initialization error: %w", err)
	}
	return nil
}

// Start the server.
func (s *Server) Start() {
	r := chi.NewRouter()
	r.Get("/nodes", s.Config.Handler.Node.Index)
	r.Post("/nodes", s.Config.Handler.Node.Create)
	s.instance.Handler = r

	go func() {
		if err := s.instance.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			err = fmt.Errorf("failed to start: %w", err)
			s.Config.AsyncErrorHandler(err)
		}
	}()
}

// Stop the server.
func (s *Server) Stop() error {
	if err := s.instance.Close(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to stop: %w", err)
	}
	return nil
}
