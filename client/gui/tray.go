package main

import (
	_ "embed"

	"fyne.io/systray"

	"github.com/rookery/client/internal/engine"
)

//go:embed assets/tray/disconnected.ico
var iconDisconnected []byte

//go:embed assets/tray/connecting.ico
var iconConnecting []byte

//go:embed assets/tray/connected.ico
var iconConnected []byte

//go:embed assets/tray/error.ico
var iconError []byte

// setupTray wires the systray icon and menu (Connect/Disconnect, Open, Quit)
// to app, and returns a state callback for app to drive the icon.
func setupTray(app *App) {
	systray.SetIcon(iconDisconnected)
	systray.SetTitle("Rookery")
	systray.SetTooltip("Rookery — отключено")

	mConnect := systray.AddMenuItem("Подключить", "Запустить туннель")
	mDisconnect := systray.AddMenuItem("Отключить", "Остановить туннель")
	mDisconnect.Disable()
	systray.AddSeparator()
	mOpen := systray.AddMenuItem("Открыть", "Показать главное окно")
	mQuit := systray.AddMenuItem("Выход", "Закрыть Rookery")

	app.setTrayState = func(state engine.State) {
		switch state {
		case engine.StateDisconnected:
			systray.SetIcon(iconDisconnected)
			systray.SetTooltip("Rookery — отключено")
			mConnect.Enable()
			mDisconnect.Disable()
		case engine.StateConnecting:
			systray.SetIcon(iconConnecting)
			systray.SetTooltip("Rookery — подключение")
			mConnect.Disable()
			mDisconnect.Enable()
		case engine.StateConnected:
			systray.SetIcon(iconConnected)
			systray.SetTooltip("Rookery — подключено")
			mConnect.Disable()
			mDisconnect.Enable()
		case engine.StateError:
			systray.SetIcon(iconError)
			systray.SetTooltip("Rookery — ошибка")
			mConnect.Enable()
			mDisconnect.Disable()
		}
	}

	systray.SetOnTapped(app.Show)

	go func() {
		for {
			select {
			case <-mConnect.ClickedCh:
				app.Connect()
			case <-mDisconnect.ClickedCh:
				app.Disconnect()
			case <-mOpen.ClickedCh:
				app.Show()
			case <-mQuit.ClickedCh:
				app.Quit()
				return
			}
		}
	}()
}
