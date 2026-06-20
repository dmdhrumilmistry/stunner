// Package vault provides authenticated encryption for data at rest.
//
// Seal/Open use AES-256-GCM with a 32-byte key. In Stunner the key is supplied
// by the platform secure store (Keychain / Android Keystore / Windows DPAPI /
// macOS Keychain) and never persisted in plaintext; see docs/THREAT_MODEL.md.
// DeriveKey turns a passphrase/PIN into a key for the app-lock flow using
// PBKDF2-HMAC-SHA256 (RFC 2898).
//
// This package uses only the Go standard library.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"
	"io"
)

// KeySize is the required key length for Seal/Open (AES-256).
const KeySize = 32

// Seal encrypts plaintext with key, returning nonce||ciphertext. key must be
// KeySize bytes.
func Seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Open reverses Seal.
func Open(key, sealed []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(sealed) < ns {
		return nil, errors.New("vault: ciphertext too short")
	}
	nonce, ct := sealed[:ns], sealed[ns:]
	return gcm.Open(nil, nonce, ct, nil)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, errors.New("vault: key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// DeriveKey derives a KeySize key from a passphrase and salt using
// PBKDF2-HMAC-SHA256 with the given iteration count.
func DeriveKey(passphrase string, salt []byte, iterations int) []byte {
	return pbkdf2([]byte(passphrase), salt, iterations, KeySize, sha256.New)
}

// pbkdf2 is a minimal RFC 2898 PBKDF2 implementation over an HMAC PRF.
func pbkdf2(password, salt []byte, iter, keyLen int, h func() hash.Hash) []byte {
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	out := make([]byte, 0, numBlocks*hashLen)
	u := make([]byte, hashLen)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		var idx [4]byte
		binary.BigEndian.PutUint32(idx[:], uint32(block))
		prf.Write(idx[:])
		t := prf.Sum(nil)
		copy(u, t)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for i := range t {
				t[i] ^= u[i]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}
