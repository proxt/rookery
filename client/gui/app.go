package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/rookery/client/internal/config"
	"github.com/rookery/client/internal/engine"
	"github.com/rookery/shared/profile"
)

// defaultSOCKSPort is used whenever no SOCKS port has been configured yet.
const defaultSOCKSPort = 1080

// Fixed defaults for knobs the settings panel doesn't expose.
const (
	defaultBufferedAmountLowKB  = 256
	defaultBufferedAmountHighKB = 1024
	defaultReconnectMaxBackoffS = 30
)

// Settings is the subset of the client config the GUI settings panel edits.
type Settings struct {
	ProfileName    string `json:"profileName"`
	NodeAddr       string `json:"nodeAddr"`
	SOCKSPort      int    `json:"socksPort"`
	UserID         string `json:"userId"`
	Secret         string `json:"secret"`
	AutoStart      bool   `json:"autoStart"`
	StartMinimized bool   `json:"startMinimized"`
}

// App bridges the Svelte frontend to the engine. All engine access from the
// frontend goes through App's bound methods.
type App struct {
	ctx        context.Context
	eng        *engine.Engine
	configPath string
	quitting   bool
	setTrayState func(engine.State)
}

// NewApp creates an idle App. Call startup (invoked by Wails) before using it.
func NewApp() *App {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return &App{
		eng:        engine.New(),
		configPath: filepath.Join(dir, "Rookery", "client.yaml"),
	}
}

// startup is called by Wails once the window's JS runtime is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.forwardEvents()

	settings, err := a.GetSettings()
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

// Connect loads the saved settings and starts the tunnel.
func (a *App) Connect() error {
	settings, err := a.GetSettings()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	if settings.NodeAddr == "" || settings.UserID == "" || settings.Secret == "" {
		return fmt.Errorf("сначала добавьте профиль по ссылке rookery:// в настройках")
	}

	cfg := settingsToEngineConfig(settings)
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

// GetSettings reads the saved settings from disk. If no config file exists
// yet, it returns sane defaults instead of an error.
func (a *App) GetSettings() (Settings, error) {
	data, err := os.ReadFile(a.configPath)
	if os.IsNotExist(err) {
		return Settings{SOCKSPort: defaultSOCKSPort}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read config: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Settings{}, fmt.Errorf("parse config: %w", err)
	}

	return configToSettings(cfg), nil
}

// SaveSettings writes settings to disk and applies the autostart toggle.
func (a *App) SaveSettings(settings Settings) error {
	if err := os.MkdirAll(filepath.Dir(a.configPath), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfg := settingsToConfig(settings)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(a.configPath, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if err := setAutoStart(settings.AutoStart); err != nil {
		return fmt.Errorf("update autostart: %w", err)
	}
	return nil
}

// ParseLink decodes a rookery:// profile link (from the node's admin panel)
// into Settings, without saving it — the frontend previews it and the user
// still has to call SaveSettings to persist it.
func (a *App) ParseLink(link string) (Settings, error) {
	current, err := a.GetSettings()
	if err != nil {
		current = Settings{SOCKSPort: defaultSOCKSPort}
	}

	l, err := profile.Decode(link)
	if err != nil {
		return Settings{}, fmt.Errorf("некорректная ссылка: %w", err)
	}

	current.ProfileName = l.Name
	current.NodeAddr = l.NodeAddr
	current.UserID = l.UserID
	current.Secret = l.Secret
	return current, nil
}

// Show brings the main window to the foreground.
func (a *App) Show() {
	runtime.WindowShow(a.ctx)
}

// Quit exits the application entirely, bypassing the minimize-to-tray
// behavior on window close.
func (a *App) Quit() {
	a.quitting = true
	a.eng.Stop()
	runtime.Quit(a.ctx)
}

func configToSettings(cfg config.Config) Settings {
	port := defaultSOCKSPort
	if _, portStr, err := net.SplitHostPort(cfg.SOCKSAddr); err == nil {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}
	return Settings{
		ProfileName:    cfg.ProfileName,
		NodeAddr:       cfg.NodeAddr,
		SOCKSPort:      port,
		UserID:         cfg.UserID,
		Secret:         cfg.Secret,
		AutoStart:      cfg.AutoStart,
		StartMinimized: cfg.StartMinimized,
	}
}

func settingsToConfig(s Settings) config.Config {
	port := s.SOCKSPort
	if port == 0 {
		port = defaultSOCKSPort
	}
	return config.Config{
		ProfileName:          s.ProfileName,
		NodeAddr:             s.NodeAddr,
		SOCKSAddr:            fmt.Sprintf("127.0.0.1:%d", port),
		UserID:               s.UserID,
		Secret:               s.Secret,
		BufferedAmountLowKB:  defaultBufferedAmountLowKB,
		BufferedAmountHighKB: defaultBufferedAmountHighKB,
		ReconnectMaxBackoffS: defaultReconnectMaxBackoffS,
		StartMinimized:       s.StartMinimized,
		AutoStart:            s.AutoStart,
		LogLevel:             "info",
	}
}

func settingsToEngineConfig(s Settings) engine.Config {
	cfg := settingsToConfig(s)
	return engine.Config{
		NodeAddr:                    cfg.NodeAddr,
		SOCKSAddr:                   cfg.SOCKSAddr,
		UserID:                      cfg.UserID,
		Secret:                      cfg.Secret,
		BufferedAmountLowThreshold:  uint64(cfg.BufferedAmountLowKB) * 1024,
		BufferedAmountHighWaterMark: uint64(cfg.BufferedAmountHighKB) * 1024,
		ReconnectMaxBackoff:         time.Duration(cfg.ReconnectMaxBackoffS) * time.Second,
	}
}
