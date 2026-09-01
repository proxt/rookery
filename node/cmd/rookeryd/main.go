// Command rookeryd is the Rookery node: it terminates client WebRTC sessions
// and relays smux streams to their requested destinations.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rookery/node/internal/config"
	"github.com/rookery/node/internal/server"
)

// dataChannelOpenTimeout bounds how long a session waits for its
// DataChannel to open after signaling completes.
const dataChannelOpenTimeout = 30 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "configs/node.yaml", "path to node YAML config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("rookeryd: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv, err := server.New(server.Config{
		ListenAddr:                  cfg.ListenAddr,
		Secret:                      cfg.Secret,
		ICEUDPPort:                  cfg.ICEUDPPort,
		MaxStreamsPerSession:        cfg.MaxStreams,
		DialTimeout:                 time.Duration(cfg.DialTimeoutSec) * time.Second,
		AllowPrivateNet:             cfg.AllowPrivateNet,
		BufferedAmountLowThreshold:  uint64(cfg.BufferedAmountLowKB) * 1024,
		BufferedAmountHighWaterMark: uint64(cfg.BufferedAmountHighKB) * 1024,
		DataChannelOpenTimeout:      dataChannelOpenTimeout,
	})
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

func parseLogLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return l
}
