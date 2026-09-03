// Package admin implements the panel's web admin panel: login, user
// (subscription) management, node registration, traffic statistics, admin
// account management, and client release hosting.
package admin

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/buildinfo"
	"github.com/rookery/panel/internal/store"
)

//go:embed all:frontend/dist
var uiFS embed.FS

const sessionCookieName = "rookery_admin_session"

// Server implements the admin panel's HTTP handlers.
type Server struct {
	store       *store.Store
	sessions    *sessionStore
	releasesDir string
}

// NewServer builds an admin Server backed by st. Uploaded client releases
// are saved under releasesDir.
func NewServer(st *store.Store, releasesDir string) *Server {
	return &Server{store: st, sessions: newSessionStore(), releasesDir: releasesDir}
}

// RegisterRoutes wires the admin panel's routes (API and static UI) onto mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /admin/api/login", s.handleLogin)
	mux.HandleFunc("POST /admin/api/logout", s.requireAuth(s.handleLogout))
	mux.HandleFunc("GET /admin/api/session", s.requireAuth(s.handleSessionCheck))
	mux.HandleFunc("GET /admin/api/version", s.requireAuth(s.handleVersion))
	mux.HandleFunc("GET /admin/api/settings", s.requireAuth(s.handleGetSettings))
	mux.HandleFunc("PUT /admin/api/settings", s.requireAuth(s.handleUpdateSettings))
	mux.HandleFunc("PUT /admin/api/account/password", s.requireAuth(s.handleChangeOwnPassword))

	mux.HandleFunc("GET /admin/api/admins", s.requireAuth(s.handleListAdmins))
	mux.HandleFunc("POST /admin/api/admins", s.requireAuth(s.handleCreateAdmin))
	mux.HandleFunc("DELETE /admin/api/admins/{id}", s.requireAuth(s.handleDeleteAdmin))

	mux.HandleFunc("GET /admin/api/users", s.requireAuth(s.handleListUsers))
	mux.HandleFunc("POST /admin/api/users", s.requireAuth(s.handleCreateUser))
	mux.HandleFunc("GET /admin/api/users/{id}", s.requireAuth(s.handleGetUser))
	mux.HandleFunc("PUT /admin/api/users/{id}", s.requireAuth(s.handleUpdateUser))
	mux.HandleFunc("DELETE /admin/api/users/{id}", s.requireAuth(s.handleDeleteUser))
	mux.HandleFunc("PUT /admin/api/users/{id}/nodes", s.requireAuth(s.handleSetUserNodes))

	mux.HandleFunc("GET /admin/api/nodes", s.requireAuth(s.handleListNodes))
	mux.HandleFunc("POST /admin/api/nodes", s.requireAuth(s.handleCreateNode))
	mux.HandleFunc("PATCH /admin/api/nodes/{id}", s.requireAuth(s.handleUpdateNode))
	mux.HandleFunc("DELETE /admin/api/nodes/{id}", s.requireAuth(s.handleDeleteNode))

	mux.HandleFunc("GET /admin/api/stats/overview", s.requireAuth(s.handleStatsOverview))
	mux.HandleFunc("GET /admin/api/stats/timeseries", s.requireAuth(s.handleStatsTimeSeries))
	mux.HandleFunc("GET /admin/api/stats/users/{id}", s.requireAuth(s.handleStatsUser))
	mux.HandleFunc("GET /admin/api/stats/users/{id}/series", s.requireAuth(s.handleStatsUserSeries))
	mux.HandleFunc("GET /admin/api/stats/nodes/{id}", s.requireAuth(s.handleStatsNode))

	mux.HandleFunc("GET /admin/api/releases", s.requireAuth(s.handleListReleases))
	mux.HandleFunc("POST /admin/api/releases", s.requireAuth(s.handleUploadRelease))
	mux.HandleFunc("DELETE /admin/api/releases/{id}", s.requireAuth(s.handleDeleteRelease))

	mux.HandleFunc("GET /admin/api/audit-log", s.requireAuth(s.handleListAuditLog))

	uiSub, err := fs.Sub(uiFS, "frontend/dist")
	if err != nil {
		panic(err) // embedded at build time; cannot fail at runtime
	}
	mux.Handle("GET /admin/", http.StripPrefix("/admin/", http.FileServerFS(uiSub)))
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		adminID, ok := s.sessions.get(cookie.Value)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(withAdminID(r.Context(), adminID)))
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	adminID, ok := s.store.VerifyAdmin(req.Username, req.Password)
	if !ok {
		time.Sleep(300 * time.Millisecond) // slow down credential guessing
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := s.sessions.create(adminID)
	if err != nil {
		slog.Error("admin: create session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/admin",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(sessionTTL),
	})
	s.logAudit(adminID, "auth.login", "admin", adminID, "")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.revoke(cookie.Value)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSessionCheck(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetAdmin(adminIDFrom(r.Context()))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{"id": a.ID, "username": a.Username})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"build_time": buildinfo.BuildTime, "commit": buildinfo.Commit})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	autoUpdate, err := s.store.AutoUpdateEnabled()
	if err != nil {
		slog.Error("admin: get auto update enabled", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"public_addr": publicAddr, "auto_update_enabled": autoUpdate})
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicAddr        string `json:"public_addr"`
		AutoUpdateEnabled bool   `json:"auto_update_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.store.SetPublicAddr(req.PublicAddr); err != nil {
		slog.Error("admin: set public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.store.SetAutoUpdateEnabled(req.AutoUpdateEnabled); err != nil {
		slog.Error("admin: set auto update enabled", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.logAudit(adminIDFrom(r.Context()), "settings.update", "settings", "",
		fmt.Sprintf("public_addr=%s auto_update_enabled=%v", req.PublicAddr, req.AutoUpdateEnabled))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleChangeOwnPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewPassword == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	a, err := s.store.GetAdmin(adminIDFrom(r.Context()))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, ok := s.store.VerifyAdmin(a.Username, req.CurrentPassword); !ok {
		http.Error(w, "current password is incorrect", http.StatusForbidden)
		return
	}

	if err := s.store.UpdateAdminPassword(a.ID, req.NewPassword); err != nil {
		slog.Error("admin: update own password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.logAudit(a.ID, "account.password.change", "admin", a.ID, "")
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
