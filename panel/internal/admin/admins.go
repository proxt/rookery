package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/store"
)

type adminView struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
	IsYou     bool   `json:"is_you"`
}

func (s *Server) handleListAdmins(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListAdmins()
	if err != nil {
		slog.Error("admin: list admins", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	self := adminIDFrom(r.Context())
	views := make([]adminView, 0, len(list))
	for _, a := range list {
		views = append(views, adminView{ID: a.ID, Username: a.Username, CreatedAt: a.CreatedAt.Format(time.RFC3339), IsYou: a.ID == self})
	}
	writeJSON(w, views)
}

func (s *Server) handleCreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || len(req.Password) < 8 {
		http.Error(w, "username and a password of at least 8 characters are required", http.StatusBadRequest)
		return
	}

	a, err := s.store.CreateAdmin(req.Username, req.Password)
	if err != nil {
		if err == store.ErrAdminExists {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}
		slog.Error("admin: create admin", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, adminView{ID: a.ID, Username: a.Username, CreatedAt: a.CreatedAt.Format(time.RFC3339)})
}

func (s *Server) handleDeleteAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == adminIDFrom(r.Context()) {
		http.Error(w, "cannot delete your own account", http.StatusBadRequest)
		return
	}

	count, err := s.store.AdminCount()
	if err != nil {
		slog.Error("admin: count admins", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if count <= 1 {
		http.Error(w, "cannot delete the last remaining admin", http.StatusBadRequest)
		return
	}

	if err := s.store.DeleteAdmin(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
