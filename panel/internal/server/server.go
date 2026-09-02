// Package server wires the panel's HTTP surface together: the admin panel,
// the node-facing API, and the client-facing subscription endpoint.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/admin"
	"github.com/rookery/panel/internal/nodeapi"
	"github.com/rookery/panel/internal/releaseapi"
	"github.com/rookery/panel/internal/store"
	"github.com/rookery/panel/internal/subapi"
)

// Config controls the HTTP server.
type Config struct {
	ListenAddr      string
	SessionTokenTTL time.Duration
	// ReleasesDir is where uploaded client builds are saved.
	ReleasesDir string
}

// Server is the panel's HTTP server.
type Server struct {
	httpSrv *http.Server
}

// New builds a Server backed by st.
func New(cfg Config, st *store.Store) *Server {
	mux := http.NewServeMux()
	admin.NewServer(st, cfg.ReleasesDir).RegisterRoutes(mux)
	nodeapi.NewServer(st).RegisterRoutes(mux)
	subapi.NewServer(st, cfg.SessionTokenTTL).RegisterRoutes(mux)
	releaseapi.NewServer(st).RegisterRoutes(mux)

	return &Server{httpSrv: &http.Server{Addr: cfg.ListenAddr, Handler: mux}}
}

// Serve starts the HTTP server and blocks until ctx is canceled, at which
// point it shuts the server down gracefully.
func (s *Server) Serve(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: listen: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server: shutdown: %w", err)
		}
		return nil
	}
}
