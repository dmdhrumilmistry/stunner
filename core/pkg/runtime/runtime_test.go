package runtime

import (
	"bytes"
	"testing"
	"time"

	"github.com/dmdhrumilmistry/stunner/core/pkg/account"
	"github.com/dmdhrumilmistry/stunner/core/pkg/node"
	"github.com/dmdhrumilmistry/stunner/core/pkg/signaling"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

func newRuntime(t *testing.T, reg *signaling.Registry, handle string) *Runtime {
	t.Helper()
	key := bytes.Repeat([]byte{7}, 32)
	acc, err := account.LoadOrCreate(t.TempDir(), key)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	tr, err := transport.New(transport.Config{}) // no ICE servers: loopback host candidates
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	sig := reg.Join(acc.Fingerprint())
	return StartWith(node.New(acc, nil), tr, sig, handle)
}

// waitEvent polls until an event of the given kind (and optional text) arrives,
// returning it. Events not matched are accumulated so earlier ones aren't lost.
func waitEvent(t *testing.T, r *Runtime, kind, text string) Event {
	t.Helper()
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range r.Poll() {
			if e.Kind == kind && (text == "" || e.Text == text) {
				return e
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q event (text=%q)", kind, text)
	return Event{}
}

// TestRuntimeBidirectional runs two full runtimes over real pion WebRTC
// (loopback) and an in-process signaler, and verifies messages flow both ways
// with the expected events — exercising listen/connect/recv, the async send
// worker, link reuse, and the event queue end to end (no networking).
func TestRuntimeBidirectional(t *testing.T) {
	reg := signaling.NewRegistry()
	alice := newRuntime(t, reg, "alice")
	bob := newRuntime(t, reg, "bob")
	defer alice.Stop()
	defer bob.Stop()

	// Alice -> Bob (Alice connects, Bob accepts).
	alice.Send(bob.MyURI(), "hello bob 🔒", "m1")
	if got := waitEvent(t, bob, "message", "hello bob 🔒"); got.PeerFP != alice.Fingerprint() {
		t.Errorf("bob saw peer %q, want %q", got.PeerFP, alice.Fingerprint())
	}
	if sent := waitEvent(t, alice, "sent", ""); sent.MsgID != "m1" {
		t.Errorf("alice sent msgId = %q, want m1", sent.MsgID)
	}

	// Bob -> Alice over the established link (reverse direction).
	bob.Send(alice.MyURI(), "hi alice 👋", "m2")
	if got := waitEvent(t, alice, "message", "hi alice 👋"); got.PeerFP != bob.Fingerprint() {
		t.Errorf("alice saw peer %q, want %q", got.PeerFP, bob.Fingerprint())
	}
}

// TestRuntimeSendInvalidURI surfaces a sendFailed event for a malformed URI.
func TestRuntimeSendInvalidURI(t *testing.T) {
	reg := signaling.NewRegistry()
	r := newRuntime(t, reg, "solo")
	defer r.Stop()

	r.Send("not-a-stunner-uri", "x", "m9")
	if e := waitEvent(t, r, "sendFailed", ""); e.MsgID != "m9" {
		t.Errorf("sendFailed msgId = %q, want m9", e.MsgID)
	}
}
