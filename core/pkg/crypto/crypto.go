// Package crypto provides end-to-end encryption for Stunner.
//
// Stunner uses the Signal protocol: an X3DH handshake establishes a shared
// secret from published prekeys (x3dh.go), and the Double Ratchet derives a
// fresh key per message, giving forward secrecy and post-compromise security
// (ratchet.go, session.go).
//
// IMPORTANT: the primitives are the Go standard library's vetted AES-GCM,
// HMAC-SHA256, SHA-512 and X25519. The Double Ratchet / X3DH *composition* here
// is a from-scratch reference implementation and MUST receive an independent
// cryptographic audit before production; swapping in a maintained library such
// as go.mau.fi/libsignal remains a tracked option (see docs/ROADMAP.md).
//
// File payloads are sealed with AES-256-GCM under a per-transfer key carried
// inside the secure session (SealFile/OpenFile, used by pkg/filetransfer).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// ErrNotImplemented is retained for callers that branch on optional features.
var ErrNotImplemented = errors.New("crypto: not implemented")

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

// SealFile seals a single file chunk with AES-256-GCM under transferKey. nonce
// must be unique per (key, chunk); pkg/filetransfer derives it from the chunk
// index. Returns ciphertext||tag.
func SealFile(transferKey, nonce, plaintext []byte) ([]byte, error) {
	gcm, err := fileGCM(transferKey)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("crypto: bad file nonce size")
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

// OpenFile reverses SealFile.
func OpenFile(transferKey, nonce, ciphertext []byte) ([]byte, error) {
	gcm, err := fileGCM(transferKey)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("crypto: bad file nonce size")
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func fileGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("crypto: transfer key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
