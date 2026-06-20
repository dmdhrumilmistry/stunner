// Package signaling discovers peers and exchanges the SDP/ICE needed to open a
// WebRTC connection — without a central server.
//
// The default implementation (roadmap phase 4) is a libp2p host participating in
// a Kademlia DHT: a user advertises and is found under a salted hash of their
// identity key (the "discovery key", see pkg/identity.DiscoveryKey), and SDP/ICE
// is exchanged over an authenticated libp2p stream. An optional relay
// implementation is available for networks hostile to DHT traffic.
//
// This file defines the pluggable Signaler interface.
package signaling

import "errors"

// ErrNotImplemented marks skeleton stubs awaiting a roadmap phase.
var ErrNotImplemented = errors.New("signaling: not implemented (see docs/ROADMAP.md phase 4)")

// PeerInfo is what discovery yields for a peer: an identity and the addresses
// the signaling layer can reach it on.
type PeerInfo struct {
	PeerID    string
	Addresses []string
}

// Signaler is the pluggable control-plane abstraction. Implementations:
//   - DHT (default, decentralized): libp2p Kademlia.
//   - Relay (optional): a minimal self-hostable signaling server.
type Signaler interface {
	// Advertise publishes this node under discoveryKey so peers can find it.
	Advertise(discoveryKey []byte) error
	// Find locates a peer by its discovery key.
	Find(discoveryKey []byte) (PeerInfo, error)

	// SendSDP / RecvSDP and SendCandidate / RecvCandidate exchange WebRTC
	// negotiation data with a peer over an authenticated channel.
	SendSDP(peerID string, sdp []byte) error
	RecvSDP(peerID string) ([]byte, error)
	SendCandidate(peerID string, candidate []byte) error
	RecvCandidate(peerID string) ([]byte, error)

	// Presence reports whether a peer currently appears reachable; used by the
	// messaging outbox to drive retries (offline-delivery tradeoff).
	Presence(peerID string) (online bool, err error)

	// Close shuts down the signaler.
	Close() error
}

// NewDHT constructs the default decentralized signaler.
//
// TODO(phase 4): build a libp2p host + Kademlia DHT and implement Signaler.
func NewDHT() (Signaler, error) {
	return nil, ErrNotImplemented
}

// NewRelay constructs an optional relay-backed signaler.
//
// TODO(phase 8): connect to a self-hostable signaling/mailbox relay.
func NewRelay(address string) (Signaler, error) {
	return nil, ErrNotImplemented
}
