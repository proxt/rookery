package engine

import (
	"context"
	"time"

	"github.com/pion/webrtc/v4"
)

// statsLoop periodically emits an EventStatsTick with bytes/s since the last
// tick, and refreshes RTT from the active PeerConnection's ICE stats.
func (e *Engine) statsLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastUp, lastDown uint64
	for {
		select {
		case <-ticker.C:
			up := e.bytesUp.Load()
			down := e.bytesDown.Load()
			e.emitStatsTick(Event{
				Type:            EventStatsTick,
				BytesUpPerSec:   up - lastUp,
				BytesDownPerSec: down - lastDown,
			})
			lastUp, lastDown = up, down

			if pc := e.activePC.Load(); pc != nil {
				if rtt, ok := currentRTT(pc); ok {
					e.rtt.Store(int64(rtt))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// currentRTT reads the round trip time of the nominated, succeeded ICE
// candidate pair from pc's stats.
func currentRTT(pc *webrtc.PeerConnection) (time.Duration, bool) {
	for _, s := range pc.GetStats() {
		pairStats, ok := s.(webrtc.ICECandidatePairStats)
		if !ok {
			continue
		}
		if pairStats.State == webrtc.StatsICECandidatePairStateSucceeded && pairStats.Nominated {
			return time.Duration(pairStats.CurrentRoundTripTime * float64(time.Second)), true
		}
	}
	return 0, false
}
