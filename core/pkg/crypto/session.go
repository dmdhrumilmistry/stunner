package crypto

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"sync"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
)

// Handshake is sent by the initiator as the first message so the responder can
// run X3DH and derive the same shared secret.
type Handshake struct {
	IdentitySign ed25519.PublicKey `json:"identitySign"`
	IdentityDH   []byte            `json:"identityDH"`
	Ephemeral    []byte            `json:"ephemeral"`
	UsedOTP      bool              `json:"usedOTP"`
}

// ratchetSession implements Session over a Double Ratchet.
type ratchetSession struct {
	mu      sync.Mutex
	r       *ratchet
	peerFP  string
	peerSig ed25519.PublicKey
}

func (s *ratchetSession) Encrypt(plaintext []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.r.encrypt(plaintext)
}

func (s *ratchetSession) Decrypt(ciphertext []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.r.decrypt(ciphertext)
}

func (s *ratchetSession) PeerFingerprint() string { return s.peerFP }

// associatedData binds both parties' identity keys into every message,
// preventing unknown-key-share attacks. It is symmetric (sorted) so both sides
// compute the same value.
func associatedData(a, b ed25519.PublicKey) []byte {
	if bytes.Compare(a, b) <= 0 {
		return append(append([]byte{}, a...), b...)
	}
	return append(append([]byte{}, b...), a...)
}

// SessionStore manages secure sessions keyed by peer fingerprint.
type SessionStore interface {
	// Get returns an existing session for a peer fingerprint, if any.
	Get(peerFP string) (Session, bool)
	// Initiate starts a session from a peer's published bundle, returning the
	// session and the handshake to send as the first message.
	Initiate(bundle PreKeyBundle) (Session, Handshake, error)
	// Accept creates a responder session from a received handshake.
	Accept(hs Handshake) (Session, error)
}

// memStore is an in-memory SessionStore. Session state can later be persisted
// via pkg/storage; this keeps the reference implementation self-contained.
type memStore struct {
	mu       sync.Mutex
	local    *identity.Identity
	pre      *PreKeys
	sessions map[string]*ratchetSession
}

// NewMemStore creates an in-memory session store for the given local identity
// and prekeys.
func NewMemStore(local *identity.Identity, pre *PreKeys) SessionStore {
	return &memStore{local: local, pre: pre, sessions: map[string]*ratchetSession{}}
}

func (m *memStore) Get(peerFP string) (Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[peerFP]
	return s, ok
}

func (m *memStore) Initiate(bundle PreKeyBundle) (Session, Handshake, error) {
	sk, ephPub, usedOTP, err := x3dhInitiate(m.local, bundle)
	if err != nil {
		return nil, Handshake{}, err
	}
	ad := associatedData(m.local.SigningPub, bundle.IdentitySign)
	r, err := newRatchetInitiator(sk, bundle.SignedPreKey, ad)
	if err != nil {
		return nil, Handshake{}, err
	}
	s := &ratchetSession{
		r:       r,
		peerFP:  identity.Fingerprint(bundle.IdentitySign),
		peerSig: bundle.IdentitySign,
	}
	m.mu.Lock()
	m.sessions[s.peerFP] = s
	m.mu.Unlock()

	hs := Handshake{
		IdentitySign: m.local.SigningPub,
		IdentityDH:   m.local.AgreementPub().Bytes(),
		Ephemeral:    ephPub,
		UsedOTP:      usedOTP,
	}
	return s, hs, nil
}

func (m *memStore) Accept(hs Handshake) (Session, error) {
	if m.pre == nil {
		return nil, errors.New("crypto: no prekeys to accept handshake")
	}
	sk, err := x3dhRespond(m.pre, hs.IdentityDH, hs.Ephemeral, hs.UsedOTP)
	if err != nil {
		return nil, err
	}
	ad := associatedData(m.local.SigningPub, hs.IdentitySign)
	r := newRatchetResponder(sk, m.pre.signedPre, ad)
	s := &ratchetSession{
		r:       r,
		peerFP:  identity.Fingerprint(hs.IdentitySign),
		peerSig: hs.IdentitySign,
	}
	m.mu.Lock()
	m.sessions[s.peerFP] = s
	m.mu.Unlock()
	return s, nil
}
