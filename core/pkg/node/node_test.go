package node

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/account"
	"github.com/dmdhrumilmistry/stunner/core/pkg/mailbox"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/storage"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

func newNode(t *testing.T) *Node {
	t.Helper()
	key := bytes.Repeat([]byte{5}, 32)
	dir := t.TempDir()
	acc, err := account.LoadOrCreate(dir, key)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	store, err := storage.Open(storage.Options{Path: dir + "/db.bin", Key: key})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	return New(acc, store)
}

func TestOnlineTextExchange(t *testing.T) {
	alice, bob := newNode(t), newNode(t)
	ca, cb := transport.Pipe()

	linkA, err := alice.Dial(ca, bob.Bundle())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	linkB, err := bob.Accept(cb)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}

	if linkA.PeerFingerprint() != bob.Account.Fingerprint() {
		t.Error("link peer fingerprint mismatch")
	}

	if _, err := linkA.SendText(alice, "conv1", "hello bob 🎉"); err != nil {
		t.Fatalf("send: %v", err)
	}
	env, err := linkB.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	body, _ := env.Text()
	if body.Text != "hello bob 🎉" {
		t.Errorf("got %q", body.Text)
	}

	// Persistence: Alice's store recorded the outgoing message.
	msgs, _ := alice.Store.Messages("conv1", 0, 0)
	if len(msgs) != 1 {
		t.Errorf("expected 1 stored message, got %d", len(msgs))
	}
}

func TestFileTransfer(t *testing.T) {
	alice, bob := newNode(t), newNode(t)
	ca, cb := transport.Pipe()
	linkA, _ := alice.Dial(ca, bob.Bundle())
	linkB, _ := bob.Accept(cb)

	data := make([]byte, 50*1024)
	rand.Read(data)

	if _, err := linkA.SendFile(alice, "conv1", "f.bin", "application/octet-stream", data); err != nil {
		t.Fatalf("send file: %v", err)
	}
	offer, got, err := linkB.ReceiveFile()
	if err != nil {
		t.Fatalf("receive file: %v", err)
	}
	if offer.Name != "f.bin" {
		t.Errorf("offer name = %q", offer.Name)
	}
	if !bytes.Equal(got, data) {
		t.Error("received file does not match sent file")
	}
}

func TestOfflineDeliveryViaMailbox(t *testing.T) {
	alice, bob := newNode(t), newNode(t)
	mb := mailbox.NewMemory()

	env, _ := messaging.NewText("conv1", "you were offline 👋")
	if err := alice.SendOffline(mb, bob.Bundle(), env); err != nil {
		t.Fatalf("send offline: %v", err)
	}
	if mb.Pending(bob.Account.Fingerprint()) != 1 {
		t.Fatalf("expected 1 queued message")
	}

	got, err := bob.FetchOffline(mb)
	if err != nil {
		t.Fatalf("fetch offline: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 fetched message, got %d", len(got))
	}
	body, _ := got[0].Text()
	if body.Text != "you were offline 👋" {
		t.Errorf("got %q", body.Text)
	}
}
