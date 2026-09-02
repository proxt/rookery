package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rookery/panel/internal/store"
)

type nodeSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Tags string `json:"tags"`
}

type userView struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	SubURL       string        `json:"sub_url"`
	Enabled      bool          `json:"enabled"`
	StartsAt     string        `json:"starts_at"`
	ExpiresAt    string        `json:"expires_at"`
	LastActiveAt string        `json:"last_active_at"`
	CreatedAt    string        `json:"created_at"`
	Nodes        []nodeSummary `json:"nodes"`
}

func (s *Server) toUserView(u store.User, publicAddr string) (userView, error) {
	nodes, err := s.store.ListUserNodes(u.ID)
	if err != nil {
		return userView{}, err
	}
	summaries := make([]nodeSummary, 0, len(nodes))
	for _, n := range nodes {
		summaries = append(summaries, nodeSummary{ID: n.ID, Name: n.Name, Tags: n.Tags})
	}

	return userView{
		ID:           u.ID,
		Name:         u.Name,
		SubURL:       strings.TrimSuffix(publicAddr, "/") + "/sub/" + u.Token,
		Enabled:      u.Enabled,
		StartsAt:     u.StartsAt,
		ExpiresAt:    u.ExpiresAt,
		LastActiveAt: u.LastActiveAt,
		CreatedAt:    u.CreatedAt.Format(time.RFC3339),
		Nodes:        summaries,
	}, nil
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListUsers()
	if err != nil {
		slog.Error("admin: list users", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	views := make([]userView, 0, len(list))
	for _, u := range list {
		v, err := s.toUserView(u, publicAddr)
		if err != nil {
			slog.Error("admin: build user view", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		views = append(views, v)
	}
	writeJSON(w, views)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	u, err := s.store.GetUser(r.PathValue("id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	v, err := s.toUserView(u, publicAddr)
	if err != nil {
		slog.Error("admin: build user view", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, v)
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

	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	v, err := s.toUserView(u, publicAddr)
	if err != nil {
		slog.Error("admin: build user view", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, v)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name      string `json:"name"`
		Enabled   bool   `json:"enabled"`
		StartsAt  string `json:"starts_at"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateUser(id, req.Name, req.Enabled, req.StartsAt, req.ExpiresAt); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteUser(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetUserNodes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		NodeIDs []string `json:"node_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := s.store.SetUserNodes(id, req.NodeIDs); err != nil {
		slog.Error("admin: set user nodes", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
