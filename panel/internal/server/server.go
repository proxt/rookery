// Package server wires the panel's HTTP surface together: the admin panel,
// the node-facing API, and the client-facing subscription endpoint.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/admin"
	"github.com/rookery/panel/internal/nodeapi"
	"github.com/rookery/panel/internal/releaseapi"
	"github.com/rookery/panel/internal/store"
	"github.com/rookery/panel/internal/subapi"
)

// auditLogRetention bounds how long audit log entries are kept — an admin
// panel produces very little of this data, but "kept forever" is still an
// unbounded table over a long enough deployment lifetime. 90 days covers any
// realistic "who changed this" investigation without growing without limit.
const auditLogRetention = 90 * 24 * time.Hour

// auditLogPruneInterval is how often the retention sweep runs. Coarse on
// purpose — this is housekeeping, not a hot path.
const auditLogPruneInterval = 24 * time.Hour

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
	store   *store.Store
}

// New builds a Server backed by st.
func New(cfg Config, st *store.Store) *Server {
	mux := http.NewServeMux()
	admin.NewServer(st, cfg.ReleasesDir).RegisterRoutes(mux)
	nodeapi.NewServer(st).RegisterRoutes(mux)
	subapi.NewServer(st, cfg.SessionTokenTTL).RegisterRoutes(mux)
	releaseapi.NewServer(st).RegisterRoutes(mux)

	return &Server{httpSrv: &http.Server{Addr: cfg.ListenAddr, Handler: mux}, store: st}
}

// Serve starts the HTTP server and blocks until ctx is canceled, at which
// point it shuts the server down gracefully.
func (s *Server) Serve(ctx context.Context) error {
	go s.pruneAuditLogPeriodically(ctx)

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

// pruneAuditLogPeriodically deletes audit log entries older than
// auditLogRetention, once at startup and then every auditLogPruneInterval,
// until ctx is canceled.
func (s *Server) pruneAuditLogPeriodically(ctx context.Context) {
	prune := func() {
		n, err := s.store.PruneAuditLog(auditLogRetention)
		if err != nil {
			slog.Error("server: prune audit log", "error", err)
			return
		}
		if n > 0 {
			slog.Info("server: pruned audit log", "deleted", n)
		}
	}

	prune()
	ticker := time.NewTicker(auditLogPruneInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			prune()
		}
	}
}
