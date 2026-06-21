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

import (
	"encoding/json"
	"sync"

	"github.com/dmdhrumilmistry/stunner/core/pkg/core"
	"github.com/dmdhrumilmistry/stunner/core/pkg/runtime"
)

// The messaging runtime is a process-global singleton driven from the app. All
// calls are non-blocking; Poll drains incoming messages / status events as JSON.
var (
	rtMu sync.Mutex
	rt   *runtime.Runtime
)

// StartResult is the gobind-friendly result of Start (one struct + error).
type StartResult struct {
	URI         string
	Fingerprint string
}

// Start loads/creates the persistent account at dataDir and starts the live
// messaging runtime (WebRTC + DHT). handle is the display name in the URI.
func Start(dataDir, handle string) (*StartResult, error) {
	rtMu.Lock()
	defer rtMu.Unlock()
	if rt == nil {
		r, err := runtime.Start(dataDir, handle)
		if err != nil {
			return nil, err
		}
		rt = r
	}
	return &StartResult{URI: rt.MyURI(), Fingerprint: rt.Fingerprint()}, nil
}

// Send enqueues a text message to the peer identified by their contact URI.
func Send(peerURI, text, msgID string) {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r != nil {
		r.Send(peerURI, text, msgID)
	}
}

// MarkRead sends a read receipt for the latest message from the peer (call when
// the user opens the conversation).
func MarkRead(peerURI string) {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r != nil {
		r.MarkRead(peerURI)
	}
}

// Poll returns pending runtime events as a JSON array (empty "[]" if none).
func Poll() string {
	rtMu.Lock()
	r := rt
	rtMu.Unlock()
	if r == nil {
		return "[]"
	}
	b, err := json.Marshal(r.Poll())
	if err != nil {
		return "[]"
	}
	return string(b)
}

// Stop shuts down the runtime.
func Stop() {
	rtMu.Lock()
	r := rt
	rt = nil
	rtMu.Unlock()
	if r != nil {
		_ = r.Stop()
	}
}

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

// ContactInfo is the gobind-friendly result of ValidateContactURI. gomobile
// requires exported functions to return at most one value plus an error, so the
// two fields are wrapped in a struct (exposed as a class to Java/Swift).
type ContactInfo struct {
	Handle      string
	Fingerprint string
}

// ValidateContactURI parses a scanned contact URI, returning the handle and
// identity fingerprint.
func ValidateContactURI(uri string) (*ContactInfo, error) {
	handle, fingerprint, err := core.ValidateContactURI(uri)
	if err != nil {
		return nil, err
	}
	return &ContactInfo{Handle: handle, Fingerprint: fingerprint}, nil
}

// STUNResult is the gobind-friendly result of CheckSTUN (one struct + error).
type STUNResult struct {
	OK            bool
	ReflexiveAddr string
	Detail        string
}

// CheckSTUN probes the default STUN servers and reports whether a public address
// could be discovered — the app's "Test STUN connection" diagnostic.
func CheckSTUN() (*STUNResult, error) {
	ok, addr, detail, err := core.CheckSTUN()
	if err != nil {
		return nil, err
	}
	return &STUNResult{OK: ok, ReflexiveAddr: addr, Detail: detail}, nil
}

// EventHandler receives asynchronous events pushed from the core (incoming
// messages, presence, transfer progress). gomobile turns this into a callback
// interface the app implements. Wired up as the runtime is exposed over FFI.
type EventHandler interface {
	OnEvent(kind string, payloadJSON string)
}
