// Package storage persists Stunner data locally, encrypted at rest.
//
// The store is backed by SQLCipher (planned: github.com/mutecomm/go-sqlcipher,
// roadmap phase 5). The database key is NOT kept on disk in plaintext; it lives
// in the platform secure store (Keychain / Android Keystore / Windows DPAPI /
// macOS Keychain), provided by the app over the FFI boundary, and is gated by
// the app-lock (biometric/PIN). See docs/THREAT_MODEL.md (Data at rest).
//
// This file defines the stable Store interface; implemented in roadmap phase 5.
package storage

import (
	"errors"

	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
)

// ErrNotImplemented marks skeleton stubs awaiting a roadmap phase.
var ErrNotImplemented = errors.New("storage: not implemented (see docs/ROADMAP.md phase 5)")

// Store is the encrypted persistence API used by the rest of the core.
type Store interface {
	// Settings load/save.
	LoadSettings() (settings.Settings, error)
	SaveSettings(s settings.Settings) error

	// Conversations & messages.
	Conversations() ([]messaging.Conversation, error)
	UpsertConversation(c messaging.Conversation) error
	AppendMessage(convID string, env messaging.Envelope, state messaging.DeliveryState) error
	Messages(convID string, limit, offset int) ([]messaging.Envelope, error)

	// Outbox for pure-P2P retry (offline-delivery tradeoff).
	EnqueueOutbox(convID string, env messaging.Envelope) error
	PendingOutbox() ([]messaging.Envelope, error)

	// Identity & Signal session state (opaque blobs managed by pkg/crypto).
	SaveBlob(namespace, key string, value []byte) error
	LoadBlob(namespace, key string) ([]byte, error)

	// Close flushes and closes the database.
	Close() error
}

// Options configure how the encrypted database is opened.
type Options struct {
	// Path is the database file location.
	Path string
	// Key is the SQLCipher key, supplied by the app from the OS secure store.
	// It must never be logged or persisted in plaintext by this package.
	Key []byte
}

// Open opens (or creates) the encrypted store.
//
// TODO(phase 5): open a SQLCipher database with opts.Key, run migrations, and
// return a working Store.
func Open(opts Options) (Store, error) {
	return nil, ErrNotImplemented
}
