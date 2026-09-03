// Package engine is the client's tunnel core: SOCKS5 ingress, the WebRTC/smux
// session, reconnect, and stats. It has no dependency on any GUI toolkit; the
// GUI and the headless CLI both drive it through this package's exported API.
package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/xtaci/smux"
)

// State is the tunnel's connection state.
type State int

const (
	StateDisconnected State = iota
	StateConnecting
	StateConnected
	StateError
)

func (s State) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Config controls a single Start call.
type Config struct {
	// NodeAddr is the relay node's public base URL. Fixed for the lifetime
	// of a Start call — picking a different node means Stop, then Start
	// again with a new Config.
	NodeAddr  string
	SOCKSAddr string
	// TokenFunc returns a session token authenticating this client to
	// NodeAddr, valid at least at the moment it returns. Called before
	// every signaling attempt (including reconnects), not just once, since
	// panel-issued tokens expire — a typical implementation re-fetches the
	// subscription from the panel each time rather than caching a token
	// that might outlive its TTL.
	TokenFunc                   func(ctx context.Context) (string, error)
	BufferedAmountLowThreshold  uint64
	BufferedAmountHighWaterMark uint64
	ReconnectMaxBackoff         time.Duration
	// SystemWide routes all of the OS's traffic through a virtual network
	// adapter instead of requiring apps to be configured for the SOCKS5
	// proxy individually. Requires administrator privileges.
	SystemWide bool
	// KillSwitch blocks all outbound traffic outside the tunnel adapter
	// whenever the tunnel drops while SystemWide routing is active, until
	// it reconnects (or killSwitchAutoDisengageAfter elapses). No effect
	// unless SystemWide is also true.
	KillSwitch bool
}

// StatusSnapshot is a point-in-time read of the tunnel's status.
type StatusSnapshot struct {
	State         State         `json:"state"`
	Uptime        time.Duration `json:"uptimeNs"`
	RTT           time.Duration `json:"rttNs"`
	ActiveStreams int           `json:"activeStreams"`
	BytesUp       uint64        `json:"bytesUp"`
	BytesDown     uint64        `json:"bytesDown"`
	LastError     string        `json:"lastError"`
	// KillSwitchEngaged is true while the kill switch is actively blocking
	// outbound traffic outside the tunnel (see killswitch.go).
	KillSwitchEngaged bool `json:"killSwitchEngaged"`
}

// EventType identifies what kind of Event was emitted.
type EventType int

const (
	EventStateChanged EventType = iota
	EventStatsTick
	// EventKillSwitchWarning is pushed when the kill switch auto-disengages
	// after killSwitchAutoDisengageAfter without a successful reconnect —
	// Err carries a human-readable message for the GUI to surface.
	EventKillSwitchWarning
)

// Event is pushed on the Events() channel: a state transition or a periodic
// throughput sample.
type Event struct {
	Type EventType `json:"type"`

	// Set when Type == EventStateChanged.
	State State  `json:"state"`
	Err   string `json:"err"`

	// Set when Type == EventStatsTick.
	BytesUpPerSec   uint64 `json:"bytesUpPerSec"`
	BytesDownPerSec uint64 `json:"bytesDownPerSec"`
}

// Engine is the tunnel core. It is instantiated with New and driven through
// Start/Stop; Status and Events are safe to call at any time, including
// before the first Start.
type Engine struct {
	events chan Event

	runMu   sync.Mutex
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	stateMu     sync.Mutex
	state       State
	lastErr     string
	connectedAt time.Time

	rtt           atomic.Int64
	bytesUp       atomic.Uint64
	bytesDown     atomic.Uint64
	activeStreams atomic.Int32

	sess     atomic.Pointer[smux.Session]
	activePC atomic.Pointer[webrtc.PeerConnection]

	httpClient *http.Client

	ks killSwitch
}

// New creates an idle Engine. Call Start to begin tunneling.
func New() *Engine {
	return &Engine{
		events:     make(chan Event, 128),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Start begins tunneling: it opens the local SOCKS5 listener and starts the
// WebRTC session with automatic reconnect. It returns once the SOCKS5
// listener is up; the session itself connects asynchronously and its
// progress is reported through Events/Status. Start fails if the Engine is
// already running.
func (e *Engine) Start(ctx context.Context, cfg Config) error {
	e.runMu.Lock()
	defer e.runMu.Unlock()

	if e.running {
		return fmt.Errorf("engine: already running")
	}

	listener, err := net.Listen("tcp", cfg.SOCKSAddr)
	if err != nil {
		return fmt.Errorf("engine: listen socks5 on %s: %w", cfg.SOCKSAddr, err)
	}

	innerCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.running = true
	e.armKillSwitch(cfg)
	e.setState(innerCtx, StateConnecting, "")

	e.wg.Add(4)
	go func() { defer e.wg.Done(); <-innerCtx.Done(); listener.Close() }()
	go func() { defer e.wg.Done(); e.acceptSOCKS(innerCtx, listener) }()
	go func() { defer e.wg.Done(); e.connectLoop(innerCtx, cfg) }()
	go func() { defer e.wg.Done(); e.statsLoop(innerCtx) }()

	if cfg.SystemWide {
		e.wg.Add(1)
		go func() { defer e.wg.Done(); e.runSystemCapture(innerCtx, cfg) }()
	}

	return nil
}

// Stop ends tunneling and blocks until every goroutine Start spawned has
// exited. It is a no-op if the Engine is not running. The Engine may be
// Start-ed again afterward.
func (e *Engine) Stop() {
	e.runMu.Lock()
	if !e.running {
		e.runMu.Unlock()
		return
	}
	cancel := e.cancel
	e.running = false
	e.runMu.Unlock()

	cancel()
	e.wg.Wait()

	e.sess.Store(nil)
	e.activePC.Store(nil)
	e.activeStreams.Store(0)
	e.setState(context.Background(), StateDisconnected, "")
}

// Status returns a point-in-time snapshot of the tunnel's status.
func (e *Engine) Status() StatusSnapshot {
	e.stateMu.Lock()
	state := e.state
	lastErr := e.lastErr
	connectedAt := e.connectedAt
	e.stateMu.Unlock()

	var uptime time.Duration
	if state == StateConnected && !connectedAt.IsZero() {
		uptime = time.Since(connectedAt)
	}

	e.ks.mu.Lock()
	killSwitchEngaged := e.ks.engaged
	e.ks.mu.Unlock()

	return StatusSnapshot{
		State:             state,
		Uptime:            uptime,
		RTT:               time.Duration(e.rtt.Load()),
		ActiveStreams:     int(e.activeStreams.Load()),
		BytesUp:           e.bytesUp.Load(),
		BytesDown:         e.bytesDown.Load(),
		LastError:         lastErr,
		KillSwitchEngaged: killSwitchEngaged,
	}
}

// Events returns the channel Engine pushes state changes and stats ticks to.
// The same channel is reused across Start/Stop cycles.
func (e *Engine) Events() <-chan Event {
	return e.events
}

// setState updates the current state and emits an EventStateChanged. The
// send blocks until delivered or ctx is done, so state transitions are never
// silently dropped.
func (e *Engine) setState(ctx context.Context, state State, errMsg string) {
	e.stateMu.Lock()
	e.state = state
	e.lastErr = errMsg
	if state == StateConnected {
		e.connectedAt = time.Now()
	}
	e.stateMu.Unlock()

	e.onKillSwitchStateChanged(state)

	select {
	case e.events <- Event{Type: EventStateChanged, State: state, Err: errMsg}:
	case <-ctx.Done():
	}
}

// emitStatsTick pushes a throughput sample on a best-effort basis: losing an
// occasional tick under backpressure is harmless.
func (e *Engine) emitStatsTick(ev Event) {
	select {
	case e.events <- ev:
	default:
	}
}
