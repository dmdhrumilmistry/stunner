package session

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dmdhrumilmistry/stunner/core/pkg/account"
	"github.com/dmdhrumilmistry/stunner/core/pkg/signaling"
	"github.com/dmdhrumilmistry/stunner/core/pkg/storage"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

const testRendezvous = "stunner-session-test"

// --- in-process transport for the orchestration test -----------------------

// connHub pairs Dial and Accept across in-process memTransports keyed by the
// destination peer ID, modelling the real Transport's Dial(peerID)/Accept(sig)
// without WebRTC.
type connHub struct {
	mu        sync.Mutex
	listeners map[string]chan transport.Conn
}

func newConnHub() *connHub { return &connHub{listeners: map[string]chan transport.Conn{}} }

func (h *connHub) listener(peerID string) chan transport.Conn {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.listeners[peerID]
	if !ok {
		ch = make(chan transport.Conn, 8)
		h.listeners[peerID] = ch
	}
	return ch
}

type memTransport struct {
	hub    *connHub
	peerID string
	done   chan struct{}
	once   sync.Once
}

var _ transport.Transport = (*memTransport)(nil)

func (t *memTransport) Dial(peerID string, _ transport.SignalingExchange) (transport.Conn, error) {
	a, b := transport.Pipe()
	select {
	case t.hub.listener(peerID) <- b:
		return a, nil
	case <-t.done:
		return nil, transport.ErrClosed
	}
}

func (t *memTransport) Accept(_ transport.SignalingExchange) (transport.Conn, error) {
	select {
	case c := <-t.hub.listener(t.peerID):
		return c, nil
	case <-t.done:
		return nil, transport.ErrClosed
	}
}

func (t *memTransport) Close() error {
	t.once.Do(func() { close(t.done) })
	return nil
}

// --- helpers ----------------------------------------------------------------

func startMemRuntime(t *testing.T, name string, reg *signaling.Registry, hub *connHub) *Runtime {
	t.Helper()
	dir := t.TempDir()
	key := bytes.Repeat([]byte{name[0]}, 32)
	acc, err := account.LoadOrCreate(dir, key)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	store, err := storage.Open(storage.Options{Path: filepath.Join(dir, "db.bin"), Key: key})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	sig := reg.Join(name)
	bx, ok := sig.(signaling.BundleExchanger)
	if !ok {
		t.Fatal("memory signaler does not implement BundleExchanger")
	}
	tr := &memTransport{hub: hub, peerID: name, done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	rt, err := newRuntime(ctx, cancel, acc, store, sig, bx, tr, []byte(testRendezvous))
	if err != nil {
		t.Fatalf("newRuntime: %v", err)
	}
	return rt
}

func waitForKind(t *testing.T, ch <-chan Event, kind string, d time.Duration) Event {
	t.Helper()
	timeout := time.After(d)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == kind {
				return ev
			}
			if ev.Kind == "error" {
				t.Fatalf("unexpected error event while waiting for %q: %s", kind, ev.Err)
			}
		case <-timeout:
			t.Fatalf("timed out waiting for %q event", kind)
		}
	}
}

// --- tests ------------------------------------------------------------------

// TestRuntimeConnectAndMessage drives the full orchestration over the in-memory
// signaler and in-process transport: discover, exchange prekey bundle, dial,
// X3DH handshake, send a text, and receive it as an event on the peer.
//
// It also covers the "messages from non-contacts" requirement: bob never adds
// alice to his contact book, yet still receives her message (the core does not
// gate inbound messages on contact membership).
func TestRuntimeConnectAndMessage(t *testing.T) {
	reg := signaling.NewRegistry()
	hub := newConnHub()

	alice := startMemRuntime(t, "alice", reg, hub)
	defer alice.Stop()
	bob := startMemRuntime(t, "bob", reg, hub)
	defer bob.Stop()

	peerFP, err := alice.Connect(bob.acc.ContactURI("bob"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if peerFP != bob.acc.Fingerprint() {
		t.Fatalf("peerFP = %q, want %q", peerFP, bob.acc.Fingerprint())
	}

	if _, err := alice.SendText("conv1", peerFP, "hello bob 🔒"); err != nil {
		t.Fatalf("send: %v", err)
	}

	ev := waitForKind(t, bob.Events(), "message", 5*time.Second)
	if ev.Text != "hello bob 🔒" {
		t.Fatalf("received text = %q, want %q", ev.Text, "hello bob 🔒")
	}
	if ev.PeerFP != alice.acc.Fingerprint() {
		t.Fatalf("event PeerFP = %q, want alice %q", ev.PeerFP, alice.acc.Fingerprint())
	}

	// Non-contact assertion: bob received the message without ever having added
	// alice as a contact.
	if got := len(bob.acc.Contacts.List()); got != 0 {
		t.Fatalf("bob contact book has %d entries; message receipt must not depend on contacts", got)
	}
}

// TestRuntimeBidirectionalMessage verifies a reply flows back over the same link
// (alice receives bob's message after bob replies on the accepted link).
func TestRuntimeBidirectionalMessage(t *testing.T) {
	reg := signaling.NewRegistry()
	hub := newConnHub()

	alice := startMemRuntime(t, "alice", reg, hub)
	defer alice.Stop()
	bob := startMemRuntime(t, "bob", reg, hub)
	defer bob.Stop()

	if _, err := alice.Connect(bob.acc.ContactURI("bob")); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := alice.SendText("conv1", bob.acc.Fingerprint(), "ping"); err != nil {
		t.Fatalf("send ping: %v", err)
	}
	if ev := waitForKind(t, bob.Events(), "message", 5*time.Second); ev.Text != "ping" {
		t.Fatalf("bob got %q", ev.Text)
	}

	// bob replies on the link he accepted from alice.
	if _, err := bob.SendText("conv1", alice.acc.Fingerprint(), "pong"); err != nil {
		t.Fatalf("send pong: %v", err)
	}
	if ev := waitForKind(t, alice.Events(), "message", 5*time.Second); ev.Text != "pong" {
		t.Fatalf("alice got %q", ev.Text)
	}
}

// TestRuntimeDrainEvents covers the poll-based event drain the desktop FFI uses:
// it must return a non-nil slice and surface the inbound message.
func TestRuntimeDrainEvents(t *testing.T) {
	reg := signaling.NewRegistry()
	hub := newConnHub()

	alice := startMemRuntime(t, "alice", reg, hub)
	defer alice.Stop()
	bob := startMemRuntime(t, "bob", reg, hub)
	defer bob.Stop()

	peerFP, err := alice.Connect(bob.acc.ContactURI("bob"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := alice.SendText("conv1", peerFP, "drain me"); err != nil {
		t.Fatalf("send: %v", err)
	}

	// Poll until the message shows up (events arrive asynchronously).
	deadline := time.Now().Add(5 * time.Second)
	for {
		var found bool
		for _, ev := range bob.DrainEvents() {
			if ev.Kind == "message" && ev.Text == "drain me" {
				found = true
			}
		}
		if found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("drained events never contained the message")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// An empty drain must be a non-nil empty slice (so the FFI marshals "[]").
	if got := bob.DrainEvents(); got == nil {
		t.Fatal("DrainEvents returned nil; want non-nil empty slice")
	}
}

// TestRuntimeLive assembles the production path — real pion/WebRTC transport and
// libp2p DHT signaling — on loopback and round-trips a message. It is opt-in
// (set STUNNER_LIVE_TEST=1) because DHT content-routing propagation in a tiny
// two-node network is environment-sensitive and slow.
func TestRuntimeLive(t *testing.T) {
	if os.Getenv("STUNNER_LIVE_TEST") == "" {
		t.Skip("set STUNNER_LIVE_TEST=1 to run the live pion+DHT integration test")
	}

	aSig, err := signaling.NewDHT(context.Background())
	if err != nil {
		t.Fatalf("new dht a: %v", err)
	}
	bSig, err := signaling.NewDHT(context.Background())
	if err != nil {
		t.Fatalf("new dht b: %v", err)
	}
	// Bootstrap the two hosts directly so DHT records can propagate.
	if err := aSig.Connect(bSig.AddrInfo()); err != nil {
		t.Fatalf("bootstrap connect: %v", err)
	}

	alice := newLiveRuntime(t, "alice", aSig)
	defer alice.Stop()
	bob := newLiveRuntime(t, "bob", bSig)
	defer bob.Stop()

	bobURI := bob.acc.ContactURI("bob")

	// Retry Connect until the DHT resolves bob's advertisement.
	var peerFP string
	deadline := time.Now().Add(50 * time.Second)
	for {
		peerFP, err = alice.Connect(bobURI)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("connect never succeeded: %v", err)
		}
		time.Sleep(2 * time.Second)
	}

	if _, err := alice.SendText("conv1", peerFP, "live hello 🔒"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if ev := waitForKind(t, bob.Events(), "message", 30*time.Second); ev.Text != "live hello 🔒" {
		t.Fatalf("bob received %q", ev.Text)
	}
}

func newLiveRuntime(t *testing.T, name string, sig *signaling.DHTSignaler) *Runtime {
	t.Helper()
	dir := t.TempDir()
	key := bytes.Repeat([]byte{name[0]}, 32)
	acc, err := account.LoadOrCreate(dir, key)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	store, err := storage.Open(storage.Options{Path: filepath.Join(dir, "db.bin"), Key: key})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	// No ICE servers: loopback uses host candidates, keeping the test hermetic.
	tr, err := transport.New(transport.Config{})
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	rt, err := newRuntime(ctx, cancel, acc, store, sig, sig, tr, []byte(testRendezvous))
	if err != nil {
		t.Fatalf("newRuntime: %v", err)
	}
	return rt
}
