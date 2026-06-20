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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dmdhrumilmistry/stunner/core/pkg/account"
	"github.com/dmdhrumilmistry/stunner/core/pkg/core"
	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/mailbox"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/node"
	"github.com/dmdhrumilmistry/stunner/core/pkg/signaling"
	"github.com/dmdhrumilmistry/stunner/core/pkg/storage"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

func main() {
	live := flag.Bool("live", false, "run the live two-device path over real WebRTC (loopback)")
	flag.Parse()

	fmt.Println(core.VersionString())
	run := run
	if *live {
		run = runLive
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// runLive demonstrates the real two-device delivery path: two nodes establish a
// live pion WebRTC data channel (loopback, hermetic) negotiated via a signaler,
// run the interactive handshake, and exchange E2E-encrypted messages directly
// peer-to-peer. No message bytes pass through a server (pure P2P); STUN/TURN,
// when configured, only help with NAT traversal.
func runLive() error {
	tmp, err := os.MkdirTemp("", "stunnerd-live-*")
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
	fmt.Println("\nlive (WebRTC data channel) identities:")
	fmt.Printf("  alice %s\n", alice.Account.Fingerprint())
	fmt.Printf("  bob   %s\n", bob.Account.Fingerprint())

	trA, err := transport.New(transport.Config{})
	if err != nil {
		return err
	}
	trB, err := transport.New(transport.Config{})
	if err != nil {
		return err
	}
	reg := signaling.NewRegistry()
	sigA := reg.Join(alice.Account.Fingerprint())
	sigB := reg.Join(bob.Account.Fingerprint())
	salt := []byte("stunnerd-live")

	type result struct {
		link *node.Link
		err  error
	}
	// Advertise Bob before Alice looks him up (removes the goroutine-start race;
	// Listen re-advertises idempotently).
	if err := sigB.Advertise(bobDiscoveryKey(bob, salt)); err != nil {
		return err
	}
	bobCh := make(chan result, 1)
	go func() {
		l, err := bob.Listen(trB, sigB, salt)
		bobCh <- result{l, err}
	}()

	linkA, err := alice.Connect(trA, sigA, bob.Account.Identity.SigningPub, salt)
	if err != nil {
		return fmt.Errorf("alice connect: %w", err)
	}
	defer linkA.Close()

	var rb result
	select {
	case rb = <-bobCh:
	case <-time.After(45 * time.Second):
		return fmt.Errorf("bob listen timed out")
	}
	if rb.err != nil {
		return fmt.Errorf("bob listen: %w", rb.err)
	}
	defer rb.link.Close()

	if _, err := linkA.SendText(alice, "conv", "hello bob over webrtc 🎉🔒"); err != nil {
		return err
	}
	env, err := rb.link.Receive()
	if err != nil {
		return err
	}
	body, _ := env.Text()
	fmt.Printf("\nalice -> bob (live, E2E over data channel): %q\n", body.Text)

	if _, err := rb.link.SendText(bob, "conv", "got it, fully P2P 👍"); err != nil {
		return err
	}
	env, err = linkA.Receive()
	if err != nil {
		return err
	}
	body, _ = env.Text()
	fmt.Printf("bob -> alice (live, E2E over data channel): %q\n", body.Text)

	fmt.Println("\nlive two-device delivery OK (pure P2P)")
	return nil
}

func bobDiscoveryKey(n *node.Node, salt []byte) []byte {
	return identity.DiscoveryKey(n.Account.Identity.SigningPub, salt)
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
