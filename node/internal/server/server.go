// Package server runs the node's HTTP signaling endpoint: it authenticates
// SDP offers, answers them, and hands established peers off to the session
// package.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"

	"github.com/rookery/node/internal/session"
	"github.com/rookery/shared/signaling"
	"github.com/rookery/shared/transport"
)

// maxBodyBytes bounds how much of a request body we'll read before giving
// up; an SDP offer is a few KB at most.
const maxBodyBytes = 256 * 1024

// gatherTimeout bounds how long we wait for ICE candidate gathering to
// finish before answering.
const gatherTimeout = 10 * time.Second

// Config controls the signaling server and the sessions it spawns.
type Config struct {
	ListenAddr                  string
	Secret                      string
	ICEUDPPort                  int
	MaxStreamsPerSession        int
	DialTimeout                 time.Duration
	AllowPrivateNet             bool
	BufferedAmountLowThreshold  uint64
	BufferedAmountHighWaterMark uint64
	DataChannelOpenTimeout      time.Duration
}

// Server is the node's HTTP signaling endpoint.
type Server struct {
	cfg          Config
	secret       []byte
	api          *webrtc.API
	udpMuxCloser io.Closer
	httpSrv      *http.Server
	rootCtx      context.Context
	wg           sync.WaitGroup
}

// New builds a Server bound to cfg.ICEUDPPort. It does not start listening
// until Serve is called.
func New(cfg Config) (*Server, error) {
	api, udpMuxCloser, err := transport.NewNodeAPI(cfg.ICEUDPPort)
	if err != nil {
		return nil, fmt.Errorf("server: %w", err)
	}

	s := &Server{
		cfg:          cfg,
		secret:       []byte(cfg.Secret),
		api:          api,
		udpMuxCloser: udpMuxCloser,
		rootCtx:      context.Background(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /session", s.handleSession)
	s.httpSrv = &http.Server{Addr: cfg.ListenAddr, Handler: mux}

	return s, nil
}

// Serve starts the HTTP server and blocks until ctx is canceled, at which
// point it shuts the server down gracefully, waits for all spawned sessions
// to end, and releases the ICE UDP socket.
func (s *Server) Serve(ctx context.Context) error {
	s.rootCtx = ctx

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: listen: %w", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server: shutdown: %w", err)
		}
	}

	s.wg.Wait()
	return s.udpMuxCloser.Close()
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil || len(body) > maxBodyBytes {
		http.NotFound(w, r)
		return
	}

	if err := s.authenticate(r, body); err != nil {
		slog.Debug("server: rejected unauthenticated request", "error", err, "remote", r.RemoteAddr)
		http.NotFound(w, r)
		return
	}

	var req signaling.SessionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.NotFound(w, r)
		return
	}

	pc, err := s.api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		slog.Error("server: create peer connection", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dcReady := make(chan *webrtc.DataChannel, 1)
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		select {
		case dcReady <- dc:
		default:
		}
	})

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: req.SDP}); err != nil {
		slog.Warn("server: set remote description", "error", err)
		pc.Close()
		http.Error(w, "invalid offer", http.StatusBadRequest)
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		slog.Error("server: create answer", "error", err)
		pc.Close()
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		slog.Error("server: set local description", "error", err)
		pc.Close()
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	gatherCtx, cancel := context.WithTimeout(r.Context(), gatherTimeout)
	defer cancel()
	if err := transport.WaitGatherComplete(gatherCtx, pc); err != nil {
		slog.Warn("server: ice gathering", "error", err)
		pc.Close()
		http.Error(w, "ice gathering failed", http.StatusInternalServerError)
		return
	}

	respBody, err := json.Marshal(signaling.SessionResponse{SDP: pc.LocalDescription().SDP})
	if err != nil {
		slog.Error("server: marshal response", "error", err)
		pc.Close()
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)

	sessionCfg := session.Config{
		MaxStreams:                  s.cfg.MaxStreamsPerSession,
		DialTimeout:                 s.cfg.DialTimeout,
		AllowPrivateNet:             s.cfg.AllowPrivateNet,
		BufferedAmountLowThreshold:  s.cfg.BufferedAmountLowThreshold,
		BufferedAmountHighWaterMark: s.cfg.BufferedAmountHighWaterMark,
		DataChannelOpenTimeout:      s.cfg.DataChannelOpenTimeout,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		session.Run(s.rootCtx, pc, dcReady, sessionCfg)
	}()
}

// authenticate verifies the X-Signature/X-Timestamp headers against body.
func (s *Server) authenticate(r *http.Request, body []byte) error {
	sig := r.Header.Get(signaling.HeaderSignature)
	ts := r.Header.Get(signaling.HeaderTimestamp)
	return signaling.Verify(s.secret, ts, body, sig, time.Now())
}
