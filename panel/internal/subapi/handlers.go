// Package subapi implements the panel's single client-facing endpoint:
// resolving a subscription token into the list of nodes it grants access to,
// each with a freshly signed, node-scoped session token.
package subapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/store"
	"github.com/rookery/shared/signaling"
)

// Server implements the client-facing HTTP handlers.
type Server struct {
	store    *store.Store
	tokenTTL time.Duration
}

// NewServer builds a subapi Server backed by st. Session tokens it issues
// are valid for tokenTTL.
func NewServer(st *store.Store, tokenTTL time.Duration) *Server {
	return &Server{store: st, tokenTTL: tokenTTL}
}

// RegisterRoutes wires the public routes onto mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sub/{token}", s.handleSub)
}

type nodeOut struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Tags         string `json:"tags"`
	Address      string `json:"address"`
	SessionToken string `json:"session_token"`
}

type subOut struct {
	Name  string    `json:"name"`
	Nodes []nodeOut `json:"nodes"`
}

func (s *Server) handleSub(w http.ResponseWriter, r *http.Request) {
	sub, err := s.store.GetSubscriptionByToken(r.PathValue("token"))
	if err != nil || !sub.Enabled || subscriptionExpired(sub) {
		http.NotFound(w, r)
		return
	}

	nodes, err := s.store.ListSubscriptionNodes(sub.ID)
	if err != nil {
		slog.Error("subapi: list subscription nodes", "subscription_id", sub.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	out := subOut{Name: sub.Name, Nodes: make([]nodeOut, 0, len(nodes))}
	for _, n := range nodes {
		claims := signaling.Claims{
			SubscriptionID: sub.ID,
			NodeID:         n.ID,
			Expiry:         time.Now().Add(s.tokenTTL).Unix(),
		}
		token, err := signaling.IssueToken([]byte(n.APIKey), claims)
		if err != nil {
			slog.Error("subapi: issue session token", "node_id", n.ID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out.Nodes = append(out.Nodes, nodeOut{ID: n.ID, Name: n.Name, Tags: n.Tags, Address: n.Address, SessionToken: token})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func subscriptionExpired(sub store.Subscription) bool {
	if sub.ExpiresAt == "" {
		return false
	}
	exp, err := time.Parse(time.RFC3339Nano, sub.ExpiresAt)
	if err != nil {
		return false
	}
	return time.Now().After(exp)
}
