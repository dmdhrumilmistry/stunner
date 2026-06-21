// Package node is the Stunner runtime: it wires an account's identity and
// sessions to a transport and (optionally) the offline mailbox, exposing the
// operations the app drives — establish a connection, send/receive messages,
// transfer files, and queue messages for offline peers.
//
// The reference wiring uses the in-process transport (transport.Pipe) and
// in-memory signaling/mailbox so the full pipeline runs and is tested without
// networking; the pion/libp2p backends slot in behind the same interfaces.
package node

import (
	"crypto/ed25519"
	"errors"
	"sync"

	"github.com/dmdhrumilmistry/stunner/core/pkg/account"
	"github.com/dmdhrumilmistry/stunner/core/pkg/crypto"
	"github.com/dmdhrumilmistry/stunner/core/pkg/filetransfer"
	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/mailbox"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/storage"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

// Node is a running Stunner instance for one account.
type Node struct {
	Account  *account.Account
	Store    storage.Store
	sessions crypto.SessionStore
}

// New creates a node for an account. store may be nil for ephemeral use.
func New(acc *account.Account, store storage.Store) *Node {
	return &Node{Account: acc, Store: store, sessions: acc.Sessions()}
}

// Bundle returns this node's publishable prekey bundle (for peers to Dial).
func (n *Node) Bundle() crypto.PreKeyBundle { return n.Account.PreKeys.Bundle() }

// Link is an established, encrypted connection to one peer over a transport.
type Link struct {
	conn    transport.Conn
	session crypto.Session
	peerFP  string
	peerKey ed25519.PublicKey

	// sendMu serializes encrypt+send so the Double Ratchet chain advances
	// atomically and multi-frame sends (a file's chunks) aren't interleaved
	// with other sends (receipts, typing) on the same link.
	sendMu sync.Mutex
}

// PeerFingerprint returns the connected peer's identity fingerprint.
func (l *Link) PeerFingerprint() string { return l.peerFP }

// PeerIdentityKey returns the connected peer's Ed25519 identity public key,
// learned during the handshake. It lets a responder reconstruct the peer's
// contact URI so an inbound-only peer can be replied to and saved.
func (l *Link) PeerIdentityKey() ed25519.PublicKey { return l.peerKey }

// Close tears down the link.
func (l *Link) Close() error { return l.conn.Close() }

// Dial establishes a session with a peer (whose bundle we have) over conn,
// sending the X3DH handshake as the first frame.
func (n *Node) Dial(conn transport.Conn, bundle crypto.PreKeyBundle) (*Link, error) {
	session, hs, err := n.sessions.Initiate(bundle)
	if err != nil {
		return nil, err
	}
	frame, err := messaging.EncodeFrame(messaging.Frame{Handshake: &hs})
	if err != nil {
		return nil, err
	}
	if err := conn.Send(frame); err != nil {
		return nil, err
	}
	return &Link{conn: conn, session: session, peerFP: session.PeerFingerprint(), peerKey: bundle.IdentitySign}, nil
}

// Accept receives an incoming handshake frame on conn and establishes a session.
func (n *Node) Accept(conn transport.Conn) (*Link, error) {
	raw, err := conn.Recv()
	if err != nil {
		return nil, err
	}
	frame, err := messaging.DecodeFrame(raw)
	if err != nil {
		return nil, err
	}
	if frame.Handshake == nil {
		return nil, errors.New("node: first frame missing handshake")
	}
	session, err := n.sessions.Accept(*frame.Handshake)
	if err != nil {
		return nil, err
	}
	return &Link{conn: conn, session: session, peerFP: session.PeerFingerprint(), peerKey: frame.Handshake.IdentitySign}, nil
}

// encryptSendLocked encrypts and writes one envelope. Caller must hold sendMu.
func (l *Link) encryptSendLocked(env messaging.Envelope) error {
	pt, err := env.Encode()
	if err != nil {
		return err
	}
	ct, err := l.session.Encrypt(pt)
	if err != nil {
		return err
	}
	frame, err := messaging.EncodeFrame(messaging.Frame{Payload: ct})
	if err != nil {
		return err
	}
	return l.conn.Send(frame)
}

// persistable reports whether an envelope type is a stored conversation message
// (as opposed to a transport/ephemeral control frame).
func persistable(t messaging.Type) bool {
	return t == messaging.TypeText || t == messaging.TypeFileOffer
}

func (l *Link) sendEnvelopeLocked(n *Node, env messaging.Envelope) error {
	if err := l.encryptSendLocked(env); err != nil {
		return err
	}
	if n.Store != nil && persistable(env.Type) {
		_ = n.Store.AppendMessage(env.ConvID, env, messaging.StateSent)
	}
	return nil
}

// SendEnvelope encrypts and sends an application envelope, persisting it to the
// store when one is configured (and the type is a conversation message).
func (l *Link) SendEnvelope(n *Node, env messaging.Envelope) error {
	l.sendMu.Lock()
	defer l.sendMu.Unlock()
	return l.sendEnvelopeLocked(n, env)
}

// SendText sends a text message.
func (l *Link) SendText(n *Node, convID, text string) (messaging.Envelope, error) {
	env, err := messaging.NewText(convID, text)
	if err != nil {
		return messaging.Envelope{}, err
	}
	return env, l.SendEnvelope(n, env)
}

// SendTyping sends an ephemeral typing indicator (not persisted, best effort).
func (l *Link) SendTyping(convID string) error {
	l.sendMu.Lock()
	defer l.sendMu.Unlock()
	return l.encryptSendLocked(messaging.NewEnvelope(messaging.TypeTyping, convID, nil))
}

// Receive reads and decrypts the next envelope from the peer.
func (l *Link) Receive() (messaging.Envelope, error) {
	raw, err := l.conn.Recv()
	if err != nil {
		return messaging.Envelope{}, err
	}
	frame, err := messaging.DecodeFrame(raw)
	if err != nil {
		return messaging.Envelope{}, err
	}
	pt, err := l.session.Decrypt(frame.Payload)
	if err != nil {
		return messaging.Envelope{}, err
	}
	return messaging.DecodeEnvelope(pt)
}

// SendOffline encrypts an envelope for a peer (using their published bundle) and
// queues it in the mailbox for later delivery — the pure-P2P offline path. The
// recipient retrieves it with FetchOffline.
func (n *Node) SendOffline(mb mailbox.Mailbox, bundle crypto.PreKeyBundle, env messaging.Envelope) error {
	session, hs, err := n.sessions.Initiate(bundle)
	if err != nil {
		return err
	}
	pt, err := env.Encode()
	if err != nil {
		return err
	}
	ct, err := session.Encrypt(pt)
	if err != nil {
		return err
	}
	frame, err := messaging.EncodeFrame(messaging.Frame{Handshake: &hs, Payload: ct})
	if err != nil {
		return err
	}
	return mb.Put(identity.Fingerprint(bundle.IdentitySign), frame)
}

// FetchOffline retrieves and decrypts all messages queued for this node.
func (n *Node) FetchOffline(mb mailbox.Mailbox) ([]messaging.Envelope, error) {
	raws, err := mb.Fetch(n.Account.Fingerprint())
	if err != nil {
		return nil, err
	}
	var out []messaging.Envelope
	for _, raw := range raws {
		frame, err := messaging.DecodeFrame(raw)
		if err != nil {
			return nil, err
		}
		if frame.Handshake == nil {
			return nil, errors.New("node: offline message missing handshake")
		}
		session, err := n.sessions.Accept(*frame.Handshake)
		if err != nil {
			return nil, err
		}
		pt, err := session.Decrypt(frame.Payload)
		if err != nil {
			return nil, err
		}
		env, err := messaging.DecodeEnvelope(pt)
		if err != nil {
			return nil, err
		}
		out = append(out, env)
	}
	return out, nil
}

// SendFile splits data into sealed chunks and sends the offer plus all chunks as
// envelopes over the link. Returns the offer that describes the transfer.
func (l *Link) SendFile(n *Node, convID, name, mime string, data []byte) (filetransfer.Offer, error) {
	offer, chunks, err := filetransfer.Split(name, mime, data, 0)
	if err != nil {
		return filetransfer.Offer{}, err
	}
	// Hold the lock across offer+chunks so no other send (receipt/typing)
	// interleaves between them and breaks the receiver's chunk reassembly.
	l.sendMu.Lock()
	defer l.sendMu.Unlock()
	if err := l.sendEnvelopeLocked(n, envFor(messaging.TypeFileOffer, convID, offer)); err != nil {
		return filetransfer.Offer{}, err
	}
	for _, c := range chunks {
		if err := l.sendEnvelopeLocked(n, envFor(messaging.TypeFileChunk, convID, c)); err != nil {
			return filetransfer.Offer{}, err
		}
	}
	return offer, nil
}

func envFor(t messaging.Type, convID string, v any) messaging.Envelope {
	body := mustJSON(v)
	return messaging.NewEnvelope(t, convID, body)
}
