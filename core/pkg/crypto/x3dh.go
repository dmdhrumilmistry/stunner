package crypto

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"errors"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
)

// PreKeyBundle is the public material a peer publishes so others can start a
// session with them via X3DH (see docs/PROTOCOL.md §3).
type PreKeyBundle struct {
	IdentitySign    ed25519.PublicKey `json:"identitySign"` // verifies the signature & forms the fingerprint
	IdentityDH      []byte            `json:"identityDH"`   // X25519 identity key (32)
	SignedPreKey    []byte            `json:"signedPreKey"` // X25519 (32)
	SignedPreKeySig []byte            `json:"signedPreKeySig"`
	OneTimePreKey   []byte            `json:"oneTimePreKey,omitempty"` // X25519 (32), optional
}

// PreKeys holds a responder's prekey private material. Generate one per account
// and republish Bundle() as one-time keys are consumed.
type PreKeys struct {
	id           *identity.Identity
	signedPre    *ecdh.PrivateKey
	signedPreSig []byte
	oneTime      *ecdh.PrivateKey
}

// GeneratePreKeys creates a signed prekey (signed by the Ed25519 identity) and
// one one-time prekey for the given identity.
func GeneratePreKeys(id *identity.Identity) (*PreKeys, error) {
	signedPre, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	oneTime, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	sig, err := id.Sign(signedPre.PublicKey().Bytes())
	if err != nil {
		return nil, err
	}
	return &PreKeys{id: id, signedPre: signedPre, signedPreSig: sig, oneTime: oneTime}, nil
}

// Bundle returns the publishable public bundle.
func (p *PreKeys) Bundle() PreKeyBundle {
	return PreKeyBundle{
		IdentitySign:    p.id.SigningPub,
		IdentityDH:      p.id.AgreementPub().Bytes(),
		SignedPreKey:    p.signedPre.PublicKey().Bytes(),
		SignedPreKeySig: p.signedPreSig,
		OneTimePreKey:   p.oneTime.PublicKey().Bytes(),
	}
}

// x3dhInitiate runs X3DH as the initiator against a peer's bundle, returning the
// shared secret and the ephemeral public key to send to the peer.
func x3dhInitiate(local *identity.Identity, b PreKeyBundle) (sk, ephPub []byte, usedOTP bool, err error) {
	if !identity.Verify(b.IdentitySign, b.SignedPreKey, b.SignedPreKeySig) {
		return nil, nil, false, errors.New("crypto: bad signed prekey signature")
	}
	eph, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, false, err
	}
	spk, err := ecdh.X25519().NewPublicKey(b.SignedPreKey)
	if err != nil {
		return nil, nil, false, err
	}
	ikB, err := ecdh.X25519().NewPublicKey(b.IdentityDH)
	if err != nil {
		return nil, nil, false, err
	}
	dh1, _ := local.AgreementPriv.ECDH(spk)
	dh2, _ := eph.ECDH(ikB)
	dh3, _ := eph.ECDH(spk)
	concat := append(append(append([]byte{}, dh1...), dh2...), dh3...)
	if len(b.OneTimePreKey) > 0 {
		opk, err := ecdh.X25519().NewPublicKey(b.OneTimePreKey)
		if err != nil {
			return nil, nil, false, err
		}
		dh4, _ := eph.ECDH(opk)
		concat = append(concat, dh4...)
		usedOTP = true
	}
	return deriveX3DH(concat), eph.PublicKey().Bytes(), usedOTP, nil
}

// x3dhRespond runs X3DH as the responder, reconstructing the shared secret.
func x3dhRespond(pre *PreKeys, initiatorIK, ephPub []byte, usedOTP bool) ([]byte, error) {
	ikA, err := ecdh.X25519().NewPublicKey(initiatorIK)
	if err != nil {
		return nil, err
	}
	eph, err := ecdh.X25519().NewPublicKey(ephPub)
	if err != nil {
		return nil, err
	}
	dh1, _ := pre.signedPre.ECDH(ikA)
	dh2, _ := pre.id.AgreementPriv.ECDH(eph)
	dh3, _ := pre.signedPre.ECDH(eph)
	concat := append(append(append([]byte{}, dh1...), dh2...), dh3...)
	if usedOTP {
		dh4, _ := pre.oneTime.ECDH(eph)
		concat = append(concat, dh4...)
	}
	return deriveX3DH(concat), nil
}

// deriveX3DH applies the X3DH KDF: HKDF over (0xFF padding || DH concatenation).
func deriveX3DH(concat []byte) []byte {
	ikm := append(make([]byte, 32), concat...)
	for i := 0; i < 32; i++ {
		ikm[i] = 0xFF
	}
	return hkdf(make([]byte, 32), ikm, []byte("stunner-x3dh"), 32)
}
