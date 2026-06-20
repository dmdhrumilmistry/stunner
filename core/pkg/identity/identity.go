// Package identity manages a user's long-term cryptographic identity.
//
// Each Stunner install owns an Ed25519 identity keypair (the canonical user
// "address") and an X25519 key-agreement keypair used by the Signal X3DH
// handshake. The public identity key is hashed to produce a human-comparable
// fingerprint ("safety number") for out-of-band verification, and a salted hash
// for DHT discovery.
//
// This package uses only the Go standard library (crypto/ed25519, crypto/ecdh)
// so it works without external dependencies. See ../../docs/PROTOCOL.md §1.
package identity

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
)

// Identity is a user's long-term key material. The private components must never
// leave the device unencrypted; persistence is handled by pkg/storage.
type Identity struct {
	// SigningPub/SigningPriv are the Ed25519 identity keys.
	SigningPub  ed25519.PublicKey
	SigningPriv ed25519.PrivateKey

	// AgreementPriv is the X25519 private key used for key agreement (X3DH).
	AgreementPriv *ecdh.PrivateKey
}

// Generate creates a fresh identity with new Ed25519 and X25519 keypairs.
func Generate() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("identity: generate ed25519: %w", err)
	}
	agreement, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("identity: generate x25519: %w", err)
	}
	return &Identity{
		SigningPub:    pub,
		SigningPriv:   priv,
		AgreementPriv: agreement,
	}, nil
}

// AgreementPub returns the X25519 public key for this identity.
func (id *Identity) AgreementPub() *ecdh.PublicKey {
	return id.AgreementPriv.PublicKey()
}

// Fingerprint returns a stable, human-comparable representation of the public
// identity key. Users compare fingerprints (or scan QR codes) to authenticate
// each other and defeat active MITM attacks.
func (id *Identity) Fingerprint() string {
	return Fingerprint(id.SigningPub)
}

// Fingerprint computes the display fingerprint for an Ed25519 public key:
// uppercase base32 of SHA-256(pub), grouped into 5-char blocks for readability.
func Fingerprint(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	return group(enc, 5)
}

// DiscoveryKey returns the DHT lookup key for a public identity key: a salted
// hash so the raw identity key is not used directly as the rendezvous key.
// See ../../docs/PROTOCOL.md §1.
func DiscoveryKey(pub ed25519.PublicKey, rendezvousSalt []byte) []byte {
	h := sha256.New()
	h.Write(pub)
	h.Write(rendezvousSalt)
	return h.Sum(nil)
}

// Verify reports whether sig is a valid signature of msg by pub.
func Verify(pub ed25519.PublicKey, msg, sig []byte) bool {
	if len(pub) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(pub, msg, sig)
}

// Sign signs msg with the identity's signing key.
func (id *Identity) Sign(msg []byte) ([]byte, error) {
	if len(id.SigningPriv) != ed25519.PrivateKeySize {
		return nil, errors.New("identity: signing key not initialized")
	}
	return ed25519.Sign(id.SigningPriv, msg), nil
}

// group inserts a space every n characters for readable fingerprints.
func group(s string, n int) string {
	if n <= 0 {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && i%n == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
