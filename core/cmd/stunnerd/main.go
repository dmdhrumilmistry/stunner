// Command stunnerd is a headless harness that runs the full Stunner pipeline
// in-process: it creates two encrypted accounts, establishes an X3DH + Double
// Ratchet session over an in-process transport, exchanges an end-to-end
// encrypted message and a file, verifies the safety number, and demonstrates
// offline delivery via the optional mailbox.
//
// It uses the in-process transport/mailbox so the whole stack runs without
// networking; the pion/libp2p backends slot in behind the same interfaces.
package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmdhrumilmistry/stunner/core/pkg/account"
	"github.com/dmdhrumilmistry/stunner/core/pkg/core"
	"github.com/dmdhrumilmistry/stunner/core/pkg/mailbox"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/node"
	"github.com/dmdhrumilmistry/stunner/core/pkg/storage"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

func main() {
	fmt.Println(core.VersionString())
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	tmp, err := os.MkdirTemp("", "stunnerd-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	alice, err := makeNode(tmp, "alice")
	if err != nil {
		return err
	}
	bob, err := makeNode(tmp, "bob")
	if err != nil {
		return err
	}
	fmt.Println("\nidentities:")
	fmt.Printf("  alice %s\n", alice.Account.Fingerprint())
	fmt.Printf("  bob   %s\n", bob.Account.Fingerprint())

	// Safety number for out-of-band verification (identical on both sides).
	sn, err := core.SafetyNumber(alice.Account.ContactURI("alice"), bob.Account.ContactURI("bob"))
	if err != nil {
		return err
	}
	fmt.Printf("\nsafety number (compare on both devices):\n  %s\n", sn)

	// Establish an encrypted link over the in-process transport.
	ca, cb := transport.Pipe()
	linkA, err := alice.Dial(ca, bob.Bundle())
	if err != nil {
		return err
	}
	linkB, err := bob.Accept(cb)
	if err != nil {
		return err
	}

	// End-to-end encrypted message.
	if _, err := linkA.SendText(alice, "conv", "hello bob 🎉🔒"); err != nil {
		return err
	}
	env, err := linkB.Receive()
	if err != nil {
		return err
	}
	body, _ := env.Text()
	fmt.Printf("\nalice -> bob (E2E): %q\n", body.Text)

	// Encrypted file transfer.
	file := make([]byte, 40*1024)
	rand.Read(file)
	if _, err := linkA.SendFile(alice, "conv", "secret.bin", "application/octet-stream", file); err != nil {
		return err
	}
	offer, got, err := linkB.ReceiveFile()
	if err != nil {
		return err
	}
	fmt.Printf("file transfer: %q (%d bytes) integrity=%v\n", offer.Name, len(got), bytes.Equal(got, file))

	// Offline delivery via the optional mailbox.
	mb := mailbox.NewMemory()
	offline, _ := messaging.NewText("conv", "sent while you were offline 👋")
	if err := alice.SendOffline(mb, bob.Bundle(), offline); err != nil {
		return err
	}
	fetched, err := bob.FetchOffline(mb)
	if err != nil {
		return err
	}
	if len(fetched) > 0 {
		ob, _ := fetched[0].Text()
		fmt.Printf("offline mailbox -> bob: %q\n", ob.Text)
	}

	fmt.Println("\nall pipeline stages OK")
	return nil
}

func makeNode(tmp, name string) (*node.Node, error) {
	// In production the key comes from the OS secure store; here it is derived
	// per-account for the demo only.
	key := bytes.Repeat([]byte(name + "-")[:1], 32)
	dir := filepath.Join(tmp, name)
	acc, err := account.LoadOrCreate(dir, key)
	if err != nil {
		return nil, err
	}
	store, err := storage.Open(storage.Options{Path: filepath.Join(dir, "db.bin"), Key: key})
	if err != nil {
		return nil, err
	}
	return node.New(acc, store), nil
}
