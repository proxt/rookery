package main

import (
	"context"
	"embed"
	"os"
	"strings"

	"fyne.io/systray"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
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

	app := NewApp()
	app.pendingLink = link

	go systray.Run(func() { setupTray(app) }, func() {})

	err := wails.Run(&options.App{
		Title:     "Rookery",
		Width:     440,
		Height:    700,
		MinWidth:  380,
		MinHeight: 600,
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
