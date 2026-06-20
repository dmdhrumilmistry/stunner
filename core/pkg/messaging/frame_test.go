package messaging

import "testing"

func TestEnvelopeRoundTrip(t *testing.T) {
	env, err := NewText("conv1", "hi :wave:", "party_parrot")
	if err != nil {
		t.Fatalf("NewText: %v", err)
	}
	if env.Type != TypeText || env.Version != ProtocolVersion || env.MsgID == "" {
		t.Errorf("bad envelope: %+v", env)
	}

	raw, err := env.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := DecodeEnvelope(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	body, err := back.Text()
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	if body.Text != "hi :wave:" || len(body.AnimatedEmoji) != 1 || body.AnimatedEmoji[0] != "party_parrot" {
		t.Errorf("bad text body: %+v", body)
	}
}

func TestFrameRoundTrip(t *testing.T) {
	f := Frame{Payload: []byte("ciphertext")}
	raw, err := EncodeFrame(f)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := DecodeFrame(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(back.Payload) != "ciphertext" || back.Handshake != nil {
		t.Errorf("bad frame: %+v", back)
	}
}
