package admin

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rookery/panel/internal/store"
)

// maxReleaseUploadSize bounds an uploaded client build. Generous — a Wails
// installer with the WebView2 bootstrapper and wintun.dll bundled in is a
// few dozen MB at most, but there's no reason to pinch here.
const maxReleaseUploadSize = 500 << 20

type releaseView struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

func (s *Server) toReleaseView(rel store.Release, publicAddr string) releaseView {
	return releaseView{
		ID: rel.ID, Version: rel.Version, Notes: rel.Notes, Filename: rel.Filename, Size: rel.Size,
		CreatedAt: rel.CreatedAt.Format(time.RFC3339),
		URL:       strings.TrimSuffix(publicAddr, "/") + "/releases/" + rel.ID + "/download",
	}
}

func (s *Server) handleListReleases(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListReleases()
	if err != nil {
		slog.Error("admin: list releases", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	views := make([]releaseView, 0, len(list))
	for _, rel := range list {
		views = append(views, s.toReleaseView(rel, publicAddr))
	}
	writeJSON(w, views)
}

func (s *Server) handleUploadRelease(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReleaseUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "file too large or malformed upload", http.StatusBadRequest)
		return
	}

	version := r.FormValue("version")
	if version == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	notes := r.FormValue("notes")

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	id, err := randomToken(8)
	if err != nil {
		slog.Error("admin: generate release id", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dir := filepath.Join(s.releasesDir, id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		slog.Error("admin: create release dir", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	filename := filepath.Base(header.Filename)
	dst := filepath.Join(dir, filename)

	out, err := os.Create(dst)
	if err != nil {
		slog.Error("admin: create release file", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	size, err := io.Copy(out, file)
	out.Close()
	if err != nil {
		os.RemoveAll(dir)
		slog.Error("admin: write release file", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	rel, err := s.store.CreateRelease(id, version, notes, filename, dst, size)
	if err != nil {
		os.RemoveAll(dir)
		slog.Error("admin: create release", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, s.toReleaseView(rel, publicAddr))
}

func (s *Server) handleDeleteRelease(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rel, err := s.store.GetRelease(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := s.store.DeleteRelease(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := os.RemoveAll(filepath.Dir(rel.FilePath)); err != nil {
		slog.Warn("admin: remove release file", "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}
