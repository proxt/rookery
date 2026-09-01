// Command rookery-cli is the headless client front end: it drives
// client/internal/engine without any GUI.
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

	"github.com/rookery/client/internal/config"
	"github.com/rookery/client/internal/engine"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "configs/client.yaml", "path to client YAML config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("rookery-cli: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	eng := engine.New()
	engCfg := engine.Config{
		NodeAddr:                    cfg.NodeAddr,
		SOCKSAddr:                   cfg.SOCKSAddr,
		UserID:                      cfg.UserID,
		Secret:                      cfg.Secret,
		BufferedAmountLowThreshold:  uint64(cfg.BufferedAmountLowKB) * 1024,
		BufferedAmountHighWaterMark: uint64(cfg.BufferedAmountHighKB) * 1024,
		ReconnectMaxBackoff:         time.Duration(cfg.ReconnectMaxBackoffS) * time.Second,
		SystemWide:                  cfg.SystemWide,
	}

	if err := eng.Start(ctx, engCfg); err != nil {
		return fmt.Errorf("rookery-cli: %w", err)
	}
	slog.Info("rookery-cli started", "node_addr", cfg.NodeAddr, "socks_addr", cfg.SOCKSAddr)

	go logEvents(ctx, eng)

	<-ctx.Done()
	slog.Info("rookery-cli shutting down")
	eng.Stop()
	return nil
}

func logEvents(ctx context.Context, eng *engine.Engine) {
	for {
		select {
		case ev := <-eng.Events():
			switch ev.Type {
			case engine.EventStateChanged:
				if ev.Err != "" {
					slog.Warn("tunnel state changed", "state", ev.State.String(), "error", ev.Err)
				} else {
					slog.Info("tunnel state changed", "state", ev.State.String())
				}
			case engine.EventStatsTick:
				status := eng.Status()
				slog.Debug("stats",
					"up_bps", ev.BytesUpPerSec,
					"down_bps", ev.BytesDownPerSec,
					"rtt", status.RTT,
					"active_streams", status.ActiveStreams,
				)
			}
		case <-ctx.Done():
			return
		}
	}
}

func parseLogLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return l
}
