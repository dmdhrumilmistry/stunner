// Package transport carries already-encrypted bytes between peers over WebRTC
// data channels.
//
// The production implementation (pion.go) uses pion/webrtc. ICE servers
// (STUN/TURN) come from pkg/settings and are passed straight into the WebRTC
// configuration, so users can override NAT-traversal infrastructure. TURN, when
// used, only relays E2E ciphertext and cannot read content. An in-process Pipe
// (inproc.go) provides the same Conn interface for tests and the headless
// harness.
package transport

import (
	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
)

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

// SignalingExchange is the minimal SDP exchange the transport needs from the
// signaling layer (pkg/signaling) to negotiate a connection. The transport uses
// non-trickle ICE, so candidates are embedded in the SDP and only SendSDP/RecvSDP
// are required; SendCandidate/RecvCandidate are reserved for trickle-ICE
// implementations. The peerID argument is a routing hint that some signalers
// (e.g. per-stream ones) may ignore.
type SignalingExchange interface {
	SendSDP(peerID string, sdp []byte) error
	RecvSDP(peerID string) ([]byte, error)
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
