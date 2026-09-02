// Package nodeapi implements the panel's node-facing HTTP endpoints:
// heartbeat and traffic reporting, authenticated by each node's own API key.
package nodeapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/rookery/panel/internal/store"
)

// Server implements the node-facing HTTP handlers.
type Server struct {
	store *store.Store
}

// NewServer builds a nodeapi Server backed by st.
func NewServer(st *store.Store) *Server {
	return &Server{store: st}
}

// RegisterRoutes wires the node-facing routes onto mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/nodes/heartbeat", s.requireNodeAuth(s.handleHeartbeat))
	mux.HandleFunc("POST /api/nodes/report", s.requireNodeAuth(s.handleReport))
}

// requireNodeAuth checks X-Node-ID/X-Node-Key against the stored node,
// passing it to next.
func (s *Server) requireNodeAuth(next func(w http.ResponseWriter, r *http.Request, node store.Node)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		node, err := s.store.NodeByAPIKey(r.Header.Get("X-Node-ID"), r.Header.Get("X-Node-Key"))
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !node.Enabled {
			http.Error(w, "node disabled", http.StatusForbidden)
			return
		}
		next(w, r, node)
	}
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request, node store.Node) {
	if err := s.store.TouchNode(node.ID); err != nil {
		slog.Error("nodeapi: touch node", "node_id", node.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type reportEntry struct {
	UserID    string `json:"user_id"`
	BytesUp   uint64 `json:"bytes_up"`
	BytesDown uint64 `json:"bytes_down"`
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request, node store.Node) {
	var entries []reportEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	for _, e := range entries {
		if e.UserID == "" {
			continue
		}
		if err := s.store.RecordTraffic(e.UserID, node.ID, e.BytesUp, e.BytesDown); err != nil {
			slog.Error("nodeapi: record traffic", "node_id", node.ID, "user_id", e.UserID, "error", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
