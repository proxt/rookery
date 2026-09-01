// Package transport holds the WebRTC/smux plumbing shared by the node and
// the client: PeerConnection setup, the DataChannel-to-net.Conn adapter, and
// bringing up smux over it.
package transport

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
)

// NewClientAPI builds a WebRTC API with default settings: ephemeral local
// ports, no STUN/TURN servers.
func NewClientAPI() *webrtc.API {
	return webrtc.NewAPI(webrtc.WithSettingEngine(webrtc.SettingEngine{}))
}

// NewNodeAPI builds a WebRTC API whose ICE traffic is bound to a single,
// fixed UDP port so the node's firewall rules stay deterministic. The
// returned io.Closer must be closed on shutdown to release the UDP socket.
func NewNodeAPI(udpPort int) (*webrtc.API, io.Closer, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: udpPort})
	if err != nil {
		return nil, nil, fmt.Errorf("transport: listen udp %d: %w", udpPort, err)
	}

	mux := ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: conn})

	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetICEUDPMux(mux)

	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))
	return api, mux, nil
}

// WaitGatherComplete blocks until pc has finished gathering ICE candidates,
// or ctx is done. Rookery does not use trickle ICE: the full candidate set
// must be gathered before the local description is sent to the peer.
func WaitGatherComplete(ctx context.Context, pc *webrtc.PeerConnection) error {
	select {
	case <-webrtc.GatheringCompletePromise(pc):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("transport: wait ice gathering complete: %w", ctx.Err())
	}
}
