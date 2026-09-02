// Package subapi implements the panel's single client-facing endpoint:
// resolving a subscription token into the list of nodes it grants access to,
// each with a freshly signed, node-scoped session token.
package subapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
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
	u, err := s.store.GetUserByToken(r.PathValue("token"))
	if err != nil || !u.Enabled || !withinWindow(u) {
		if wantsHTML(r) {
			renderSubPage(w, subPageData{Found: false})
			return
		}
		http.NotFound(w, r)
		return
	}

	nodes, err := s.store.ListUserNodes(u.ID)
	if err != nil {
		slog.Error("subapi: list user nodes", "user_id", u.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	out := subOut{Name: u.Name, Nodes: make([]nodeOut, 0, len(nodes))}
	for _, n := range nodes {
		claims := signaling.Claims{
			UserID: u.ID,
			NodeID: n.ID,
			Expiry: time.Now().Add(s.tokenTTL).Unix(),
		}
		token, err := signaling.IssueToken([]byte(n.APIKey), claims)
		if err != nil {
			slog.Error("subapi: issue session token", "node_id", n.ID, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out.Nodes = append(out.Nodes, nodeOut{ID: n.ID, Name: n.Name, Tags: n.Tags, Address: n.Address, SessionToken: token})
	}

	if wantsHTML(r) {
		renderSubPage(w, subPageData{
			Found: true, Name: u.Name, ExpiresAt: u.ExpiresAt, NodeCount: len(out.Nodes), SubURL: r.URL.String(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// wantsHTML reports whether r looks like a human opening the link in a
// browser (no Accept: application/json) rather than the client app fetching
// it programmatically — the app always sets that header explicitly.
func wantsHTML(r *http.Request) bool {
	return !strings.Contains(r.Header.Get("Accept"), "application/json")
}

func withinWindow(u store.User) bool {
	now := time.Now()
	if u.StartsAt != "" {
		starts, err := time.Parse(time.RFC3339Nano, u.StartsAt)
		if err == nil && now.Before(starts) {
			return false
		}
	}
	if u.ExpiresAt != "" {
		expires, err := time.Parse(time.RFC3339Nano, u.ExpiresAt)
		if err == nil && now.After(expires) {
			return false
		}
	}
	return true
}
