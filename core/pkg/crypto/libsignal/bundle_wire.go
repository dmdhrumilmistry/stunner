package libsignal

import (
	"encoding/json"
	"errors"

	"go.mau.fi/libsignal/ecc"
	"go.mau.fi/libsignal/keys/identity"
	"go.mau.fi/libsignal/keys/prekey"
	"go.mau.fi/libsignal/util/optional"
)

// wireBundle is the JSON-serializable form of a libsignal prekey bundle, so it
// can be published/exchanged over a transport (the equivalent of sending
// crypto.PreKeyBundle in a handshake frame). Public keys use libsignal's own
// 33-byte point encoding.
type wireBundle struct {
	RegistrationID  uint32  `json:"registrationId"`
	DeviceID        uint32  `json:"deviceId"`
	PreKeyID        *uint32 `json:"preKeyId,omitempty"`
	PreKeyPub       []byte  `json:"preKeyPub"`
	SignedPreKeyID  uint32  `json:"signedPreKeyId"`
	SignedPreKeyPub []byte  `json:"signedPreKeyPub"`
	SignedPreKeySig []byte  `json:"signedPreKeySig"`
	IdentityKey     []byte  `json:"identityKey"`
}

// SerializeBundle encodes a prekey bundle for transport.
func SerializeBundle(b *prekey.Bundle) ([]byte, error) {
	if b == nil {
		return nil, errors.New("libsignal: nil bundle")
	}
	sig := b.SignedPreKeySignature()
	w := wireBundle{
		RegistrationID:  b.RegistrationID(),
		DeviceID:        b.DeviceID(),
		PreKeyPub:       b.PreKey().Serialize(),
		SignedPreKeyID:  b.SignedPreKeyID(),
		SignedPreKeyPub: b.SignedPreKey().Serialize(),
		SignedPreKeySig: sig[:],
		IdentityKey:     b.IdentityKey().Serialize(),
	}
	if id := b.PreKeyID(); id != nil && !id.IsEmpty {
		v := id.Value
		w.PreKeyID = &v
	}
	return json.Marshal(w)
}

// DeserializeBundle reverses SerializeBundle.
func DeserializeBundle(data []byte) (*prekey.Bundle, error) {
	var w wireBundle
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	if len(w.SignedPreKeySig) != 64 {
		return nil, errors.New("libsignal: bad signed-prekey signature length")
	}
	preKeyPub, err := ecc.DecodePoint(w.PreKeyPub, 0)
	if err != nil {
		return nil, err
	}
	signedPreKeyPub, err := ecc.DecodePoint(w.SignedPreKeyPub, 0)
	if err != nil {
		return nil, err
	}
	idPub, err := ecc.DecodePoint(w.IdentityKey, 0)
	if err != nil {
		return nil, err
	}
	var sig [64]byte
	copy(sig[:], w.SignedPreKeySig)

	preKeyID := optional.NewEmptyUint32()
	if w.PreKeyID != nil {
		preKeyID = optional.NewOptionalUint32(*w.PreKeyID)
	}
	return prekey.NewBundle(
		w.RegistrationID, w.DeviceID, preKeyID, w.SignedPreKeyID,
		preKeyPub, signedPreKeyPub, sig, identity.NewKey(idPub),
	), nil
}
