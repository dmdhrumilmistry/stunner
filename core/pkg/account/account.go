// Package account is the persistent local user: an encrypted-at-rest identity
// plus its prekeys and contact book.
//
// The identity is sealed with pkg/vault (AES-256-GCM) using a key the app
// supplies from the OS secure store, and written to <dir>/identity.bin. On next
// launch it is loaded and decrypted. Prekeys and contacts are held in memory in
// this phase and gain persistence via pkg/storage in a later phase.
package account

import (
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"

	"github.com/dmdhrumilmistry/stunner/core/pkg/contact"
	"github.com/dmdhrumilmistry/stunner/core/pkg/crypto"
	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/safetynumber"
	"github.com/dmdhrumilmistry/stunner/core/pkg/vault"
)

const identityFile = "identity.bin"

// Account is the persistent local user.
type Account struct {
	Identity *identity.Identity
	PreKeys  *crypto.PreKeys
	Contacts contact.Book

	dir string
	key []byte
}

// LoadOrCreate loads the encrypted identity from dir, or creates and persists a
// new one if none exists. key must be vault.KeySize bytes (from the OS secure
// store / app-lock).
func LoadOrCreate(dir string, key []byte) (*Account, error) {
	if len(key) != vault.KeySize {
		return nil, errors.New("account: key must be 32 bytes")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, identityFile)

	var id *identity.Identity
	sealed, err := os.ReadFile(path)
	switch {
	case err == nil:
		blob, oerr := vault.Open(key, sealed)
		if oerr != nil {
			return nil, oerr
		}
		if id, err = identity.Unmarshal(blob); err != nil {
			return nil, err
		}
	case os.IsNotExist(err):
		if id, err = identity.Generate(); err != nil {
			return nil, err
		}
		sealed, err = vault.Seal(key, id.Marshal())
		if err != nil {
			return nil, err
		}
		if err = os.WriteFile(path, sealed, 0o600); err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	pre, err := crypto.GeneratePreKeys(id)
	if err != nil {
		return nil, err
	}
	return &Account{
		Identity: id,
		PreKeys:  pre,
		Contacts: contact.NewMemory(),
		dir:      dir,
		key:      key,
	}, nil
}

// Fingerprint returns the local identity fingerprint.
func (a *Account) Fingerprint() string { return a.Identity.Fingerprint() }

// ContactURI returns this account's shareable contact URI (for a QR code).
func (a *Account) ContactURI(handle string) string {
	return contact.URI(handle, a.Identity.SigningPub)
}

// SafetyNumberWith computes the verification safety number with a peer.
func (a *Account) SafetyNumberWith(peer ed25519.PublicKey) string {
	return safetynumber.Compute(a.Identity.SigningPub, peer)
}

// Sessions creates a session store bound to this account's identity and prekeys.
func (a *Account) Sessions() crypto.SessionStore {
	return crypto.NewMemStore(a.Identity, a.PreKeys)
}
