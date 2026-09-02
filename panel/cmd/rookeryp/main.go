// Command rookeryp is the Rookery panel: it manages users, subscriptions,
// and registered nodes, and is the source of truth for session tokens and
// traffic statistics.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rookery/panel/internal/config"
	"github.com/rookery/panel/internal/server"
	"github.com/rookery/panel/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "configs/panel.yaml", "path to panel YAML config (optional; every setting has a default)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("rookeryp: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(filepath.Join(cfg.DataDir, "panel.db"))
	if err != nil {
		return fmt.Errorf("rookeryp: %w", err)
	}
	defer st.Close()

	if err := initAdmin(st); err != nil {
		return fmt.Errorf("rookeryp: %w", err)
	}

	srv := server.New(server.Config{
		ListenAddr:      cfg.ListenAddr,
		SessionTokenTTL: time.Duration(cfg.SessionTokenTTLMin) * time.Minute,
	}, st)

	slog.Info("rookeryp starting", "listen_addr", cfg.ListenAddr)

	if err := srv.Serve(ctx); err != nil {
		return fmt.Errorf("rookeryp: %w", err)
	}

	slog.Info("rookeryp shut down cleanly")
	return nil
}

// initAdmin ensures an admin login exists, generating and logging a
// password on first run.
func initAdmin(st *store.Store) error {
	password, err := st.EnsureAdmin()
	if err != nil {
		return err
	}
	if password != "" {
		username, _ := st.AdminUsername()
		slog.Warn("generated admin panel credentials — save these, they will not be shown again",
			"username", username, "password", password)
	}
	return nil
}

func parseLogLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return l
}
