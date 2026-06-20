// Package mobile is the gomobile-bound API surface for Android and iOS.
//
// Only gobind-safe types are exposed here (strings, ints, []byte, and simple
// return tuples). Build the bindings with, e.g.:
//
//	gomobile bind -target=android -o ../app/android/stunnercore.aar ./mobile
//	gomobile bind -target=ios     -o ../app/ios/Stunnercore.xcframework ./mobile
//
// See ../../docs/ROADMAP.md for the full build instructions.
package mobile

import "github.com/dmdhrumilmistry/stunner/core/pkg/core"

// Version returns the core version string.
func Version() string { return core.VersionString() }

// Ping echoes msg back, prefixed, proving the Dart<->Go call path works.
func Ping(msg string) string { return "pong: " + msg }

// NewIdentityFingerprint generates a fresh identity and returns its fingerprint.
func NewIdentityFingerprint() (string, error) { return core.NewIdentityFingerprint() }

// NewContactURI generates a fresh identity and returns its shareable contact
// URI (for a QR code). Ephemeral convenience until the persistent account is
// exposed over FFI.
func NewContactURI(handle string) (string, error) { return core.NewContactURI(handle) }

// SafetyNumber computes the verification safety number between two contacts,
// each passed as a "stunner:contact" URI (e.g. scanned from a QR code).
func SafetyNumber(myContactURI, peerContactURI string) (string, error) {
	return core.SafetyNumber(myContactURI, peerContactURI)
}

// ValidateContactURI parses a scanned contact URI, returning the handle and
// identity fingerprint (gomobile maps multiple returns to an out-param struct).
func ValidateContactURI(uri string) (handle string, fingerprint string, err error) {
	return core.ValidateContactURI(uri)
}

// EventHandler receives asynchronous events pushed from the core (incoming
// messages, presence, transfer progress). gomobile turns this into a callback
// interface the app implements. Wired up as the runtime is exposed over FFI.
type EventHandler interface {
	OnEvent(kind string, payloadJSON string)
}
