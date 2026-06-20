package identity

import (
	"crypto/ed25519"
	"testing"
)

func TestGenerateProducesValidKeys(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(id.SigningPub) != ed25519.PublicKeySize {
		t.Errorf("signing pub size = %d, want %d", len(id.SigningPub), ed25519.PublicKeySize)
	}
	if id.AgreementPriv == nil || id.AgreementPub() == nil {
		t.Error("agreement key not initialized")
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	msg := []byte("stunner")
	sig, err := id.Sign(msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !Verify(id.SigningPub, msg, sig) {
		t.Error("valid signature failed to verify")
	}
	if Verify(id.SigningPub, []byte("tampered"), sig) {
		t.Error("signature verified for wrong message")
	}
}

func TestFingerprintStableAndDistinct(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	fp1 := id.Fingerprint()
	fp2 := Fingerprint(id.SigningPub)
	if fp1 != fp2 {
		t.Errorf("fingerprint not stable: %q != %q", fp1, fp2)
	}
	if fp1 == "" {
		t.Error("empty fingerprint")
	}

	other, _ := Generate()
	if other.Fingerprint() == fp1 {
		t.Error("distinct identities produced identical fingerprints")
	}
}

func TestDiscoveryKeyDependsOnSalt(t *testing.T) {
	id, _ := Generate()
	a := DiscoveryKey(id.SigningPub, []byte("salt-a"))
	b := DiscoveryKey(id.SigningPub, []byte("salt-b"))
	if string(a) == string(b) {
		t.Error("discovery key did not change with salt")
	}
	if len(a) != 32 {
		t.Errorf("discovery key length = %d, want 32", len(a))
	}
}
