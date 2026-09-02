// Package admin implements the panel's web admin panel: login, user
// (subscription) management, node registration, and traffic statistics.
package admin

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/rookery/panel/internal/store"
)

//go:embed all:frontend/dist
var uiFS embed.FS

const sessionCookieName = "rookery_admin_session"

// Server implements the admin panel's HTTP handlers.
type Server struct {
	store    *store.Store
	sessions *sessionStore
}

// NewServer builds an admin Server backed by st.
func NewServer(st *store.Store) *Server {
	return &Server{store: st, sessions: newSessionStore()}
}

// RegisterRoutes wires the admin panel's routes (API and static UI) onto mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /admin/api/login", s.handleLogin)
	mux.HandleFunc("POST /admin/api/logout", s.requireAuth(s.handleLogout))
	mux.HandleFunc("GET /admin/api/session", s.requireAuth(s.handleSessionCheck))
	mux.HandleFunc("GET /admin/api/settings", s.requireAuth(s.handleGetSettings))
	mux.HandleFunc("PUT /admin/api/settings", s.requireAuth(s.handleUpdateSettings))
	mux.HandleFunc("PUT /admin/api/credentials", s.requireAuth(s.handleUpdateCredentials))

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
		if err != nil || !s.sessions.valid(cookie.Value) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
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

	if !s.store.VerifyAdmin(req.Username, req.Password) {
		time.Sleep(300 * time.Millisecond) // slow down credential guessing
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := s.sessions.create()
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
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.revoke(cookie.Value)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSessionCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	publicAddr, err := s.store.PublicAddr()
	if err != nil {
		slog.Error("admin: get public addr", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	username, err := s.store.AdminUsername()
	if err != nil {
		slog.Error("admin: get admin username", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"public_addr": publicAddr, "admin_username": username})
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicAddr string `json:"public_addr"`
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
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewUsername     string `json:"new_username"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	currentUsername, err := s.store.AdminUsername()
	if err != nil {
		slog.Error("admin: get admin username", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !s.store.VerifyAdmin(currentUsername, req.CurrentPassword) {
		http.Error(w, "current password is incorrect", http.StatusForbidden)
		return
	}

	if err := s.store.UpdateAdmin(req.NewUsername, req.NewPassword); err != nil {
		slog.Error("admin: update credentials", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
