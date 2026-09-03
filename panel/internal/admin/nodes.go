package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/store"
)

type nodeView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	APIKey     string `json:"api_key"`
	Tags       string `json:"tags"`
	Enabled    bool   `json:"enabled"`
	LastSeenAt string `json:"last_seen_at"`
	CreatedAt  string `json:"created_at"`
}

func toNodeView(n store.Node) nodeView {
	return nodeView{
		ID: n.ID, Name: n.Name, Address: n.Address, APIKey: n.APIKey, Tags: n.Tags,
		Enabled: n.Enabled, LastSeenAt: n.LastSeenAt, CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListNodes()
	if err != nil {
		slog.Error("admin: list nodes", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	views := make([]nodeView, 0, len(list))
	for _, n := range list {
		views = append(views, toNodeView(n))
	}
	writeJSON(w, views)
}

func (s *Server) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Tags    string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Address == "" {
		http.Error(w, "name and address are required", http.StatusBadRequest)
		return
	}

	n, err := s.store.CreateNode(req.Name, req.Address, req.Tags)
	if err != nil {
		slog.Error("admin: create node", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.logAudit(adminIDFrom(r.Context()), "node.create", "node", n.ID, n.Name)
	writeJSON(w, toNodeView(n))
}

func (s *Server) handleUpdateNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Tags    string `json:"tags"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateNode(id, req.Name, req.Address, req.Tags, req.Enabled); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	s.logAudit(adminIDFrom(r.Context()), "node.update", "node", id, req.Name)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteNode(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	s.logAudit(adminIDFrom(r.Context()), "node.delete", "node", id, "")
	w.WriteHeader(http.StatusNoContent)
}
