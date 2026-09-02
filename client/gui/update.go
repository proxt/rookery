package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// updateCheckTimeout bounds the /releases/latest request.
const updateCheckTimeout = 10 * time.Second

// UpdateInfo is what the frontend's update-check UI renders.
type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	Version        string `json:"version"`
	Notes          string `json:"notes"`
	DownloadURL    string `json:"downloadUrl"`
	Size           int64  `json:"size"`
}

// CheckForUpdate asks the active subscription's panel for its latest
// uploaded release and compares it against this build's version. There's
// no separate "update server" — whichever panel a subscription points at
// is also where its operator publishes client builds (Releases tab).
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	settings, err := a.GetAppSettings()
	if err != nil {
		return UpdateInfo{}, err
	}
	sub, ok := settings.ActiveSubscription()
	if !ok {
		return UpdateInfo{}, fmt.Errorf("добавьте и выберите подписку, чтобы проверить обновления")
	}

	ctx, cancel := context.WithTimeout(a.ctx, updateCheckTimeout)
	defer cancel()

	url := strings.TrimSuffix(sub.PanelAddr, "/") + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return UpdateInfo{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("не удалось связаться с панелью: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// The panel hasn't uploaded any release yet — not an error.
		return UpdateInfo{CurrentVersion: AppVersion}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("панель вернула статус %d", resp.StatusCode)
	}

	var rel struct {
		Version string `json:"version"`
		Notes   string `json:"notes"`
		Size    int64  `json:"size"`
		URL     string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return UpdateInfo{}, fmt.Errorf("не удалось разобрать ответ панели: %w", err)
	}

	return UpdateInfo{
		Available:      rel.Version != AppVersion,
		CurrentVersion: AppVersion,
		Version:        rel.Version,
		Notes:          rel.Notes,
		DownloadURL:    rel.URL,
		Size:           rel.Size,
	}, nil
}

// DownloadAndInstallUpdate downloads the installer at url to a temp file,
// launches it, and quits this app so the installer can overwrite its
// files. The installer inherits this process's elevation (the app's exe
// manifest already requires admin), so it doesn't need its own UAC prompt.
func (a *App) DownloadAndInstallUpdate(url string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("не удалось скачать обновление: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер вернул статус %d", resp.StatusCode)
	}

	dst := filepath.Join(os.TempDir(), "rookery-update-installer.exe")
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл: %w", err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return fmt.Errorf("не удалось сохранить обновление: %w", err)
	}
	out.Close()

	cmd := exec.Command(dst)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("не удалось запустить установщик: %w", err)
	}

	a.Quit()
	return nil
}
