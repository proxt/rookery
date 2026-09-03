package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/rookery/client/internal/vpn"
)

// killSwitchTimeout bounds every individual netsh call the kill switch
// makes — generous relative to route_windows.go's own 10s runCommand
// timeout, but this must never hang indefinitely since it runs on the
// engine's state-transition path.
const killSwitchTimeout = 10 * time.Second

// killSwitchAutoDisengageAfter bounds how long the kill switch blocks
// outbound traffic without a successful reconnect. Without this, a machine
// that genuinely has no path back to the node (no internet at all, node
// down for good) would stay blocked forever with the app itself the only
// way out — this gives up and restores connectivity instead, on the
// assumption that "unprotected but online" beats "protected but stuck".
const killSwitchAutoDisengageAfter = 10 * time.Minute

// killSwitch tracks kill-switch engagement across one Engine session (one
// Start/Stop cycle). Re-armed at the start of every Start call rather than
// being a persistent Engine field with config baked in, since SystemWide/
// KillSwitch are per-Config, not per-Engine.
type killSwitch struct {
	mu sync.Mutex
	// enabled is cfg.SystemWide && cfg.KillSwitch for this session.
	enabled bool
	// routingActive is true only once system-wide routing has actually
	// taken over the default route at least once this session (set by
	// runSystemCapture after vpn.Setup succeeds). A dropped *first*
	// connection attempt, before routing was ever active, must never
	// engage the kill switch — the user was never on the tunnel's route
	// in the first place, so blocking outbound now would cut off traffic
	// that was never protected to begin with.
	routingActive bool
	engaged       bool
	timer         *time.Timer
}

// armKillSwitch resets kill-switch state for a new Start call.
func (e *Engine) armKillSwitch(cfg Config) {
	e.ks.mu.Lock()
	defer e.ks.mu.Unlock()
	e.ks.enabled = cfg.SystemWide && cfg.KillSwitch
	e.ks.routingActive = false
	e.ks.engaged = false
	e.ks.timer = nil
}

// setKillSwitchRoutingActive is called by runSystemCapture once system-wide
// routing is actually up (true), and again when it tears down (false).
func (e *Engine) setKillSwitchRoutingActive(active bool) {
	e.ks.mu.Lock()
	e.ks.routingActive = active
	e.ks.mu.Unlock()
}

// onKillSwitchStateChanged reacts to every engine state transition. Called
// synchronously from setState — the netsh calls it triggers are quick
// (a few hundred ms) and state transitions are not a hot path, so this
// doesn't need to be async.
func (e *Engine) onKillSwitchStateChanged(state State) {
	switch state {
	case StateError:
		e.maybeEngageKillSwitch()
	case StateConnected, StateDisconnected:
		e.disengageKillSwitch("")
	}
}

func (e *Engine) maybeEngageKillSwitch() {
	e.ks.mu.Lock()
	if !e.ks.enabled || !e.ks.routingActive || e.ks.engaged {
		e.ks.mu.Unlock()
		return
	}
	e.ks.engaged = true
	e.ks.timer = time.AfterFunc(killSwitchAutoDisengageAfter, e.autoDisengageKillSwitch)
	e.ks.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), killSwitchTimeout)
	defer cancel()
	if err := vpn.EngageKillSwitch(ctx); err != nil {
		slog.Error("engine: engage kill switch", "error", err)
		return
	}
	slog.Warn("engine: kill switch engaged — outbound traffic blocked outside the tunnel")
}

// disengageKillSwitch removes the block, if one is active. warning, if
// non-empty, is surfaced to the GUI as an EventKillSwitchWarning (used by
// the auto-disengage timeout, not by normal reconnect/disconnect).
func (e *Engine) disengageKillSwitch(warning string) {
	e.ks.mu.Lock()
	if !e.ks.engaged {
		e.ks.mu.Unlock()
		return
	}
	e.ks.engaged = false
	if e.ks.timer != nil {
		e.ks.timer.Stop()
		e.ks.timer = nil
	}
	e.ks.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), killSwitchTimeout)
	defer cancel()
	if err := vpn.DisengageKillSwitch(ctx); err != nil {
		slog.Error("engine: disengage kill switch", "error", err)
	} else {
		slog.Info("engine: kill switch disengaged")
	}

	if warning != "" {
		select {
		case e.events <- Event{Type: EventKillSwitchWarning, Err: warning}:
		default:
		}
	}
}

func (e *Engine) autoDisengageKillSwitch() {
	e.disengageKillSwitch("Аварийный сброс блокировки — переподключиться не удалось за 10 минут, трафик восстановлен без защиты kill switch")
}
