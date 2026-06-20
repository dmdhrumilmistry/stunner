package libsignal

import (
	"context"

	"go.mau.fi/libsignal/keys/identity"
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/serialize"
	"go.mau.fi/libsignal/state/record"
)

// In-memory implementations of the libsignal store interfaces. libsignal ships
// only the interfaces (state/store); these straightforward maps back a single
// account in memory. Persisting them via pkg/storage is a later integration.

// --- IdentityKeyStore ---

type identityStore struct {
	trusted map[*protocol.SignalAddress]*identity.Key
	keyPair *identity.KeyPair
	regID   uint32
}

func newIdentityStore(kp *identity.KeyPair, regID uint32) *identityStore {
	return &identityStore{trusted: map[*protocol.SignalAddress]*identity.Key{}, keyPair: kp, regID: regID}
}

func (s *identityStore) GetIdentityKeyPair() *identity.KeyPair { return s.keyPair }
func (s *identityStore) GetLocalRegistrationID() uint32        { return s.regID }

func (s *identityStore) SaveIdentity(_ context.Context, address *protocol.SignalAddress, key *identity.Key) error {
	s.trusted[address] = key
	return nil
}

func (s *identityStore) IsTrustedIdentity(_ context.Context, address *protocol.SignalAddress, key *identity.Key) (bool, error) {
	t := s.trusted[address]
	return t == nil || t.Fingerprint() == key.Fingerprint(), nil
}

// --- PreKeyStore ---

type preKeyStore struct{ store map[uint32]*record.PreKey }

func newPreKeyStore() *preKeyStore { return &preKeyStore{store: map[uint32]*record.PreKey{}} }

func (s *preKeyStore) LoadPreKey(_ context.Context, id uint32) (*record.PreKey, error) {
	return s.store[id], nil
}
func (s *preKeyStore) StorePreKey(_ context.Context, id uint32, r *record.PreKey) error {
	s.store[id] = r
	return nil
}
func (s *preKeyStore) ContainsPreKey(_ context.Context, id uint32) (bool, error) {
	_, ok := s.store[id]
	return ok, nil
}
func (s *preKeyStore) RemovePreKey(_ context.Context, id uint32) error {
	delete(s.store, id)
	return nil
}

// --- SignedPreKeyStore ---

type signedPreKeyStore struct {
	store map[uint32]*record.SignedPreKey
}

func newSignedPreKeyStore() *signedPreKeyStore {
	return &signedPreKeyStore{store: map[uint32]*record.SignedPreKey{}}
}

func (s *signedPreKeyStore) LoadSignedPreKey(_ context.Context, id uint32) (*record.SignedPreKey, error) {
	return s.store[id], nil
}
func (s *signedPreKeyStore) LoadSignedPreKeys(_ context.Context) ([]*record.SignedPreKey, error) {
	out := make([]*record.SignedPreKey, 0, len(s.store))
	for _, r := range s.store {
		out = append(out, r)
	}
	return out, nil
}
func (s *signedPreKeyStore) StoreSignedPreKey(_ context.Context, id uint32, r *record.SignedPreKey) error {
	s.store[id] = r
	return nil
}
func (s *signedPreKeyStore) ContainsSignedPreKey(_ context.Context, id uint32) (bool, error) {
	_, ok := s.store[id]
	return ok, nil
}
func (s *signedPreKeyStore) RemoveSignedPreKey(_ context.Context, id uint32) error {
	delete(s.store, id)
	return nil
}

// --- SessionStore ---

type sessionStore struct {
	sessions   map[*protocol.SignalAddress]*record.Session
	serializer *serialize.Serializer
}

func newSessionStore(s *serialize.Serializer) *sessionStore {
	return &sessionStore{sessions: map[*protocol.SignalAddress]*record.Session{}, serializer: s}
}

func (s *sessionStore) LoadSession(ctx context.Context, address *protocol.SignalAddress) (*record.Session, error) {
	if ok, _ := s.ContainsSession(ctx, address); ok {
		return s.sessions[address], nil
	}
	r := record.NewSession(s.serializer.Session, s.serializer.State)
	s.sessions[address] = r
	return r, nil
}

func (s *sessionStore) GetSubDeviceSessions(_ context.Context, name string) ([]uint32, error) {
	var ids []uint32
	for k := range s.sessions {
		if k.Name() == name && k.DeviceID() != 1 {
			ids = append(ids, k.DeviceID())
		}
	}
	return ids, nil
}

func (s *sessionStore) StoreSession(_ context.Context, address *protocol.SignalAddress, r *record.Session) error {
	s.sessions[address] = r
	return nil
}

func (s *sessionStore) ContainsSession(_ context.Context, address *protocol.SignalAddress) (bool, error) {
	_, ok := s.sessions[address]
	return ok, nil
}

func (s *sessionStore) DeleteSession(_ context.Context, address *protocol.SignalAddress) error {
	delete(s.sessions, address)
	return nil
}

func (s *sessionStore) DeleteAllSessions(_ context.Context) error {
	s.sessions = map[*protocol.SignalAddress]*record.Session{}
	return nil
}
