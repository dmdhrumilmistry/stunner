// Package mobile is the gomobile-bound API surface for Android and iOS.
//
// Only gobind-safe types are exposed here (strings, ints, []byte, and simple
// structs / interface callbacks). Build the bindings with, e.g.:
//
//	gomobile bind -target=android -o ../app/android/stunnercore.aar ./mobile
//	gomobile bind -target=ios     -o ../app/ios/Stunnercore.xcframework ./mobile
//
// See ../../docs/ROADMAP.md for the full build instructions. The skeleton
// exposes Version/Ping plus a NewIdentityFingerprint smoke test; richer APIs
// (send/receive, settings, file transfer) are added per roadmap phase.
package mobile

import "github.com/dmdhrumilmistry/stunner/core/pkg/core"

// Version returns the core version string. Smoke test for the FFI boundary.
func Version() string { return core.VersionString() }

// Ping echoes msg back, prefixed, proving the Dart<->Go call path works.
func Ping(msg string) string { return "pong: " + msg }

// NewIdentityFingerprint generates a fresh identity and returns its fingerprint.
func NewIdentityFingerprint() (string, error) { return core.NewIdentityFingerprint() }

// EventHandler receives asynchronous events pushed from the core (incoming
// messages, presence, transfer progress). gomobile turns this into a callback
// interface the app implements. Wired up in later roadmap phases.
type EventHandler interface {
	OnEvent(kind string, payloadJSON string)
}
