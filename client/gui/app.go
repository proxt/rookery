package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/rookery/client/internal/engine"
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

// Connect starts the tunnel using the currently active profile and general
// settings.
func (a *App) Connect() error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	active, ok := settings.ActiveProfile()
	if !ok {
		return fmt.Errorf("сначала добавьте и выберите профиль на вкладке «Профили»")
	}

	cfg := engine.Config{
		NodeAddr:                    active.NodeAddr,
		SOCKSAddr:                   fmt.Sprintf("127.0.0.1:%d", effectivePort(settings.SOCKSPort)),
		UserID:                      active.UserID,
		Secret:                      active.Secret,
		BufferedAmountLowThreshold:  defaultBufferedAmountLowKB * 1024,
		BufferedAmountHighWaterMark: defaultBufferedAmountHighKB * 1024,
		ReconnectMaxBackoff:         defaultReconnectMaxBackoffS * time.Second,
		SystemWide:                  settings.SystemWide,
	}
	return a.eng.Start(a.ctx, cfg)
}

// Disconnect stops the tunnel.
func (a *App) Disconnect() {
	a.eng.Stop()
}

// GetStatus returns a point-in-time snapshot; the frontend also receives
// live updates via the "tunnel:event" event.
func (a *App) GetStatus() engine.StatusSnapshot {
	return a.eng.Status()
}

// GetAppSettings reads all persisted settings (profiles + general options).
func (a *App) GetAppSettings() (AppSettings, error) {
	return loadSettings(a.settingsPath)
}

// SaveGeneralSettings updates the profile-independent settings (SOCKS port,
// autostart, start minimized, system-wide) without touching the profile list.
func (a *App) SaveGeneralSettings(socksPort int, autoStart, startMinimized, systemWide bool) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}

	settings.SOCKSPort = effectivePort(socksPort)
	settings.AutoStart = autoStart
	settings.StartMinimized = startMinimized
	settings.SystemWide = systemWide

	if err := saveSettings(a.settingsPath, settings); err != nil {
		return err
	}
	return setAutoStart(autoStart)
}

// AddProfileFromLink decodes a rookery://sub/... link and adds it as a new
// saved profile. If it's the first profile, it becomes the active one.
//
// TODO(phase 2): a link now points at a panel subscription (PanelAddr +
// Token), not a single node's address/user id/secret directly. This just
// keeps the app compiling with a placeholder mapping; the real Phase 2 work
// is fetching the node list from PanelAddr+"/sub/"+Token, letting the user
// pick one, and refreshing it periodically — see the plan's Phase 2.
func (a *App) AddProfileFromLink(link string) (Profile, error) {
	l, err := profile.Decode(link)
	if err != nil {
		return Profile{}, fmt.Errorf("некорректная ссылка: %w", err)
	}

	settings, err := a.GetAppSettings()
	if err != nil {
		return Profile{}, err
	}

	id, err := randomID()
	if err != nil {
		return Profile{}, err
	}

	p := Profile{ID: id, Name: "Без названия", NodeAddr: l.PanelAddr, UserID: "", Secret: l.Token}
	settings.Profiles = append(settings.Profiles, p)
	if settings.ActiveProfileID == "" {
		settings.ActiveProfileID = p.ID
	}

	if err := saveSettings(a.settingsPath, settings); err != nil {
		return Profile{}, err
	}
	return p, nil
}

// DeleteProfile removes a saved profile. If it was the active one, no
// profile is active afterward until the user picks another.
func (a *App) DeleteProfile(id string) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}

	kept := settings.Profiles[:0]
	for _, p := range settings.Profiles {
		if p.ID != id {
			kept = append(kept, p)
		}
	}
	settings.Profiles = kept

	if settings.ActiveProfileID == id {
		settings.ActiveProfileID = ""
	}

	return saveSettings(a.settingsPath, settings)
}

// SetActiveProfile switches which saved profile Connect uses.
func (a *App) SetActiveProfile(id string) error {
	settings, err := a.GetAppSettings()
	if err != nil {
		return err
	}
	settings.ActiveProfileID = id
	return saveSettings(a.settingsPath, settings)
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
