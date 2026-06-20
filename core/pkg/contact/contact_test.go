package contact

import (
	"errors"
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
)

func TestTOFUFlow(t *testing.T) {
	b := NewMemory()
	alice, _ := identity.Generate()

	// First sighting: trusted, stored, unverified.
	c, err := b.SeenKey("alice", alice.SigningPub)
	if err != nil {
		t.Fatalf("first SeenKey: %v", err)
	}
	if c.Verified {
		t.Error("new contact should be unverified")
	}

	// Same key again: fine.
	if _, err := b.SeenKey("alice", alice.SigningPub); err != nil {
		t.Fatalf("repeat SeenKey: %v", err)
	}

	// Different key for same handle: key change detected.
	imposter, _ := identity.Generate()
	if _, err := b.SeenKey("alice", imposter.SigningPub); !errors.Is(err, ErrKeyChanged) {
		t.Fatalf("expected ErrKeyChanged, got %v", err)
	}
}

func TestMarkVerified(t *testing.T) {
	b := NewMemory()
	alice, _ := identity.Generate()
	b.SeenKey("alice", alice.SigningPub)
	if err := b.MarkVerified("alice"); err != nil {
		t.Fatalf("MarkVerified: %v", err)
	}
	c, _ := b.Get("alice")
	if !c.Verified {
		t.Error("contact should be verified")
	}
	if err := b.MarkVerified("nobody"); err == nil {
		t.Error("expected error for unknown handle")
	}
}

func TestURIRoundTrip(t *testing.T) {
	alice, _ := identity.Generate()
	uri := URI("alice", alice.SigningPub)

	c, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI: %v", err)
	}
	if c.Handle != "alice" {
		t.Errorf("handle = %q", c.Handle)
	}
	if !c.IdentityKey.Equal(alice.SigningPub) {
		t.Error("key mismatch after URI round trip")
	}
	if c.Fingerprint != alice.Fingerprint() {
		t.Error("fingerprint mismatch")
	}
}

func TestParseURIRejectsJunk(t *testing.T) {
	for _, s := range []string{
		"https://example.com",
		"stunner:contact?k=not-base64!!",
		"stunner:other?k=AAAA",
	} {
		if _, err := ParseURI(s); err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}
