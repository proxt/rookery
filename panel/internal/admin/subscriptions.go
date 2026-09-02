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

type subscriptionView struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id"`
	Name      string        `json:"name"`
	SubURL    string        `json:"sub_url"`
	Enabled   bool          `json:"enabled"`
	ExpiresAt string        `json:"expires_at"`
	CreatedAt string        `json:"created_at"`
	Nodes     []nodeSummary `json:"nodes"`
}

func (s *Server) toSubscriptionView(sub store.Subscription, publicAddr string) (subscriptionView, error) {
	nodes, err := s.store.ListSubscriptionNodes(sub.ID)
	if err != nil {
		return subscriptionView{}, err
	}
	summaries := make([]nodeSummary, 0, len(nodes))
	for _, n := range nodes {
		summaries = append(summaries, nodeSummary{ID: n.ID, Name: n.Name, Tags: n.Tags})
	}

	return subscriptionView{
		ID: sub.ID, UserID: sub.UserID, Name: sub.Name,
		SubURL:    strings.TrimSuffix(publicAddr, "/") + "/sub/" + sub.Token,
		Enabled:   sub.Enabled,
		ExpiresAt: sub.ExpiresAt,
		CreatedAt: sub.CreatedAt.Format(time.RFC3339),
		Nodes:     summaries,
	}, nil
}

func (s *Server) handleListSubscriptions(w http.ResponseWriter, r *http.Request) {
	var list []store.Subscription
	var err error
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		list, err = s.store.ListSubscriptionsByUser(userID)
	} else {
		list, err = s.store.ListSubscriptions()
	}
	if err != nil {
		slog.Error("admin: list subscriptions", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	views := make([]subscriptionView, 0, len(list))
	for _, sub := range list {
		v, err := s.toSubscriptionView(sub, publicAddr)
		if err != nil {
			slog.Error("admin: build subscription view", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		views = append(views, v)
	}
	writeJSON(w, views)
}

func (s *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" || req.Name == "" {
		http.Error(w, "user_id and name are required", http.StatusBadRequest)
		return
	}

	sub, err := s.store.CreateSubscription(req.UserID, req.Name)
	if err != nil {
		slog.Error("admin: create subscription", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	v, err := s.toSubscriptionView(sub, publicAddr)
	if err != nil {
		slog.Error("admin: build subscription view", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, v)
}

func (s *Server) handleUpdateSubscription(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name      string `json:"name"`
		Enabled   bool   `json:"enabled"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateSubscription(id, req.Name, req.Enabled, req.ExpiresAt); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteSubscription(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetSubscriptionNodes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		NodeIDs []string `json:"node_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := s.store.SetSubscriptionNodes(id, req.NodeIDs); err != nil {
		slog.Error("admin: set subscription nodes", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
