package libsignal

import (
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/crypto"
)

// Session implements the parent crypto.Session interface.
var _ crypto.Session = (*Session)(nil)

func TestLibsignalRoundTrip(t *testing.T) {
	alice, err := NewParticipant("alice", 1)
	if err != nil {
		t.Fatalf("alice: %v", err)
	}
	bob, err := NewParticipant("bob", 1)
	if err != nil {
		t.Fatalf("bob: %v", err)
	}

	aSess, err := alice.NewOutboundSession(bob.Fingerprint(), 1, bob.Bundle())
	if err != nil {
		t.Fatalf("outbound: %v", err)
	}
	bSess := bob.NewInboundSession(alice.Fingerprint(), 1)

	if aSess.PeerFingerprint() != bob.Fingerprint() {
		t.Errorf("peer fingerprint = %q, want %q", aSess.PeerFingerprint(), bob.Fingerprint())
	}

	// First message carries the X3DH handshake (PreKeySignalMessage).
	ct, err := aSess.Encrypt([]byte("hello bob 🔒"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	pt, err := bSess.Decrypt(ct)
	if err != nil || string(pt) != "hello bob 🔒" {
		t.Fatalf("decrypt: %v %q", err, pt)
	}

	// Bidirectional traffic ratchets forward.
	for i := 0; i < 3; i++ {
		c, _ := bSess.Encrypt([]byte("from bob"))
		if p, err := aSess.Decrypt(c); err != nil || string(p) != "from bob" {
			t.Fatalf("b->a %d: %v %q", i, err, p)
		}
		c, _ = aSess.Encrypt([]byte("from alice"))
		if p, err := bSess.Decrypt(c); err != nil || string(p) != "from alice" {
			t.Fatalf("a->b %d: %v %q", i, err, p)
		}
	}
}

func TestTamperedFails(t *testing.T) {
	alice, _ := NewParticipant("alice", 1)
	bob, _ := NewParticipant("bob", 1)
	aSess, _ := alice.NewOutboundSession(bob.Fingerprint(), 1, bob.Bundle())
	bSess := bob.NewInboundSession(alice.Fingerprint(), 1)

	ct, _ := aSess.Encrypt([]byte("secret"))
	ct[len(ct)-1] ^= 0xFF
	if _, err := bSess.Decrypt(ct); err == nil {
		t.Error("expected failure on tampered ciphertext")
	}
}
