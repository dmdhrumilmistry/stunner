// Package core ties the Stunner subsystems together and exposes high-level
// operations consumed by the FFI layers (core/mobile and core/ffi) and the
// headless harness (core/cmd/stunnerd).
package core

import (
	"fmt"
	"time"

	"github.com/dmdhrumilmistry/stunner/core/pkg/contact"
	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/netcheck"
	"github.com/dmdhrumilmistry/stunner/core/pkg/safetynumber"
	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
)

// Version is the semantic version of the Stunner core.
const Version = "0.2.0"

// stunProbeTimeout bounds how long CheckSTUN waits for ICE gathering.
const stunProbeTimeout = 8 * time.Second

// CheckSTUN probes the default STUN servers and reports whether a public
// (server-reflexive) address could be discovered — the app's "Test STUN
// connection" diagnostic. ok is true when STUN succeeded; detail is a
// human-readable summary for display.
func CheckSTUN() (ok bool, reflexiveAddr, detail string, err error) {
	res, err := netcheck.STUN(settings.DefaultICEServers(), stunProbeTimeout)
	if err != nil {
		return false, "", "", err
	}
	return res.OK, res.ReflexiveAddr, res.Detail(), nil
}

// VersionString returns a human-readable version banner.
func VersionString() string {
	return fmt.Sprintf("Stunner core %s", Version)
}

// NewIdentityFingerprint generates a fresh identity and returns its display
// fingerprint.
func NewIdentityFingerprint() (string, error) {
	id, err := identity.Generate()
	if err != nil {
		return "", err
	}
	return id.Fingerprint(), nil
}

// NewContactURI generates a fresh identity and returns its shareable
// "stunner:contact" URI (suitable for a QR code). This is a convenience for the
// UI before the stateful, persistent account is exposed over FFI; the returned
// identity is ephemeral to this call.
func NewContactURI(handle string) (string, error) {
	id, err := identity.Generate()
	if err != nil {
		return "", err
	}
	return contact.URI(handle, id.SigningPub), nil
}

// ValidateContactURI parses a "stunner:contact" URI and returns the embedded
// handle and identity fingerprint, or an error if it is malformed.
func ValidateContactURI(uri string) (handle, fingerprint string, err error) {
	c, err := contact.ParseURI(uri)
	if err != nil {
		return "", "", err
	}
	return c.Handle, c.Fingerprint, nil
}

// SafetyNumber computes the verification safety number between two contacts,
// each given as a "stunner:contact" URI. The result is identical on both
// devices and is compared out of band to detect MITM.
func SafetyNumber(uriA, uriB string) (string, error) {
	a, err := contact.ParseURI(uriA)
	if err != nil {
		return "", err
	}
	b, err := contact.ParseURI(uriB)
	if err != nil {
		return "", err
	}
	return safetynumber.Compute(a.IdentityKey, b.IdentityKey), nil
}
