package main

import (
	"context"
	"embed"
	"io"
	"log/slog"
	"os"
	"strings"

	"fyne.io/systray"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/rookery/client/internal/vpn"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Windows passes a clicked rookery://... link as argv[1] (registered in
	// the installer's project.nsi as this exe's URL protocol handler).
	var link string
	if len(os.Args) > 1 && strings.Contains(os.Args[1], "://") {
		link = os.Args[1]
	}

	if !acquireSingleInstanceLock() {
		// Another instance is already running — hand off the link (if any)
		// to it instead of opening a second window/tray icon on top of it.
		if link != "" {
			forwardLinkToRunningInstance(link)
		}
		os.Exit(0)
	}

	// The GUI has no console of its own for the default slog logger's
	// output to go to (unlike rookery-cli, which sets its own), so every
	// slog call in the process would otherwise be invisible. Duplicating
	// into a bounded ring buffer is what backs the Logs screen; stderr is
	// kept too for `wails dev`/attached-debugger runs.
	logs := newLogRing()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, logs), &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Best-effort safety net: if a previous run crashed while the kill
	// switch was engaged, its firewall rules would otherwise survive
	// indefinitely (there's no GUI running to remove them). Cleared
	// unconditionally before anything else starts.
	if err := vpn.CleanupStaleKillSwitchRules(context.Background()); err != nil {
		slog.Warn("main: cleanup stale kill switch rules", "error", err)
	}

	app := NewApp()
	app.pendingLink = link
	app.logs = logs

	go systray.Run(func() { setupTray(app) }, func() {})

	err := wails.Run(&options.App{
		Title:     "Rookery",
		Width:     1040,
		Height:    680,
		MinWidth:  860,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 21, A: 255},
		OnStartup:        app.startup,
		OnBeforeClose: func(ctx context.Context) bool {
			if app.quitting {
				return false
			}
			wailsruntime.WindowHide(ctx)
			return true
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
