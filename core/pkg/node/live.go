package node

import (
	"bytes"
	"crypto/ed25519"
	"errors"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/signaling"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

// ErrIdentityMismatch is returned by Connect when the peer that answered does
// not present the identity key we expected — a potential man-in-the-middle. The
// handshake is aborted and no session is established.
var ErrIdentityMismatch = errors.New("node: peer identity does not match expected contact")

// Connect establishes a live, end-to-end-encrypted link to a peer over a real
// transport, discovering them via the signaler. This is the two-device delivery
// path: messages travel directly peer-to-peer over the transport's data channel
// (WebRTC), negotiated using the configured STUN/TURN ICE servers.
//
// It is pure P2P: STUN is used only to discover each side's public address and
// punch a hole; the message bytes never pass through it. TURN relays bytes only
// if direct connectivity fails AND a TURN server is configured (none is by
// default — see settings.DefaultICEServers), and even then it only ever sees
// E2E ciphertext.
//
// The handshake is interactive so no published prekey directory is needed: the
// peer (acting as responder via Listen) sends its prekey bundle over the freshly
// opened channel; this side verifies the bundle is bound to peerIdentity (the
// Ed25519 key from the contact's URI/QR), then runs X3DH and replies with the
// handshake. The bundle's signed prekey is in turn verified to be signed by
// peerIdentity inside X3DH, so a substituted bundle cannot establish a session.
func (n *Node) Connect(t transport.Transport, sig signaling.Signaler, peerIdentity ed25519.PublicKey, rendezvousSalt []byte) (*Link, error) {
	peer, err := sig.Find(identity.DiscoveryKey(peerIdentity, rendezvousSalt))
	if err != nil {
		return nil, err
	}
	conn, err := t.Dial(peer.PeerID, sig)
	if err != nil {
		return nil, err
	}
	link, err := n.handshakeInitiator(conn, peerIdentity)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return link, nil
}

// Listen advertises this node under its discovery key and accepts one inbound
// live link, completing the interactive handshake as the responder. Pair it with
// a peer calling Connect. Call it again to accept the next peer.
func (n *Node) Listen(t transport.Transport, sig signaling.Signaler, rendezvousSalt []byte) (*Link, error) {
	if err := sig.Advertise(identity.DiscoveryKey(n.Account.Identity.SigningPub, rendezvousSalt)); err != nil {
		return nil, err
	}
	conn, err := t.Accept(sig)
	if err != nil {
		return nil, err
	}
	link, err := n.handshakeResponder(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return link, nil
}

// handshakeInitiator runs the initiator side of the interactive handshake over
// an already-established conn: receive the peer's bundle, verify it matches the
// expected identity, run X3DH, and send the handshake.
func (n *Node) handshakeInitiator(conn transport.Conn, peerIdentity ed25519.PublicKey) (*Link, error) {
	raw, err := conn.Recv()
	if err != nil {
		return nil, err
	}
	frame, err := messaging.DecodeFrame(raw)
	if err != nil {
		return nil, err
	}
	if frame.Bundle == nil {
		return nil, errors.New("node: first frame missing prekey bundle")
	}
	if !bytes.Equal(frame.Bundle.IdentitySign, peerIdentity) {
		return nil, ErrIdentityMismatch
	}
	session, hs, err := n.sessions.Initiate(*frame.Bundle)
	if err != nil {
		return nil, err
	}
	out, err := messaging.EncodeFrame(messaging.Frame{Handshake: &hs})
	if err != nil {
		return nil, err
	}
	if err := conn.Send(out); err != nil {
		return nil, err
	}
	return &Link{conn: conn, session: session, peerFP: session.PeerFingerprint()}, nil
}

// handshakeResponder runs the responder side over an already-established conn:
// send our bundle, then receive and accept the initiator's handshake.
func (n *Node) handshakeResponder(conn transport.Conn) (*Link, error) {
	bundle := n.Bundle()
	out, err := messaging.EncodeFrame(messaging.Frame{Bundle: &bundle})
	if err != nil {
		return nil, err
	}
	if err := conn.Send(out); err != nil {
		return nil, err
	}
	raw, err := conn.Recv()
	if err != nil {
		return nil, err
	}
	frame, err := messaging.DecodeFrame(raw)
	if err != nil {
		return nil, err
	}
	if frame.Handshake == nil {
		return nil, errors.New("node: missing handshake from initiator")
	}
	session, err := n.sessions.Accept(*frame.Handshake)
	if err != nil {
		return nil, err
	}
	return &Link{conn: conn, session: session, peerFP: session.PeerFingerprint()}, nil
}
