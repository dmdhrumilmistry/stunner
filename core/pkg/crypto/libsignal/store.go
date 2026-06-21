package libsignal

import "os"

// Enabled reports whether the libsignal backend has been opted into via
// STUNNER_CRYPTO_LIBSIGNAL=1.
//
// The backend is wire-ready (see Store + SerializeBundle) and verified by tests,
// but it is NOT yet the live default. Adopting it for pkg/node requires two
// follow-ups that are intentionally out of scope here:
//   - binding the libsignal identity key to the account's Ed25519 identity so
//     contact URIs, fingerprints and safety numbers stay stable; and
//   - switching node's two-phase (bundle → handshake) exchange to libsignal's
//     model (the key exchange rides in the first PreKeySignalMessage).
//
// Until then this flag only gates experimental/local use. See docs/ROADMAP.md.
func Enabled() bool { return os.Getenv("STUNNER_CRYPTO_LIBSIGNAL") == "1" }

// Store is a libsignal-backed account: it publishes a wire-serializable prekey
// bundle and establishes forward-secret, post-compromise-secure sessions. It is
// the production-crypto counterpart to crypto.NewMemStore (the unaudited
// from-scratch reference ratchet).
type Store struct{ p *Participant }

// NewStore creates a libsignal account. name should be a stable per-account
// label (e.g. the identity fingerprint); deviceID is usually 1.
func NewStore(name string, deviceID uint32) (*Store, error) {
	p, err := NewParticipant(name, deviceID)
	if err != nil {
		return nil, err
	}
	return &Store{p: p}, nil
}

// Fingerprint returns this account's identity-key fingerprint.
func (s *Store) Fingerprint() string { return s.p.Fingerprint() }

// PublishBundle returns this account's wire-serialized prekey bundle, for a peer
// to start a session with Initiate.
func (s *Store) PublishBundle() ([]byte, error) { return SerializeBundle(s.p.Bundle()) }

// Initiate starts an outbound session to a peer from their published (wire)
// bundle. The returned *Session implements crypto.Session; its first Encrypt
// produces the establishing PreKeySignalMessage.
func (s *Store) Initiate(peerName string, peerDeviceID uint32, peerBundle []byte) (*Session, error) {
	b, err := DeserializeBundle(peerBundle)
	if err != nil {
		return nil, err
	}
	return s.p.NewOutboundSession(peerName, peerDeviceID, b)
}

// Accept returns a responder session for a peer; it is established on the first
// Decrypt of the initiator's PreKeySignalMessage.
func (s *Store) Accept(peerName string, peerDeviceID uint32) *Session {
	return s.p.NewInboundSession(peerName, peerDeviceID)
}
