// Package signaling defines the wire format and HMAC authentication used by
// the node's HTTP signaling endpoint.
package signaling

// SessionRequest is the body of POST /session: a client's SDP offer.
type SessionRequest struct {
	SDP string `json:"sdp"`
}

// SessionResponse is the node's reply: its SDP answer.
type SessionResponse struct {
	SDP string `json:"sdp"`
}
