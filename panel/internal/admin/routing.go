package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/store"
)

// randomRuleID generates an ID for a rule that arrived from the admin UI
// without one (a newly added rule in the form). Rules live inside a
// JSON blob, not their own table, so this doesn't go through the store
// package's randomToken.
func randomRuleID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("admin: generate rule id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type routingRuleView struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Value  string `json:"value"`
	Action string `json:"action"`
}

type routingRuleSetView struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Rules     []routingRuleView `json:"rules"`
	CreatedAt string            `json:"created_at"`
}

func toRoutingRuleSetView(rs store.RoutingRuleSet) routingRuleSetView {
	rules := make([]routingRuleView, 0, len(rs.Rules))
	for _, r := range rs.Rules {
		rules = append(rules, routingRuleView{ID: r.ID, Type: r.Type, Value: r.Value, Action: r.Action})
	}
	return routingRuleSetView{ID: rs.ID, Name: rs.Name, Rules: rules, CreatedAt: rs.CreatedAt.Format(time.RFC3339)}
}

func (s *Server) handleListRoutingRuleSets(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListRoutingRuleSets()
	if err != nil {
		slog.Error("admin: list routing rule sets", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	views := make([]routingRuleSetView, 0, len(list))
	for _, rs := range list {
		views = append(views, toRoutingRuleSetView(rs))
	}
	writeJSON(w, views)
}

func (s *Server) handleCreateRoutingRuleSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	rs, err := s.store.CreateRoutingRuleSet(req.Name)
	if err != nil {
		slog.Error("admin: create routing rule set", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.logAudit(adminIDFrom(r.Context()), "routing.create", "routing_rule_set", rs.ID, rs.Name)
	writeJSON(w, toRoutingRuleSetView(rs))
}

func (s *Server) handleUpdateRoutingRuleSet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name  string            `json:"name"`
		Rules []routingRuleView `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	rules := make([]store.RoutingRule, 0, len(req.Rules))
	for _, rv := range req.Rules {
		if rv.Type != "domain" && rv.Type != "app" && rv.Type != "geoip" {
			http.Error(w, "invalid rule type", http.StatusBadRequest)
			return
		}
		if rv.Action != "direct" && rv.Action != "proxy" {
			http.Error(w, "invalid rule action", http.StatusBadRequest)
			return
		}
		if rv.Value == "" {
			continue
		}
		id := rv.ID
		if id == "" {
			var err error
			id, err = randomRuleID()
			if err != nil {
				slog.Error("admin: generate rule id", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
		rules = append(rules, store.RoutingRule{ID: id, Type: rv.Type, Value: rv.Value, Action: rv.Action})
	}

	if err := s.store.UpdateRoutingRuleSet(id, req.Name, rules); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	s.logAudit(adminIDFrom(r.Context()), "routing.update", "routing_rule_set", id, req.Name)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteRoutingRuleSet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteRoutingRuleSet(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	s.logAudit(adminIDFrom(r.Context()), "routing.delete", "routing_rule_set", id, "")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetUserRoutingRuleSet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		RoutingRuleSetID string `json:"routing_rule_set_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.store.SetUserRoutingRuleSet(id, req.RoutingRuleSetID); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	s.logAudit(adminIDFrom(r.Context()), "user.routing.update", "user", id, req.RoutingRuleSetID)
	w.WriteHeader(http.StatusNoContent)
}
