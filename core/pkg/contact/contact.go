// Package contact models the people a user communicates with and implements
// trust-on-first-use (TOFU) key-change detection plus QR/URI contact exchange.
//
// A Contact pairs a user-assigned handle with an Ed25519 identity public key.
// When a key claiming a known handle changes, SeenKey returns ErrKeyChanged so
// the UI can warn the user (a potential MITM or a peer who reinstalled).
//
// Contacts are exchanged as a "stunner:contact" URI encoded into a QR code by
// the app; URI() / ParseURI handle the (de)serialization. Stdlib only.
package contact

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
)

// ErrKeyChanged indicates a known handle presented a different identity key.
var ErrKeyChanged = errors.New("contact: identity key changed")

// Contact is a known peer.
type Contact struct {
	Handle      string            `json:"handle"`
	IdentityKey ed25519.PublicKey `json:"identityKey"`
	Fingerprint string            `json:"fingerprint"`
	AddedAt     time.Time         `json:"addedAt"`
	Verified    bool              `json:"verified"`
}

// New builds a Contact from a handle and identity key, filling the fingerprint.
func New(handle string, key ed25519.PublicKey) Contact {
	return Contact{
		Handle:      handle,
		IdentityKey: key,
		Fingerprint: identity.Fingerprint(key),
		AddedAt:     time.Now().UTC(),
	}
}

// Book stores contacts and enforces TOFU. Implementations may persist via
// pkg/storage; Memory provides an in-memory implementation.
type Book interface {
	// SeenKey applies trust-on-first-use for a handle/key pair:
	//   - unknown handle: stores and returns the new (unverified) contact;
	//   - known handle, same key: returns the stored contact;
	//   - known handle, different key: returns ErrKeyChanged (nothing stored).
	SeenKey(handle string, key ed25519.PublicKey) (Contact, error)
	Get(handle string) (Contact, bool)
	List() []Contact
	// MarkVerified records that the user confirmed a contact's safety number.
	MarkVerified(handle string) error
	// Remove deletes a contact by handle. Removing an unknown handle is a no-op.
	Remove(handle string) error
}

// Memory is an in-memory Book.
type Memory struct {
	mu sync.Mutex
	m  map[string]Contact
}

// NewMemory creates an empty in-memory contact book.
func NewMemory() *Memory { return &Memory{m: map[string]Contact{}} }

func (b *Memory) SeenKey(handle string, key ed25519.PublicKey) (Contact, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	existing, ok := b.m[handle]
	if ok {
		if !existing.IdentityKey.Equal(key) {
			return Contact{}, ErrKeyChanged
		}
		return existing, nil
	}
	c := New(handle, key)
	b.m[handle] = c
	return c, nil
}

func (b *Memory) Get(handle string) (Contact, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	c, ok := b.m[handle]
	return c, ok
}

func (b *Memory) List() []Contact {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Contact, 0, len(b.m))
	for _, c := range b.m {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Handle < out[j].Handle })
	return out
}

func (b *Memory) MarkVerified(handle string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	c, ok := b.m[handle]
	if !ok {
		return errors.New("contact: unknown handle")
	}
	c.Verified = true
	b.m[handle] = c
	return nil
}

func (b *Memory) Remove(handle string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.m, handle)
	return nil
}

// URI encodes a contact as a "stunner:contact" URI suitable for a QR code, e.g.
//
//	stunner:contact?k=<base64url(identityKey)>&n=<handle>
func URI(handle string, key ed25519.PublicKey) string {
	q := url.Values{}
	q.Set("k", base64.RawURLEncoding.EncodeToString(key))
	if handle != "" {
		q.Set("n", handle)
	}
	u := url.URL{Scheme: "stunner", Opaque: "contact", RawQuery: q.Encode()}
	return u.String()
}

// ParseURI parses a contact URI produced by URI.
func ParseURI(s string) (Contact, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Contact{}, err
	}
	if u.Scheme != "stunner" || u.Opaque != "contact" {
		return Contact{}, errors.New("contact: not a stunner contact URI")
	}
	q := u.Query()
	raw, err := base64.RawURLEncoding.DecodeString(q.Get("k"))
	if err != nil {
		return Contact{}, errors.New("contact: bad key encoding")
	}
	if len(raw) != ed25519.PublicKeySize {
		return Contact{}, errors.New("contact: wrong key length")
	}
	return New(q.Get("n"), ed25519.PublicKey(raw)), nil
}
