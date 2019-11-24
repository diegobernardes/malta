package http

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

// ServerConfig used to configure the server internal state.
type ServerConfig struct {
	Address           string
	Port              uint
	AsyncErrorHandler func(error)
}

// Server implements the HTTP server.
type Server struct {
	Config ServerConfig

	instance *http.Server
}

// Init internal state.
func (s *Server) Init() {
	s.instance = &http.Server{
		Addr: fmt.Sprintf("%s:%d", s.Config.Address, s.Config.Port),
	}
}

// Start the server.
func (s *Server) Start() {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome")) // nolint: errcheck
	})
	s.instance.Handler = r

	go func() {
		if err := s.instance.ListenAndServe(); err != nil {
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
