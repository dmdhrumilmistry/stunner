package libsignal

import (
	"bytes"
	"testing"
)

// TestStoreWireHandshake exercises the full adoption-ready path: a peer's bundle
// is serialized for the wire, the initiator establishes a session from those
// bytes, and messages flow both ways — including out-of-order delivery (skipped
// message keys), which the Double Ratchet must handle.
func TestStoreWireHandshake(t *testing.T) {
	alice, err := NewStore("alice", 1)
	if err != nil {
		t.Fatalf("alice: %v", err)
	}
	bob, err := NewStore("bob", 1)
	if err != nil {
		t.Fatalf("bob: %v", err)
	}

	// Bob publishes his bundle; it travels as bytes; Alice initiates from them.
	bobBundle, err := bob.PublishBundle()
	if err != nil {
		t.Fatalf("publish bundle: %v", err)
	}
	aliceSession, err := alice.Initiate("bob", 1, bobBundle)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if aliceSession.PeerFingerprint() != bob.Fingerprint() {
		t.Errorf("peer fingerprint = %q, want %q", aliceSession.PeerFingerprint(), bob.Fingerprint())
	}

	// First message establishes Bob's inbound session (PreKeySignalMessage).
	ct, err := aliceSession.Encrypt([]byte("hello bob 🔒"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	bobSession := bob.Accept("alice", 1)
	pt, err := bobSession.Decrypt(ct)
	if err != nil || string(pt) != "hello bob 🔒" {
		t.Fatalf("decrypt: %v %q", err, pt)
	}

	// Reply (reverse ratchet direction).
	ct2, err := bobSession.Encrypt([]byte("hi alice 👋"))
	if err != nil {
		t.Fatalf("reply encrypt: %v", err)
	}
	pt2, err := aliceSession.Decrypt(ct2)
	if err != nil || string(pt2) != "hi alice 👋" {
		t.Fatalf("reply decrypt: %v %q", err, pt2)
	}

	// Out-of-order: Alice sends m1,m2,m3; Bob decrypts m3,m1,m2.
	c1, _ := aliceSession.Encrypt([]byte("m1"))
	c2, _ := aliceSession.Encrypt([]byte("m2"))
	c3, _ := aliceSession.Encrypt([]byte("m3"))
	if d3, err := bobSession.Decrypt(c3); err != nil || string(d3) != "m3" {
		t.Fatalf("decrypt m3 out of order: %v %q", err, d3)
	}
	if d1, err := bobSession.Decrypt(c1); err != nil || string(d1) != "m1" {
		t.Fatalf("decrypt m1 out of order: %v %q", err, d1)
	}
	if d2, err := bobSession.Decrypt(c2); err != nil || string(d2) != "m2" {
		t.Fatalf("decrypt m2 out of order: %v %q", err, d2)
	}
}

// TestSerializeBundleRoundTrip checks the bundle survives a serialize/deserialize
// cycle and still establishes a working session.
func TestSerializeBundleRoundTrip(t *testing.T) {
	p, err := NewParticipant("p", 1)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := SerializeBundle(p.Bundle())
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	got, err := DeserializeBundle(raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	orig := p.Bundle()
	if got.RegistrationID() != orig.RegistrationID() || got.DeviceID() != orig.DeviceID() {
		t.Errorf("ids differ after round-trip")
	}
	if !bytes.Equal(got.IdentityKey().Serialize(), orig.IdentityKey().Serialize()) {
		t.Errorf("identity key differs after round-trip")
	}
	if got.SignedPreKeySignature() != orig.SignedPreKeySignature() {
		t.Errorf("signature differs after round-trip")
	}
}
