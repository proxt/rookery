// Command rookeryd is the Rookery node: it terminates client WebRTC sessions
// and relays smux streams to their requested destinations.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rookery/node/internal/config"
	"github.com/rookery/node/internal/server"
	"github.com/rookery/node/internal/store"
)

// dataChannelOpenTimeout bounds how long a session waits for its
// DataChannel to open after signaling completes.
const dataChannelOpenTimeout = 30 * time.Second

// publicIPDetectTimeout bounds the one-time outbound lookup used to seed a
// default admin-panel public address on first run.
const publicIPDetectTimeout = 3 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "configs/node.yaml", "path to node YAML config (optional; every setting has a default)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("rookeryd: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(filepath.Join(cfg.DataDir, "rookery.db"))
	if err != nil {
		return fmt.Errorf("rookeryd: %w", err)
	}
	defer st.Close()

	if err := initAdmin(ctx, st); err != nil {
		return fmt.Errorf("rookeryd: %w", err)
	}

	srv, err := server.New(server.Config{
		ListenAddr:                  cfg.ListenAddr,
		ICEUDPPort:                  cfg.ICEUDPPort,
		MaxStreamsPerSession:        cfg.MaxStreams,
		DialTimeout:                 time.Duration(cfg.DialTimeoutSec) * time.Second,
		AllowPrivateNet:             cfg.AllowPrivateNet,
		BufferedAmountLowThreshold:  uint64(cfg.BufferedAmountLowKB) * 1024,
		BufferedAmountHighWaterMark: uint64(cfg.BufferedAmountHighKB) * 1024,
		DataChannelOpenTimeout:      dataChannelOpenTimeout,
	}, st)
	if err != nil {
		return fmt.Errorf("rookeryd: %w", err)
	}

	slog.Info("rookeryd starting", "listen_addr", cfg.ListenAddr, "ice_udp_port", cfg.ICEUDPPort)

	if err := srv.Serve(ctx); err != nil {
		return fmt.Errorf("rookeryd: %w", err)
	}

	slog.Info("rookeryd shut down cleanly")
	return nil
}

// initAdmin ensures an admin login exists (generating and logging a
// password on first run) and seeds a best-effort default public address so
// generated links work out of the box before the operator sets a real
// domain in the admin panel.
func initAdmin(ctx context.Context, st *store.Store) error {
	password, err := st.EnsureAdmin()
	if err != nil {
		return err
	}
	if password != "" {
		username, _ := st.AdminUsername()
		slog.Warn("generated admin panel credentials — save these, they will not be shown again",
			"username", username, "password", password)
	}

	detectCtx, cancel := context.WithTimeout(ctx, publicIPDetectTimeout)
	defer cancel()
	if ip, err := detectPublicIP(detectCtx); err == nil {
		changed, err := st.SetPublicAddrIfEmpty(fmt.Sprintf("http://%s:8080", ip))
		if err != nil {
			slog.Warn("seed default public address", "error", err)
		} else if changed {
			slog.Info("seeded default public address from detected IP; set your real domain in the admin panel once Caddy is configured", "ip", ip)
		}
	}
	return nil
}

func detectPublicIP(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parseLogLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return l
}
