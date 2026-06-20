package vault

import (
	"bytes"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{1}, KeySize)
	sealed, err := Seal(key, []byte("top secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	pt, err := Open(key, sealed)
	if err != nil || string(pt) != "top secret" {
		t.Fatalf("open: %v %q", err, pt)
	}
}

func TestOpenWithWrongKeyFails(t *testing.T) {
	sealed, _ := Seal(bytes.Repeat([]byte{1}, KeySize), []byte("x"))
	if _, err := Open(bytes.Repeat([]byte{2}, KeySize), sealed); err == nil {
		t.Error("expected failure with wrong key")
	}
}

func TestBadKeySize(t *testing.T) {
	if _, err := Seal([]byte("short"), []byte("x")); err == nil {
		t.Error("expected error for short key")
	}
}

func TestDeriveKeyDeterministicAndSalted(t *testing.T) {
	salt := []byte("salt-value-16byt")
	a := DeriveKey("hunter2", salt, 1000)
	b := DeriveKey("hunter2", salt, 1000)
	if !bytes.Equal(a, b) {
		t.Error("derivation not deterministic")
	}
	if len(a) != KeySize {
		t.Errorf("key size = %d, want %d", len(a), KeySize)
	}
	if bytes.Equal(a, DeriveKey("hunter2", []byte("different-salt16"), 1000)) {
		t.Error("salt did not affect derived key")
	}
}
