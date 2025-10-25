package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/justinndidit/forex/internal/app"
	"github.com/justinndidit/forex/internal/config"
)

type Server struct {
	App        *app.Application
	Config     *config.Config
	httpServer *http.Server
}

func New(app *app.Application, cfg *config.Config) (*Server, error) {

	server := &Server{
		App:    app,
		Config: cfg,
	}

	return server, nil
}

func (s *Server) SetupHTTPServer(handler http.Handler) {
	s.httpServer = &http.Server{
		Addr:         "0.0.0.0:" + s.Config.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.Server.IdleTimeout) * time.Second,
	}
}

func (s *Server) Start() error {
	if s.httpServer == nil {
		return errors.New("HTTP server not initialized")
	}

	s.App.Logger.Info().
		Str("port", s.Config.Server.Port).
		Msg("starting server")

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.App.Logger.Info().
		Str("addr", s.httpServer.Addr).
		Msg("HTTP server configured")

	if err := s.App.DB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}
