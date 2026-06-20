// Package libsignal provides a session backed by the maintained
// go.mau.fi/libsignal implementation of the Signal protocol (X3DH + Double
// Ratchet). It is the audited-library alternative to the from-scratch ratchet in
// the parent crypto package, and its Session satisfies crypto.Session.
//
// Establishment mirrors Signal: a Participant publishes a PreKeyBundle; the
// initiator processes it (NewOutboundSession) and its first Encrypt emits a
// PreKeySignalMessage (the handshake); the responder (NewInboundSession)
// establishes the session on first Decrypt. Encrypt/Decrypt tag each message
// with its type so callers exchange opaque bytes.
package libsignal

import (
	"context"
	"errors"

	"go.mau.fi/libsignal/keys/identity"
	"go.mau.fi/libsignal/keys/prekey"
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/serialize"
	"go.mau.fi/libsignal/session"
	"go.mau.fi/libsignal/state/record"
	"go.mau.fi/libsignal/util/keyhelper"
)

const (
	tagPreKey byte = 'p'
	tagSignal byte = 's'
)

// Participant is a local libsignal account: identity, prekeys, and stores.
type Participant struct {
	name     string
	deviceID uint32
	address  *protocol.SignalAddress

	identityKeyPair *identity.KeyPair
	registrationID  uint32
	preKeys         []*record.PreKey
	signedPreKey    *record.SignedPreKey
	serializer      *serialize.Serializer

	sessionStore      *sessionStore
	preKeyStore       *preKeyStore
	signedPreKeyStore *signedPreKeyStore
	identityStore     *identityStore
}

// NewParticipant generates a new account with one signed prekey and a batch of
// one-time prekeys. name/deviceID form its address (name should be stable, e.g.
// the identity fingerprint).
func NewParticipant(name string, deviceID uint32) (*Participant, error) {
	serializer := serialize.NewProtoBufSerializer()

	idKeyPair, err := keyhelper.GenerateIdentityKeyPair()
	if err != nil {
		return nil, err
	}
	regID := keyhelper.GenerateRegistrationID()
	preKeys, err := keyhelper.GeneratePreKeys(1, 100, serializer.PreKeyRecord)
	if err != nil {
		return nil, err
	}
	signedPreKey, err := keyhelper.GenerateSignedPreKey(idKeyPair, 0, serializer.SignedPreKeyRecord)
	if err != nil {
		return nil, err
	}

	p := &Participant{
		name:              name,
		deviceID:          deviceID,
		address:           protocol.NewSignalAddress(name, deviceID),
		identityKeyPair:   idKeyPair,
		registrationID:    regID,
		preKeys:           preKeys,
		signedPreKey:      signedPreKey,
		serializer:        serializer,
		sessionStore:      newSessionStore(serializer),
		preKeyStore:       newPreKeyStore(),
		signedPreKeyStore: newSignedPreKeyStore(),
		identityStore:     newIdentityStore(idKeyPair, regID),
	}

	ctx := context.Background()
	for i := range preKeys {
		_ = p.preKeyStore.StorePreKey(ctx, preKeys[i].ID().Value,
			record.NewPreKey(preKeys[i].ID().Value, preKeys[i].KeyPair(), serializer.PreKeyRecord))
	}
	_ = p.signedPreKeyStore.StoreSignedPreKey(ctx, signedPreKey.ID(),
		record.NewSignedPreKey(signedPreKey.ID(), signedPreKey.Timestamp(),
			signedPreKey.KeyPair(), signedPreKey.Signature(), serializer.SignedPreKeyRecord))

	return p, nil
}

// Fingerprint returns the participant's identity-key fingerprint.
func (p *Participant) Fingerprint() string { return p.identityKeyPair.PublicKey().Fingerprint() }

// Bundle returns the publishable prekey bundle for peers to start a session.
func (p *Participant) Bundle() *prekey.Bundle {
	return prekey.NewBundle(
		p.registrationID,
		p.deviceID,
		p.preKeys[0].ID(),
		p.signedPreKey.ID(),
		p.preKeys[0].KeyPair().PublicKey(),
		p.signedPreKey.KeyPair().PublicKey(),
		p.signedPreKey.Signature(),
		p.identityKeyPair.PublicKey(),
	)
}

func (p *Participant) builder(remote *protocol.SignalAddress) *session.Builder {
	return session.NewBuilder(p.sessionStore, p.preKeyStore, p.signedPreKeyStore, p.identityStore, remote, p.serializer)
}

// NewOutboundSession processes a peer's bundle and returns an initiator session.
func (p *Participant) NewOutboundSession(remoteName string, remoteDeviceID uint32, bundle *prekey.Bundle) (*Session, error) {
	remote := protocol.NewSignalAddress(remoteName, remoteDeviceID)
	b := p.builder(remote)
	if err := b.ProcessBundle(context.Background(), bundle); err != nil {
		return nil, err
	}
	return &Session{
		cipher:     session.NewCipher(b, remote),
		serializer: p.serializer,
		peerFP:     bundle.IdentityKey().Fingerprint(),
	}, nil
}

// NewInboundSession returns a responder session for a peer; the session is
// established on the first Decrypt of a PreKeySignalMessage.
func (p *Participant) NewInboundSession(remoteName string, remoteDeviceID uint32) *Session {
	remote := protocol.NewSignalAddress(remoteName, remoteDeviceID)
	b := p.builder(remote)
	return &Session{cipher: session.NewCipher(b, remote), serializer: p.serializer}
}

// Session is an established libsignal session. It implements crypto.Session.
type Session struct {
	cipher     *session.Cipher
	serializer *serialize.Serializer
	peerFP     string
}

// Encrypt seals plaintext, tagging the wire bytes with the message type.
func (s *Session) Encrypt(plaintext []byte) ([]byte, error) {
	msg, err := s.cipher.Encrypt(context.Background(), plaintext)
	if err != nil {
		return nil, err
	}
	tag := tagSignal
	if msg.Type() == protocol.PREKEY_TYPE {
		tag = tagPreKey
	}
	return append([]byte{tag}, msg.Serialize()...), nil
}

// Decrypt opens a tagged wire message from the peer.
func (s *Session) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 1 {
		return nil, errors.New("libsignal: empty ciphertext")
	}
	body := ciphertext[1:]
	ctx := context.Background()
	switch ciphertext[0] {
	case tagPreKey:
		m, err := protocol.NewPreKeySignalMessageFromBytes(body, s.serializer.PreKeySignalMessage, s.serializer.SignalMessage)
		if err != nil {
			return nil, err
		}
		return s.cipher.DecryptMessage(ctx, m)
	case tagSignal:
		m, err := protocol.NewSignalMessageFromBytes(body, s.serializer.SignalMessage)
		if err != nil {
			return nil, err
		}
		return s.cipher.Decrypt(ctx, m)
	default:
		return nil, errors.New("libsignal: unknown message tag")
	}
}

// PeerFingerprint returns the peer's identity fingerprint (known after the
// session is established; empty for a fresh inbound session before first
// decrypt).
func (s *Session) PeerFingerprint() string { return s.peerFP }
