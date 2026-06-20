package messaging

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/dmdhrumilmistry/stunner/core/pkg/crypto"
)

// ProtocolVersion is the current application envelope version.
const ProtocolVersion = 1

// NewEnvelope builds an envelope with a random message id and current timestamp.
func NewEnvelope(t Type, convID string, body []byte) Envelope {
	return Envelope{
		Version:   ProtocolVersion,
		Type:      t,
		MsgID:     newID(),
		ConvID:    convID,
		Timestamp: time.Now().UTC(),
		Body:      body,
	}
}

// NewText builds a TEXT envelope for a conversation.
func NewText(convID, text string, animatedEmoji ...string) (Envelope, error) {
	body, err := json.Marshal(TextBody{Text: text, AnimatedEmoji: animatedEmoji})
	if err != nil {
		return Envelope{}, err
	}
	return NewEnvelope(TypeText, convID, body), nil
}

// Encode serializes an envelope (pre-encryption).
func (e Envelope) Encode() ([]byte, error) { return json.Marshal(e) }

// DecodeEnvelope parses a decrypted envelope.
func DecodeEnvelope(b []byte) (Envelope, error) {
	var e Envelope
	err := json.Unmarshal(b, &e)
	return e, err
}

// Text extracts the TextBody from a TEXT envelope.
func (e Envelope) Text() (TextBody, error) {
	var t TextBody
	err := json.Unmarshal(e.Body, &t)
	return t, err
}

// Frame is what travels over a transport.Conn: an optional X3DH handshake (only
// on the first frame from an initiator) plus the ratchet-encrypted envelope.
type Frame struct {
	Handshake *crypto.Handshake `json:"hs,omitempty"`
	Payload   []byte            `json:"payload,omitempty"`
}

// EncodeFrame serializes a frame for the wire.
func EncodeFrame(f Frame) ([]byte, error) { return json.Marshal(f) }

// DecodeFrame parses a wire frame.
func DecodeFrame(b []byte) (Frame, error) {
	var f Frame
	err := json.Unmarshal(b, &f)
	return f, err
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
