package account

import (
	"bytes"
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/contact"
)

func key() []byte { return bytes.Repeat([]byte{9}, 32) }

func TestLoadOrCreatePersistsIdentity(t *testing.T) {
	dir := t.TempDir()

	a, err := LoadOrCreate(dir, key())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	fp := a.Fingerprint()

	// Reload from disk: same identity.
	b, err := LoadOrCreate(dir, key())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if b.Fingerprint() != fp {
		t.Errorf("identity not persisted: %q != %q", b.Fingerprint(), fp)
	}
}

func TestWrongKeyFailsToLoad(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadOrCreate(dir, key()); err != nil {
		t.Fatalf("create: %v", err)
	}
	wrong := bytes.Repeat([]byte{1}, 32)
	if _, err := LoadOrCreate(dir, wrong); err == nil {
		t.Error("expected failure decrypting identity with wrong key")
	}
}

func TestContactURIAndSafetyNumber(t *testing.T) {
	a, _ := LoadOrCreate(t.TempDir(), key())
	b, _ := LoadOrCreate(t.TempDir(), key())

	// A parses B's contact URI and computes a safety number.
	c, err := contact.ParseURI(b.ContactURI("bob"))
	if err != nil {
		t.Fatalf("parse uri: %v", err)
	}
	snA := a.SafetyNumberWith(c.IdentityKey)
	snB := b.SafetyNumberWith(a.Identity.SigningPub)
	if snA != snB {
		t.Errorf("safety numbers differ:\n A=%s\n B=%s", snA, snB)
	}
}
