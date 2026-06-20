// Package crypto provides end-to-end encryption for Stunner.
//
// Stunner uses the Signal protocol: an X3DH handshake establishes a shared
// secret from published prekeys, and the Double Ratchet derives a fresh key per
// message, giving forward secrecy and post-compromise security.
//
// IMPORTANT: cryptographic primitives are NEVER hand-rolled. The implementation
// wraps a vetted library (planned: go.mau.fi/libsignal). File payloads are
// sealed with an AEAD (XChaCha20-Poly1305) under a per-transfer key carried
// inside the secure session.
//
// This file defines the stable interfaces the rest of the core depends on; the
// concrete implementation is added in roadmap phase 3.
package crypto

import "errors"

// ErrNotImplemented marks skeleton stubs awaiting a roadmap phase.
var ErrNotImplemented = errors.New("crypto: not implemented (see docs/ROADMAP.md phase 3)")

// Session is an established, end-to-end-encrypted channel with one peer.
// Encrypt/Decrypt operate on whole application envelopes (see docs/PROTOCOL.md
// §4). Each call advances the Double Ratchet.
type Session interface {
	// Encrypt seals a plaintext application envelope for the peer.
	Encrypt(plaintext []byte) (ciphertext []byte, err error)
	// Decrypt opens ciphertext received from the peer.
	Decrypt(ciphertext []byte) (plaintext []byte, err error)
	// PeerFingerprint returns the peer's identity fingerprint for verification.
	PeerFingerprint() string
}

// SessionStore manages secure sessions and the prekeys/state the Signal
// protocol requires. Backed by pkg/storage in later phases.
type SessionStore interface {
	// Get returns an existing session for a peer identity, if any.
	Get(peerID string) (Session, bool)
	// Establish runs X3DH against a peer's published prekey bundle and returns
	// a new session.
	Establish(peerID string, prekeyBundle []byte) (Session, error)
}

// SealFile seals a single file chunk with an AEAD under transferKey.
// nonce is derived from the chunk index (see docs/PROTOCOL.md §5).
//
// TODO(phase 6): implement with XChaCha20-Poly1305.
func SealFile(transferKey, nonce, plaintext []byte) ([]byte, error) {
	return nil, ErrNotImplemented
}

// OpenFile reverses SealFile.
//
// TODO(phase 6): implement with XChaCha20-Poly1305.
func OpenFile(transferKey, nonce, ciphertext []byte) ([]byte, error) {
	return nil, ErrNotImplemented
}
