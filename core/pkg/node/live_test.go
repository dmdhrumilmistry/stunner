package node

import (
	"testing"
	"time"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/signaling"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

// TestLiveTwoDeviceDelivery proves the real two-device path: two nodes establish
// a session over an actual pion WebRTC data channel (loopback, host candidates —
// hermetic, no STUN/TURN needed) negotiated via the in-process signaler, complete
// the interactive bundle/handshake exchange, and exchange E2E-encrypted messages
// in both directions. This is the pure-P2P delivery: bytes flow node-to-node over
// the data channel, never through a server.
func TestLiveTwoDeviceDelivery(t *testing.T) {
	alice, bob := newNode(t), newNode(t)

	// No ICE servers: loopback uses host candidates, keeping the test hermetic.
	trA, err := transport.New(transport.Config{})
	if err != nil {
		t.Fatalf("transport A: %v", err)
	}
	trB, err := transport.New(transport.Config{})
	if err != nil {
		t.Fatalf("transport B: %v", err)
	}

	reg := signaling.NewRegistry()
	sigA := reg.Join(alice.Account.Fingerprint())
	sigB := reg.Join(bob.Account.Fingerprint())
	salt := []byte("stunner-test-rendezvous")

	type result struct {
		link *Link
		err  error
	}
	// Advertise Bob before Alice looks him up (the in-process registry has no
	// propagation delay; this just removes the goroutine-start race). Listen
	// re-advertises idempotently.
	if err := sigB.Advertise(identity.DiscoveryKey(bob.Account.Identity.SigningPub, salt)); err != nil {
		t.Fatalf("advertise: %v", err)
	}

	bobCh := make(chan result, 1)
	go func() {
		l, err := bob.Listen(trB, sigB, salt)
		bobCh <- result{l, err}
	}()

	linkA, err := alice.Connect(trA, sigA, bob.Account.Identity.SigningPub, salt)
	if err != nil {
		t.Fatalf("alice connect: %v", err)
	}
	defer linkA.Close()

	var rb result
	select {
	case rb = <-bobCh:
	case <-time.After(45 * time.Second):
		t.Fatal("bob listen timed out")
	}
	if rb.err != nil {
		t.Fatalf("bob listen: %v", rb.err)
	}
	linkB := rb.link
	defer linkB.Close()

	// Both sides agree on each other's identity.
	if linkA.PeerFingerprint() != bob.Account.Fingerprint() {
		t.Errorf("alice sees peer %s, want %s", linkA.PeerFingerprint(), bob.Account.Fingerprint())
	}
	if linkB.PeerFingerprint() != alice.Account.Fingerprint() {
		t.Errorf("bob sees peer %s, want %s", linkB.PeerFingerprint(), alice.Account.Fingerprint())
	}

	// Alice -> Bob, E2E over the live data channel.
	if _, err := linkA.SendText(alice, "conv1", "hello over webrtc 🔒"); err != nil {
		t.Fatalf("alice send: %v", err)
	}
	env, err := linkB.Receive()
	if err != nil {
		t.Fatalf("bob receive: %v", err)
	}
	if body, _ := env.Text(); body.Text != "hello over webrtc 🔒" {
		t.Errorf("bob got %q", body.Text)
	}

	// Bob -> Alice, proving the ratchet works in both directions.
	if _, err := linkB.SendText(bob, "conv1", "got it 👍"); err != nil {
		t.Fatalf("bob send: %v", err)
	}
	env, err = linkA.Receive()
	if err != nil {
		t.Fatalf("alice receive: %v", err)
	}
	if body, _ := env.Text(); body.Text != "got it 👍" {
		t.Errorf("alice got %q", body.Text)
	}
}

// TestLiveConnectRejectsIdentityMismatch verifies the MITM guard: if the peer
// that answers presents a different identity than the one we expected (e.g. a
// signaling-layer attacker substituting their own endpoint), Connect aborts with
// ErrIdentityMismatch and no session is formed.
func TestLiveConnectRejectsIdentityMismatch(t *testing.T) {
	alice, bob, mallory := newNode(t), newNode(t), newNode(t)

	trA, _ := transport.New(transport.Config{})
	trM, _ := transport.New(transport.Config{})

	reg := signaling.NewRegistry()
	sigA := reg.Join(alice.Account.Fingerprint())
	sigM := reg.Join(mallory.Account.Fingerprint())
	salt := []byte("salt")

	// Mallory answers on Bob's discovery key (a signaling-layer substitution).
	if err := sigM.Advertise(identity.DiscoveryKey(bob.Account.Identity.SigningPub, salt)); err != nil {
		t.Fatalf("advertise: %v", err)
	}
	go func() {
		conn, err := trM.Accept(sigM)
		if err != nil {
			return
		}
		_, _ = mallory.handshakeResponder(conn)
	}()

	_, err := alice.Connect(trA, sigA, bob.Account.Identity.SigningPub, salt)
	if err != ErrIdentityMismatch {
		t.Fatalf("expected ErrIdentityMismatch, got %v", err)
	}
}
