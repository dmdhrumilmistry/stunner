// Package core ties the Stunner subsystems together and exposes the high-level
// operations consumed by the FFI layers (core/mobile and core/ffi) and the
// headless harness (core/cmd/stunnerd).
//
// In the skeleton it provides version information and identity generation; the
// messaging node is assembled here as the roadmap phases land.
package core

import (
	"fmt"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
)

// Version is the semantic version of the Stunner core.
const Version = "0.0.1-skeleton"

// VersionString returns a human-readable version banner.
func VersionString() string {
	return fmt.Sprintf("Stunner core %s", Version)
}

// NewIdentityFingerprint generates a fresh identity and returns its display
// fingerprint. Used by the headless harness and as a simple end-to-end check
// that key generation works across the FFI boundary.
func NewIdentityFingerprint() (string, error) {
	id, err := identity.Generate()
	if err != nil {
		return "", err
	}
	return id.Fingerprint(), nil
}
