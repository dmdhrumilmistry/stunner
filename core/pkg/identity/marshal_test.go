package identity

import (
	"bytes"
	"testing"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	blob := id.Marshal()

	back, err := Unmarshal(blob)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bytes.Equal(id.SigningPriv, back.SigningPriv) {
		t.Error("signing priv mismatch")
	}
	if !bytes.Equal(id.AgreementPriv.Bytes(), back.AgreementPriv.Bytes()) {
		t.Error("agreement priv mismatch")
	}
	if id.Fingerprint() != back.Fingerprint() {
		t.Error("fingerprint mismatch after round trip")
	}

	// A signature from the original must verify under the restored public key.
	sig, _ := id.Sign([]byte("msg"))
	if !Verify(back.SigningPub, []byte("msg"), sig) {
		t.Error("restored public key failed to verify signature")
	}
}

func TestUnmarshalRejectsBadLength(t *testing.T) {
	if _, err := Unmarshal([]byte("too short")); err == nil {
		t.Error("expected error for bad length")
	}
}
