package identity

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"errors"
)

// marshaledSize is the fixed serialized length: Ed25519 private key (64) plus
// the X25519 private scalar (32).
const marshaledSize = ed25519.PrivateKeySize + 32

// Marshal serializes the identity's private key material into a fixed-size blob.
//
// The blob contains secret keys and MUST only ever be persisted sealed (see
// pkg/vault) and stored with the database key in the OS secure store.
func (id *Identity) Marshal() []byte {
	out := make([]byte, 0, marshaledSize)
	out = append(out, id.SigningPriv...)
	out = append(out, id.AgreementPriv.Bytes()...)
	return out
}

// Unmarshal reconstructs an Identity from a blob produced by Marshal.
func Unmarshal(b []byte) (*Identity, error) {
	if len(b) != marshaledSize {
		return nil, errors.New("identity: invalid marshaled length")
	}
	signingPriv := make(ed25519.PrivateKey, ed25519.PrivateKeySize)
	copy(signingPriv, b[:ed25519.PrivateKeySize])

	agreement, err := ecdh.X25519().NewPrivateKey(b[ed25519.PrivateKeySize:])
	if err != nil {
		return nil, err
	}
	signingPub, ok := signingPriv.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("identity: unexpected public key type")
	}
	return &Identity{
		SigningPub:    signingPub,
		SigningPriv:   signingPriv,
		AgreementPriv: agreement,
	}, nil
}
