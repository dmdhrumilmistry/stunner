// Package transport carries already-encrypted bytes between peers over WebRTC
// data channels.
//
// It uses pion/webrtc (planned, roadmap phase 4). ICE servers (STUN/TURN) come
// from pkg/settings and are passed straight into the WebRTC configuration, so
// users can override NAT-traversal infrastructure. TURN, when used, only relays
// E2E ciphertext and cannot read content.
//
// This file defines the stable interfaces; the concrete pion implementation is
// added in roadmap phase 4.
package transport

import (
	"errors"

	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
)

// ErrNotImplemented marks skeleton stubs awaiting a roadmap phase.
var ErrNotImplemented = errors.New("transport: not implemented (see docs/ROADMAP.md phase 4)")

// Conn is a bidirectional, ordered, reliable byte channel to one peer (a WebRTC
// DataChannel). Payloads are E2E ciphertext produced by pkg/crypto.
type Conn interface {
	// Send queues bytes to the peer. Respects data-channel backpressure.
	Send(b []byte) error
	// Recv returns the next inbound message, blocking until one is available.
	Recv() ([]byte, error)
	// Close tears down the connection.
	Close() error
}

// SignalingExchange is the minimal SDP/ICE exchange the transport needs from the
// signaling layer (pkg/signaling) to negotiate a connection.
type SignalingExchange interface {
	// SendSDP sends a local SDP offer/answer to the peer.
	SendSDP(peerID string, sdp []byte) error
	// RecvSDP returns the peer's SDP.
	RecvSDP(peerID string) ([]byte, error)
	// SendCandidate / RecvCandidate exchange ICE candidates.
	SendCandidate(peerID string, candidate []byte) error
	RecvCandidate(peerID string) ([]byte, error)
}

// Transport establishes peer connections using the configured ICE servers.
type Transport interface {
	// Dial negotiates a connection to peerID using the given signaling exchange.
	Dial(peerID string, sig SignalingExchange) (Conn, error)
	// Accept waits for an inbound connection negotiated via sig.
	Accept(sig SignalingExchange) (Conn, error)
	// Close shuts down the transport.
	Close() error
}

// Config is the transport configuration derived from user settings.
type Config struct {
	ICEServers []settings.ICEServer
}

// New constructs a Transport from settings.
//
// TODO(phase 4): build a pion/webrtc API with cfg.ICEServers mapped to
// webrtc.ICEServer and return a working Transport.
func New(cfg Config) (Transport, error) {
	return nil, ErrNotImplemented
}
