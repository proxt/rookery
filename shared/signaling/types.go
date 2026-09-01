// Package signaling defines the wire format and HMAC authentication used by
// the node's HTTP signaling endpoint.
package signaling

// SessionRequest is the body of POST /session: a client's SDP offer, signed
// with the secret belonging to UserID.
type SessionRequest struct {
	UserID string `json:"user_id"`
	SDP    string `json:"sdp"`
}

// SessionResponse is the node's reply: its SDP answer.
type SessionResponse struct {
	SDP string `json:"sdp"`
}
