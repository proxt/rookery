package admin

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/rookery/panel/internal/store"
)

type totalsView struct {
	BytesUp   uint64 `json:"bytes_up"`
	BytesDown uint64 `json:"bytes_down"`
}

func toTotalsView(t store.Totals) totalsView {
	return totalsView{BytesUp: t.BytesUp, BytesDown: t.BytesDown}
}

type pointView struct {
	BucketHour string `json:"bucket_hour"`
	BytesUp    uint64 `json:"bytes_up"`
	BytesDown  uint64 `json:"bytes_down"`
}

func toSeriesView(points []store.TimeSeriesPoint) []pointView {
	out := make([]pointView, 0, len(points))
	for _, p := range points {
		out = append(out, pointView{BucketHour: p.BucketHour, BytesUp: p.BytesUp, BytesDown: p.BytesDown})
	}
	return out
}

type overviewView struct {
	UserCount   int        `json:"user_count"`
	NodeCount   int        `json:"node_count"`
	NodesOnline int        `json:"nodes_online"`
	AllTime     totalsView `json:"all_time"`
	Today       totalsView `json:"today"`
}

// onlineWindow is how recently a node must have heartbeated to count as
// online — a few missed 30s heartbeats' worth of slack.
const onlineWindow = 2 * time.Minute

// defaultSeriesHours/maxSeriesHours bound the ?hours= query param on the
// time-series endpoints.
const (
	defaultSeriesHours = 24
	maxSeriesHours     = 24 * 30
)

func (s *Server) handleStatsOverview(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		slog.Error("admin: stats overview: list users", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	nodes, err := s.store.ListNodes()
	if err != nil {
		slog.Error("admin: stats overview: list nodes", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	allTime, err := s.store.GlobalTotals()
	if err != nil {
		slog.Error("admin: stats overview: global totals", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	today, err := s.store.GlobalTotalsSince(time.Now().Add(-24 * time.Hour))
	if err != nil {
		slog.Error("admin: stats overview: today totals", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	online := 0
	cutoff := time.Now().Add(-onlineWindow)
	for _, n := range nodes {
		if n.LastSeenAt == "" {
			continue
		}
		seen, err := time.Parse(time.RFC3339Nano, n.LastSeenAt)
		if err == nil && seen.After(cutoff) {
			online++
		}
	}

	writeJSON(w, overviewView{
		UserCount:   len(users),
		NodeCount:   len(nodes),
		NodesOnline: online,
		AllTime:     toTotalsView(allTime),
		Today:       toTotalsView(today),
	})
}

func (s *Server) handleStatsTimeSeries(w http.ResponseWriter, r *http.Request) {
	points, err := s.store.GlobalTimeSeries(parseHours(r))
	if err != nil {
		slog.Error("admin: stats timeseries", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, toSeriesView(points))
}

func (s *Server) handleStatsUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.store.TotalsForUser(id)
	if err != nil {
		slog.Error("admin: stats user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, toTotalsView(t))
}

func (s *Server) handleStatsUserSeries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	points, err := s.store.UserTimeSeries(id, parseHours(r))
	if err != nil {
		slog.Error("admin: stats user timeseries", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, toSeriesView(points))
}

func (s *Server) handleStatsNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.store.TotalsForNode(id)
	if err != nil {
		slog.Error("admin: stats node", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, toTotalsView(t))
}

func parseHours(r *http.Request) int {
	hours, err := strconv.Atoi(r.URL.Query().Get("hours"))
	if err != nil || hours <= 0 {
		return defaultSeriesHours
	}
	if hours > maxSeriesHours {
		return maxSeriesHours
	}
	return hours
}
