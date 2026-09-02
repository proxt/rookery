package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"

	"github.com/rookery/shared/signaling"
	"github.com/rookery/shared/transport"
)

// gatherTimeout bounds how long we wait for ICE candidate gathering to
// finish before giving up on a connection attempt.
const gatherTimeout = 10 * time.Second

// dataChannelOpenTimeout bounds how long we wait for the DataChannel to open
// after the node has answered.
const dataChannelOpenTimeout = 30 * time.Second

// connectLoop keeps the tunnel connected: it calls connectOnce, and on any
// disconnect (other than ctx cancellation) retries with exponential backoff
// and jitter, capped at cfg.ReconnectMaxBackoff.
func (e *Engine) connectLoop(ctx context.Context, cfg Config) {
	backoff := time.Second

	for ctx.Err() == nil {
		e.setState(ctx, StateConnecting, "")

		err := e.connectOnce(ctx, cfg)

		e.sess.Store(nil)
		e.activePC.Store(nil)
		e.activeStreams.Store(0)

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			e.setState(ctx, StateError, err.Error())
		}

		wait := backoff/2 + time.Duration(rand.Int63n(int64(backoff)/2+1))
		if wait > cfg.ReconnectMaxBackoff {
			wait = cfg.ReconnectMaxBackoff
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return
		}

		backoff *= 2
		if backoff > cfg.ReconnectMaxBackoff {
			backoff = cfg.ReconnectMaxBackoff
		}
	}
}

// connectOnce performs a full offer/answer handshake with the node, brings
// up smux over the resulting DataChannel, and then blocks until the session
// ends. It returns nil if the session ended because ctx was canceled, and an
// error for any other failure (handshake failure or an unexpected drop).
func (e *Engine) connectOnce(ctx context.Context, cfg Config) error {
	api := transport.NewClientAPI()
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return fmt.Errorf("create peer connection: %w", err)
	}
	defer pc.Close()

	dc, err := pc.CreateDataChannel("rookery", nil)
	if err != nil {
		return fmt.Errorf("create datachannel: %w", err)
	}

	dcOpen := make(chan struct{})
	dc.OnOpen(func() { close(dcOpen) })

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("set local description: %w", err)
	}

	gatherCtx, cancelGather := context.WithTimeout(ctx, gatherTimeout)
	err = transport.WaitGatherComplete(gatherCtx, pc)
	cancelGather()
	if err != nil {
		return fmt.Errorf("ice gathering: %w", err)
	}

	answerSDP, err := e.exchangeSDP(ctx, cfg, pc.LocalDescription().SDP)
	if err != nil {
		return fmt.Errorf("signaling: %w", err)
	}

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: answerSDP}); err != nil {
		return fmt.Errorf("set remote description: %w", err)
	}

	openCtx, cancelOpen := context.WithTimeout(ctx, dataChannelOpenTimeout)
	defer cancelOpen()
	select {
	case <-dcOpen:
	case <-openCtx.Done():
		return fmt.Errorf("datachannel did not open: %w", openCtx.Err())
	}

	conn := transport.NewDataChannelConn(dc, cfg.BufferedAmountLowThreshold, cfg.BufferedAmountHighWaterMark)
	sess, err := transport.NewSmuxClient(conn)
	if err != nil {
		return fmt.Errorf("bring up smux: %w", err)
	}
	defer sess.Close()

	closeSignal := make(chan struct{})
	var once sync.Once
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateDisconnected:
			once.Do(func() { close(closeSignal) })
		}
	})

	e.sess.Store(sess)
	e.activePC.Store(pc)
	e.setState(ctx, StateConnected, "")

	select {
	case <-sess.CloseChan():
	case <-closeSignal:
	case <-ctx.Done():
	}

	if ctx.Err() != nil {
		return nil
	}
	return fmt.Errorf("connection lost")
}

// exchangeSDP POSTs offerSDP to the node's signaling endpoint and returns its
// answer SDP.
//
// TODO(phase 2): the node now authenticates SessionRequest.Token — a
// panel-issued, node-scoped token fetched from the panel's /sub/{token}
// endpoint — instead of a per-user secret the client holds directly. cfg.
// Secret is passed through as a placeholder so this still compiles; it will
// not authenticate against a panel-backed node until the client is reworked
// around subscriptions (see the plan's Phase 2).
func (e *Engine) exchangeSDP(ctx context.Context, cfg Config, offerSDP string) (string, error) {
	reqBody, err := json.Marshal(signaling.SessionRequest{SDP: offerSDP, Token: cfg.Secret})
	if err != nil {
		return "", fmt.Errorf("marshal offer: %w", err)
	}

	url := strings.TrimSuffix(cfg.NodeAddr, "/") + "/session"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("node returned status %d", resp.StatusCode)
	}

	var sessResp signaling.SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return sessResp.SDP, nil
}
