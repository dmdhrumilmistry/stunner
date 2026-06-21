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

// collector buffers drained events so matching one never discards the others
// that arrived in the same Poll batch.
type collector struct {
	r   *Runtime
	buf []Event
}

func (c *collector) wait(t *testing.T, desc string, pred func(Event) bool) Event {
	t.Helper()
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		c.buf = append(c.buf, c.r.Poll()...)
		for i, e := range c.buf {
			if pred(e) {
				c.buf = append(c.buf[:i], c.buf[i+1:]...)
				return e
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", desc)
	return Event{}
}

// TestRuntimeBidirectional runs two full runtimes over real pion WebRTC
// (loopback) and an in-process signaler, verifying messages flow both ways with
// the expected events plus delivered/read receipts — exercising listen/connect/
// recv, the async send worker, link reuse, the receipt path, and the event queue
// end to end (no networking).
func TestRuntimeBidirectional(t *testing.T) {
	reg := signaling.NewRegistry()
	alice := newRuntime(t, reg, "alice")
	bob := newRuntime(t, reg, "bob")
	defer alice.Stop()
	defer bob.Stop()
	ac := &collector{r: alice}
	bc := &collector{r: bob}

	// Alice -> Bob (Alice connects, Bob accepts).
	alice.Send(bob.MyURI(), "hello bob 🔒", "m1")
	got := bc.wait(t, "bob message", func(e Event) bool { return e.Kind == "message" && e.Text == "hello bob 🔒" })
	if got.PeerFP != alice.Fingerprint() {
		t.Errorf("bob saw peer %q, want %q", got.PeerFP, alice.Fingerprint())
	}
	if e := ac.wait(t, "alice sent", func(e Event) bool { return e.Kind == "sent" }); e.MsgID != "m1" {
		t.Errorf("alice sent msgId = %q, want m1", e.MsgID)
	}

	// Bob auto-acks delivery; Alice sees a "delivered" receipt for m1.
	if e := ac.wait(t, "delivered receipt",
		func(e Event) bool { return e.Kind == "receipt" && e.Detail == "DELIVERED" }); e.MsgID != "m1" {
		t.Errorf("delivered receipt msgId = %q, want m1", e.MsgID)
	}

	// Bob opens the conversation -> read receipt back to Alice.
	bob.MarkRead(alice.MyURI())
	if e := ac.wait(t, "read receipt",
		func(e Event) bool { return e.Kind == "receipt" && e.Detail == "READ" }); e.MsgID != "m1" {
		t.Errorf("read receipt msgId = %q, want m1", e.MsgID)
	}

	// Bob -> Alice over the established link (reverse direction).
	bob.Send(alice.MyURI(), "hi alice 👋", "m2")
	got = ac.wait(t, "alice message", func(e Event) bool { return e.Kind == "message" && e.Text == "hi alice 👋" })
	if got.PeerFP != bob.Fingerprint() {
		t.Errorf("alice saw peer %q, want %q", got.PeerFP, bob.Fingerprint())
	}
}

// TestRuntimeSendInvalidURI surfaces a sendFailed event for a malformed URI.
func TestRuntimeSendInvalidURI(t *testing.T) {
	reg := signaling.NewRegistry()
	r := newRuntime(t, reg, "solo")
	defer r.Stop()
	c := &collector{r: r}

	r.Send("not-a-stunner-uri", "x", "m9")
	if e := c.wait(t, "sendFailed", func(e Event) bool { return e.Kind == "sendFailed" }); e.MsgID != "m9" {
		t.Errorf("sendFailed msgId = %q, want m9", e.MsgID)
	}
}
