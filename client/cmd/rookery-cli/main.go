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
	"github.com/rookery/client/internal/subscription"
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

	sub, err := subscription.Fetch(ctx, cfg.PanelAddr, cfg.Token)
	if err != nil {
		return fmt.Errorf("rookery-cli: resolve subscription: %w", err)
	}
	node, err := pickNode(sub, cfg.NodeID)
	if err != nil {
		return fmt.Errorf("rookery-cli: %w", err)
	}

	eng := engine.New()
	engCfg := engine.Config{
		NodeAddr:  node.Address,
		SOCKSAddr: cfg.SOCKSAddr,
		TokenFunc: func(ctx context.Context) (string, error) {
			sub, err := subscription.Fetch(ctx, cfg.PanelAddr, cfg.Token)
			if err != nil {
				return "", err
			}
			n, ok := sub.FindNode(node.ID)
			if !ok {
				return "", fmt.Errorf("node %q is no longer in this subscription", node.ID)
			}
			return n.SessionToken, nil
		},
		BufferedAmountLowThreshold:  uint64(cfg.BufferedAmountLowKB) * 1024,
		BufferedAmountHighWaterMark: uint64(cfg.BufferedAmountHighKB) * 1024,
		ReconnectMaxBackoff:         time.Duration(cfg.ReconnectMaxBackoffS) * time.Second,
		SystemWide:                  cfg.SystemWide,
	}

	if err := eng.Start(ctx, engCfg); err != nil {
		return fmt.Errorf("rookery-cli: %w", err)
	}
	slog.Info("rookery-cli started", "subscription", sub.Name, "node", node.Name, "node_addr", node.Address, "socks_addr", cfg.SOCKSAddr)

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

// pickNode selects nodeID from sub, or its first node if nodeID is empty.
func pickNode(sub subscription.Subscription, nodeID string) (subscription.Node, error) {
	if len(sub.Nodes) == 0 {
		return subscription.Node{}, fmt.Errorf("subscription %q has no nodes", sub.Name)
	}
	if nodeID == "" {
		return sub.Nodes[0], nil
	}
	n, ok := sub.FindNode(nodeID)
	if !ok {
		return subscription.Node{}, fmt.Errorf("node %q not found in subscription %q", nodeID, sub.Name)
	}
	return n, nil
}

func parseLogLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return l
}
