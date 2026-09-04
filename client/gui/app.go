package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/rookery/client/internal/engine"
	"github.com/rookery/client/internal/subscription"
	"github.com/rookery/shared/profile"
)

// Fixed defaults for knobs the settings UI doesn't expose.
const (
	defaultBufferedAmountLowKB  = 256
	defaultBufferedAmountHighKB = 1024
	defaultReconnectMaxBackoffS = 30
)

// App bridges the Svelte frontend to the engine. All engine access from the
// frontend goes through App's bound methods.
type App struct {
	ctx          context.Context
	eng          *engine.Engine
	settingsPath string
	quitting     bool
	setTrayState func(engine.State)

	// pendingLink is a rookery:// link this process was launched with
	// (os.Args[1]), stashed here because ctx isn't ready yet in main —
	// startup() picks it up once it is.
	pendingLink string
}

// NewApp creates an idle App. Call startup (invoked by Wails) before using it.
func NewApp() *App {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return &App{
		eng:          engine.New(),
		settingsPath: filepath.Join(dir, "Rookery", "settings.yaml"),
	}
}

// startup is called by Wails once the window's JS runtime is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.forwardEvents()

	settings, err := a.GetAppSettings()
	if err == nil && settings.StartMinimized {
		runtime.WindowHide(a.ctx)
	}

	if a.pendingLink != "" {
		link := a.pendingLink
		a.pendingLink = ""
		go a.HandleExternalLink(link)
	}
	go startLinkListener(a)

	if err == nil && settings.SubRefreshOnLaunch {
		go a.refreshAllSubscriptions()
	}
	go a.autoRefreshSubscriptionsLoop()
}

// subAutoRefreshCheckInterval is how often the background loop wakes up to
// check whether a refresh is due — independent of the user-configured
// SubAutoRefreshMinutes, so toggling that setting takes effect within one
// tick instead of needing a restart.
const subAutoRefreshCheckInterval = time.Minute

// autoRefreshSubscriptionsLoop re-fetches every saved subscription's node
// list every SubAutoRefreshMinutes, for as long as the app runs. Reads the
// setting fresh each tick rather than once at startup.
func (a *App) autoRefreshSubscriptionsLoop() {
	ticker := time.NewTicker(subAutoRefreshCheckInterval)
	defer ticker.Stop()

	var lastRefresh time.Time
	for range ticker.C {
		settings, err := a.GetAppSettings()
		if err != nil || settings.SubAutoRefreshMinutes <= 0 {
			continue
		}
		interval := time.Duration(settings.SubAutoRefreshMinutes) * time.Minute
		if time.Since(lastRefresh) < interval {
			continue
		}
		lastRefresh = time.Now()
		a.refreshAllSubscriptions()
	}
}

// refreshAllSubscriptions re-fetches every saved subscription's node list.
// Best-effort: one subscription's panel being unreachable doesn't stop the
// others, and failures are only logged, never surfaced (there's no
// synchronous caller waiting on this — see HandleExternalLink for the same
// convention).
func (a *App) refreshAllSubscriptions() {
	settings, err := a.GetAppSettings()
	if err != nil {
		return
	}
	for _, sub := range settings.Subscriptions {
		if _, err := a.RefreshSubscription(sub.ID); err != nil {
			slog.Warn("app: auto-refresh subscription", "subscription", sub.Name, "error", err)
		}
	}
	if a.ctx != nil {
		// Not "subscription:added" — that event also switches the frontend
		// to the Subscriptions tab, which would yank the user there every
		// time a background auto-refresh fires. This one just says "reload
		// whatever you have cached", no navigation side effect.
		runtime.EventsEmit(a.ctx, "settings:updated")
	}
}

// forwardEvents relays engine.Events() to the frontend as "tunnel:event",
// and keeps the tray icon in sync with connection state changes.
func (a *App) forwardEvents() {
	for ev := range a.eng.Events() {
		runtime.EventsEmit(a.ctx, "tunnel:event", ev)
		if ev.Type == engine.EventStateChanged && a.setTrayState != nil {
			a.setTrayState(ev.State)
		}
	}
}

// Connect starts the tunnel using the currently active subscription's
// currently active node, and general settings. It re-fetches the
// subscription first so the node's address (and the node list shown in the
// UI) is current before dialing.
func (a *App) Connect() error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	sub, ok := settings.ActiveSubscription()
	if !ok {
		return fmt.Errorf("сначала добавьте и выберите подписку на вкладке «Подписки»")
	}

	fetched, err := subscription.Fetch(a.ctx, sub.PanelAddr, sub.Token)
	if err != nil {
		return fmt.Errorf("не удалось получить список серверов подписки: %w", err)
	}
	if err := a.updateCachedNodes(sub.ID, fetched); err != nil {
		return err
	}

	nodeID := sub.ActiveNodeID
	node, ok := fetched.FindNode(nodeID)
	if !ok {
		if len(fetched.Nodes) == 0 {
			return fmt.Errorf("в подписке нет доступных серверов")
		}
		node = fetched.Nodes[0]
	}

	panelAddr, token := sub.PanelAddr, sub.Token
	cfg := engine.Config{
		NodeAddr:  node.Address,
		SOCKSAddr: fmt.Sprintf("127.0.0.1:%d", effectivePort(settings.SOCKSPort)),
		TokenFunc: func(ctx context.Context) (string, error) {
			s, err := subscription.Fetch(ctx, panelAddr, token)
			if err != nil {
				return "", err
			}
			n, ok := s.FindNode(node.ID)
			if !ok {
				return "", fmt.Errorf("сервер %q больше не входит в подписку", node.Name)
			}
			return n.SessionToken, nil
		},
		BufferedAmountLowThreshold:  defaultBufferedAmountLowKB * 1024,
		BufferedAmountHighWaterMark: defaultBufferedAmountHighKB * 1024,
		ReconnectMaxBackoff:         defaultReconnectMaxBackoffS * time.Second,
		SystemWide:                  settings.SystemWide,
		KillSwitch:                  settings.KillSwitch,
	}
	return a.eng.Start(a.ctx, cfg)
}

// Disconnect stops the tunnel.
func (a *App) Disconnect() {
	a.eng.Stop()
}

// GetAppVersion returns this build's version (see version.go), for the
// About page — kept as a real accessor rather than letting the frontend
// hardcode its own copy that drifts from the actual build.
func (a *App) GetAppVersion() string {
	return AppVersion
}

// GetStatus returns a point-in-time snapshot; the frontend also receives
// live updates via the "tunnel:event" event.
func (a *App) GetStatus() engine.StatusSnapshot {
	return a.eng.Status()
}

// GetAppSettings reads all persisted settings (subscriptions + general
// options).
func (a *App) GetAppSettings() (AppSettings, error) {
	return loadSettings(a.settingsPath)
}

// SaveGeneralSettings updates the subscription-independent settings (SOCKS
// port, autostart, start minimized, system-wide, kill switch, subscription
// auto-refresh) without touching the subscription list.
func (a *App) SaveGeneralSettings(socksPort int, autoStart, startMinimized, systemWide, killSwitch bool, subAutoRefreshMinutes int, subRefreshOnLaunch bool) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}

	settings.SOCKSPort = effectivePort(socksPort)
	settings.AutoStart = autoStart
	settings.StartMinimized = startMinimized
	settings.SystemWide = systemWide
	settings.KillSwitch = killSwitch
	settings.SubAutoRefreshMinutes = subAutoRefreshMinutes
	settings.SubRefreshOnLaunch = subRefreshOnLaunch

	if err := saveSettings(a.settingsPath, settings); err != nil {
		return err
	}
	return setAutoStart(autoStart)
}

// SetSystemWideMode is a lightweight, dashboard-level counterpart to
// SaveGeneralSettings' systemWide param — for the mode dropdown next to the
// connect button (proxy vs. system-wide/TUN), which shouldn't need to know
// every other general setting just to flip one. Doesn't touch KillSwitch:
// it only ever takes effect when SystemWide is also true (see
// killswitch.go's armKillSwitch), so a stale killSwitch=true alongside
// systemWide=false is inert, not a bug.
func (a *App) SetSystemWideMode(enabled bool) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}
	settings.SystemWide = enabled
	return saveSettings(a.settingsPath, settings)
}

// AddSubscriptionFromLink decodes a rookery://sub/... link, fetches its
// current node list, and adds it as a new saved subscription. If it's the
// first subscription, it becomes the active one.
func (a *App) AddSubscriptionFromLink(link string) (Subscription, error) {
	l, err := profile.Decode(link)
	if err != nil {
		return Subscription{}, fmt.Errorf("некорректная ссылка: %w", err)
	}

	fetched, err := subscription.Fetch(a.ctx, l.PanelAddr, l.Token)
	if err != nil {
		return Subscription{}, fmt.Errorf("не удалось получить подписку: %w", err)
	}

	settings, err := a.GetAppSettings()
	if err != nil {
		return Subscription{}, err
	}

	name := fetched.Name
	if name == "" {
		name = "Без названия"
	}

	// The same link (e.g. re-clicking a panel's "Установить в приложение"
	// button, or forwarded via IPC from a second launch) refreshes the
	// existing subscription in place instead of piling up duplicates.
	for i := range settings.Subscriptions {
		existing := &settings.Subscriptions[i]
		if existing.PanelAddr == l.PanelAddr && existing.Token == l.Token {
			existing.Name = name
			existing.Nodes = cacheNodes(fetched.Nodes)
			if _, ok := existing.ActiveNode(); !ok && len(fetched.Nodes) > 0 {
				existing.ActiveNodeID = fetched.Nodes[0].ID
			}
			if err := saveSettings(a.settingsPath, settings); err != nil {
				return Subscription{}, err
			}
			return *existing, nil
		}
	}

	id, err := randomID()
	if err != nil {
		return Subscription{}, err
	}

	sub := Subscription{ID: id, Name: name, PanelAddr: l.PanelAddr, Token: l.Token, Nodes: cacheNodes(fetched.Nodes)}
	if len(fetched.Nodes) > 0 {
		sub.ActiveNodeID = fetched.Nodes[0].ID
	}

	settings.Subscriptions = append(settings.Subscriptions, sub)
	if settings.ActiveSubscriptionID == "" {
		settings.ActiveSubscriptionID = sub.ID
	}

	if err := saveSettings(a.settingsPath, settings); err != nil {
		return Subscription{}, err
	}
	return sub, nil
}

// RefreshSubscription re-fetches a subscription's node list from its panel
// and updates the local cache.
func (a *App) RefreshSubscription(id string) (Subscription, error) {
	settings, err := a.GetAppSettings()
	if err != nil {
		return Subscription{}, err
	}

	var target *Subscription
	for i := range settings.Subscriptions {
		if settings.Subscriptions[i].ID == id {
			target = &settings.Subscriptions[i]
			break
		}
	}
	if target == nil {
		return Subscription{}, fmt.Errorf("подписка не найдена")
	}

	fetched, err := subscription.Fetch(a.ctx, target.PanelAddr, target.Token)
	if err != nil {
		return Subscription{}, fmt.Errorf("не удалось обновить подписку: %w", err)
	}
	target.Nodes = cacheNodes(fetched.Nodes)
	if _, ok := target.ActiveNode(); !ok && len(fetched.Nodes) > 0 {
		target.ActiveNodeID = fetched.Nodes[0].ID
	}

	if err := saveSettings(a.settingsPath, settings); err != nil {
		return Subscription{}, err
	}
	return *target, nil
}

// updateCachedNodes is RefreshSubscription's persistence step, reused by
// Connect (which already has a freshly fetched subscription in hand).
func (a *App) updateCachedNodes(id string, fetched subscription.Subscription) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}
	for i := range settings.Subscriptions {
		if settings.Subscriptions[i].ID == id {
			settings.Subscriptions[i].Nodes = cacheNodes(fetched.Nodes)
			return saveSettings(a.settingsPath, settings)
		}
	}
	return nil
}

// MeasureNodeLatencies pings every cached node of the given subscription
// concurrently and returns round-trip time in milliseconds per node ID.
// Unreachable nodes get -1 rather than being omitted, so the frontend can
// tell "not measured yet" (key absent) from "measured, unreachable" (-1).
func (a *App) MeasureNodeLatencies(subscriptionID string) (map[string]int, error) {
	settings, err := a.GetAppSettings()
	if err != nil {
		return nil, err
	}

	var target *Subscription
	for i := range settings.Subscriptions {
		if settings.Subscriptions[i].ID == subscriptionID {
			target = &settings.Subscriptions[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("подписка не найдена")
	}

	results := make(map[string]int, len(target.Nodes))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, node := range target.Nodes {
		wg.Add(1)
		go func(id, address string) {
			defer wg.Done()
			ms := -1
			if d, err := subscription.MeasurePing(a.ctx, address); err == nil {
				ms = int(d.Milliseconds())
			}
			mu.Lock()
			results[id] = ms
			mu.Unlock()
		}(node.ID, node.Address)
	}
	wg.Wait()
	return results, nil
}

// DeleteSubscription removes a saved subscription. If it was the active
// one, no subscription is active afterward until the user picks another.
func (a *App) DeleteSubscription(id string) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}

	kept := settings.Subscriptions[:0]
	for _, sub := range settings.Subscriptions {
		if sub.ID != id {
			kept = append(kept, sub)
		}
	}
	settings.Subscriptions = kept

	if settings.ActiveSubscriptionID == id {
		settings.ActiveSubscriptionID = ""
	}

	return saveSettings(a.settingsPath, settings)
}

// SetActiveSubscription switches which saved subscription Connect uses.
func (a *App) SetActiveSubscription(id string) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}
	settings.ActiveSubscriptionID = id
	return saveSettings(a.settingsPath, settings)
}

// SetActiveNode switches which of a subscription's nodes Connect uses.
func (a *App) SetActiveNode(subscriptionID, nodeID string) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}
	for i := range settings.Subscriptions {
		if settings.Subscriptions[i].ID == subscriptionID {
			settings.Subscriptions[i].ActiveNodeID = nodeID
			return saveSettings(a.settingsPath, settings)
		}
	}
	return fmt.Errorf("подписка не найдена")
}

// HandleExternalLink adds link as a new subscription and brings the window
// to the foreground — called for a rookery:// link this process was
// launched with, or one forwarded from a second launch via the IPC
// listener (see ipc.go). Errors are logged rather than surfaced, since
// there's no synchronous caller waiting on a Wails-bound method here.
func (a *App) HandleExternalLink(link string) {
	if _, err := a.AddSubscriptionFromLink(link); err != nil {
		slog.Warn("app: add subscription from external link", "error", err)
	}
	runtime.EventsEmit(a.ctx, "subscription:added")
	runtime.WindowShow(a.ctx)
}

// Show brings the main window to the foreground.
func (a *App) Show() {
	runtime.WindowShow(a.ctx)
}

// OpenURL opens url in the system's default browser.
func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// Quit exits the application entirely, bypassing the minimize-to-tray
// behavior on window close.
func (a *App) Quit() {
	a.quitting = true
	a.eng.Stop()
	runtime.Quit(a.ctx)
}

func effectivePort(port int) int {
	if port == 0 {
		return defaultSOCKSPort
	}
	return port
}

func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
