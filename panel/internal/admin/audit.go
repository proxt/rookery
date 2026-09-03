package admin

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// logAudit records one admin action, best-effort — a failure here is logged
// but never fails the request it's auditing; the mutation the caller just
// performed already succeeded and audit logging shouldn't be able to undo
// that from the caller's perspective.
func (s *Server) logAudit(adminID, action, targetType, targetID, detail string) {
	a, err := s.store.GetAdmin(adminID)
	if err != nil {
		slog.Error("admin: audit log admin lookup", "admin_id", adminID, "error", err)
		return
	}
	if err := s.store.CreateAuditEntry(adminID, a.Username, action, targetType, targetID, detail); err != nil {
		slog.Error("admin: create audit entry", "error", err)
	}
}

type auditEntryView struct {
	ID         string `json:"id"`
	AdminName  string `json:"admin_name"`
	Action     string `json:"action"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Detail     string `json:"detail"`
	CreatedAt  string `json:"created_at"`
}

func (s *Server) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	list, err := s.store.ListAuditLog(limit, offset)
	if err != nil {
		slog.Error("admin: list audit log", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	views := make([]auditEntryView, 0, len(list))
	for _, e := range list {
		views = append(views, auditEntryView{
			ID:         e.ID,
			AdminName:  e.AdminName,
			Action:     e.Action,
			TargetType: e.TargetType,
			TargetID:   e.TargetID,
			Detail:     e.Detail,
			CreatedAt:  e.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, views)
}
