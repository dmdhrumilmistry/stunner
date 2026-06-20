package crypto

import (
	"bytes"
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
)

// establish wires up an initiator (Alice) and responder (Bob) session pair via
// X3DH, returning both sessions after Bob accepts Alice's first message.
func establish(t *testing.T) (alice, bob Session) {
	t.Helper()
	aliceID, _ := identity.Generate()
	bobID, _ := identity.Generate()

	alicePre, err := GeneratePreKeys(aliceID)
	if err != nil {
		t.Fatalf("alice prekeys: %v", err)
	}
	bobPre, err := GeneratePreKeys(bobID)
	if err != nil {
		t.Fatalf("bob prekeys: %v", err)
	}

	aliceStore := NewMemStore(aliceID, alicePre)
	bobStore := NewMemStore(bobID, bobPre)

	alice, hs, err := aliceStore.Initiate(bobPre.Bundle())
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	bob, err = bobStore.Accept(hs)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	return alice, bob
}

func TestRoundTrip(t *testing.T) {
	alice, bob := establish(t)

	ct, err := alice.Encrypt([]byte("hello bob 🔒"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	pt, err := bob.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(pt) != "hello bob 🔒" {
		t.Errorf("got %q", pt)
	}
}

func TestBidirectionalAndRatchet(t *testing.T) {
	alice, bob := establish(t)

	for i := 0; i < 5; i++ {
		ct, _ := alice.Encrypt([]byte("a2b"))
		if pt, err := bob.Decrypt(ct); err != nil || string(pt) != "a2b" {
			t.Fatalf("a->b %d: %v %q", i, err, pt)
		}
		ct, _ = bob.Encrypt([]byte("b2a"))
		if pt, err := alice.Decrypt(ct); err != nil || string(pt) != "b2a" {
			t.Fatalf("b->a %d: %v %q", i, err, pt)
		}
	}
}

func TestOutOfOrderDelivery(t *testing.T) {
	alice, bob := establish(t)

	c1, _ := alice.Encrypt([]byte("m1"))
	c2, _ := alice.Encrypt([]byte("m2"))
	c3, _ := alice.Encrypt([]byte("m3"))

	// Deliver out of order: m3, m1, m2.
	if pt, err := bob.Decrypt(c3); err != nil || string(pt) != "m3" {
		t.Fatalf("m3: %v %q", err, pt)
	}
	if pt, err := bob.Decrypt(c1); err != nil || string(pt) != "m1" {
		t.Fatalf("m1: %v %q", err, pt)
	}
	if pt, err := bob.Decrypt(c2); err != nil || string(pt) != "m2" {
		t.Fatalf("m2: %v %q", err, pt)
	}
}

func TestTamperedCiphertextFails(t *testing.T) {
	alice, bob := establish(t)
	ct, _ := alice.Encrypt([]byte("secret"))
	ct[len(ct)-1] ^= 0xFF // flip a byte in the (base64) ciphertext
	if _, err := bob.Decrypt(ct); err == nil {
		t.Error("expected decryption failure for tampered ciphertext")
	}
}

func TestPeerFingerprintMatches(t *testing.T) {
	aliceID, _ := identity.Generate()
	bobID, _ := identity.Generate()
	alicePre, _ := GeneratePreKeys(aliceID)
	bobPre, _ := GeneratePreKeys(bobID)
	aliceStore := NewMemStore(aliceID, alicePre)
	bobStore := NewMemStore(bobID, bobPre)

	alice, hs, _ := aliceStore.Initiate(bobPre.Bundle())
	bob, _ := bobStore.Accept(hs)

	if alice.PeerFingerprint() != bobID.Fingerprint() {
		t.Error("alice's peer fingerprint != bob's identity fingerprint")
	}
	if bob.PeerFingerprint() != aliceID.Fingerprint() {
		t.Error("bob's peer fingerprint != alice's identity fingerprint")
	}
}

func TestSealOpenFile(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	nonce := bytes.Repeat([]byte{3}, 12)
	ct, err := SealFile(key, nonce, []byte("file chunk"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	pt, err := OpenFile(key, nonce, ct)
	if err != nil || string(pt) != "file chunk" {
		t.Fatalf("open: %v %q", err, pt)
	}
}
