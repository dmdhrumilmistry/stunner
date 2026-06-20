// Package signaling discovers peers and exchanges the SDP/ICE needed to open a
// WebRTC connection — without a central server.
//
// The production implementation (libp2p.go) is a libp2p host participating in a
// Kademlia DHT: a node advertises and is found under a salted hash of its
// identity key (the "discovery key", see pkg/identity.DiscoveryKey), and SDP/ICE
// is exchanged over authenticated libp2p streams. An in-memory Registry
// (memory.go) provides the same interface for tests and the headless harness.
package signaling

import "errors"

// ErrNotImplemented is returned by the in-memory signaler when a lookup fails.
var ErrNotImplemented = errors.New("signaling: peer not found")

// PeerInfo is what discovery yields for a peer: an identity and the addresses
// the signaling layer can reach it on.
type PeerInfo struct {
	PeerID    string
	Addresses []string
}

// Signaler is the pluggable control-plane abstraction. Implementations:
//   - DHTSignaler (production, decentralized): libp2p Kademlia (libp2p.go).
//   - Registry/memSignaler (tests, in-process): memory.go.
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
