package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/store"
)

type userView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func toUserView(u store.User) userView {
	return userView{ID: u.ID, Name: u.Name, CreatedAt: u.CreatedAt.Format(time.RFC3339)}
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListUsers()
	if err != nil {
		slog.Error("admin: list users", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	views := make([]userView, 0, len(list))
	for _, u := range list {
		views = append(views, toUserView(u))
	}
	writeJSON(w, views)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	u, err := s.store.CreateUser(req.Name)
	if err != nil {
		slog.Error("admin: create user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, toUserView(u))
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteUser(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
