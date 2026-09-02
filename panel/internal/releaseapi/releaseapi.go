// Package releaseapi implements the panel's public release endpoints: the
// client's update check, and a direct download link (also what the admin
// panel's own "Скачать приложение" button uses).
package releaseapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rookery/panel/internal/store"
)

// Server implements the public release HTTP handlers.
type Server struct {
	store *store.Store
}

// NewServer builds a releaseapi Server backed by st.
func NewServer(st *store.Store) *Server {
	return &Server{store: st}
}

// RegisterRoutes wires the public routes onto mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /releases/latest", s.handleLatest)
	mux.HandleFunc("GET /releases/{id}/download", s.handleDownload)
}

type latestView struct {
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) {
	rel, err := s.store.LatestRelease()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("releaseapi: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latestView{
		Version:   rel.Version,
		Notes:     rel.Notes,
		Size:      rel.Size,
		CreatedAt: rel.CreatedAt.Format(time.RFC3339),
		URL:       strings.TrimSuffix(publicAddr, "/") + "/releases/" + rel.ID + "/download",
	})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	rel, err := s.store.GetRelease(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+rel.Filename+`"`)
	http.ServeFile(w, r, rel.FilePath)
}
