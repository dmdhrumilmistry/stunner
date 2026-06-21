// Package session is the long-lived Stunner runtime that the app drives over
// FFI: it assembles an account's identity and sessions onto the production
// transport (pion/WebRTC) and signaling (libp2p DHT) and exposes the operations
// the UI needs — connect to a peer, send text, and receive inbound messages
// asynchronously.
//
// node.Node/Link (pkg/node) carry the per-connection crypto; this package owns
// the *runtime*: discovery, the prekey-bundle exchange the X3DH initiator needs,
// the accept/receive goroutines, the live-link registry, and lifecycle. It is
// the first place the production transport + signaling backends are wired
// together (the cmd/stunnerd harness only ever used the in-process Pipe).
//
// Start builds the production backends; newRuntime takes the same pieces as
// interfaces so tests can drive the identical orchestration over an in-process
// transport and the in-memory signaler.
package session

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"

	"github.com/dmdhrumilmistry/stunner/core/pkg/account"
	"github.com/dmdhrumilmistry/stunner/core/pkg/contact"
	"github.com/dmdhrumilmistry/stunner/core/pkg/crypto"
	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/node"
	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
	"github.com/dmdhrumilmistry/stunner/core/pkg/signaling"
	"github.com/dmdhrumilmistry/stunner/core/pkg/storage"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

// DefaultRendezvous is the app/build-wide salt for the DHT discovery key. All
// Stunner peers must share it to find each other; bump it only to intentionally
// partition the discovery namespace.
var DefaultRendezvous = []byte("stunner/rendezvous/v1")

// Config is everything the app supplies to start a runtime.
type Config struct {
	// AccountDir is where the encrypted identity and store live.
	AccountDir string
	// Key is the 32-byte vault key from the OS secure store.
	Key []byte
	// Settings carries the ICE (STUN/TURN) server list.
	Settings settings.Settings
	// Rendezvous salts the DHT discovery key (see identity.DiscoveryKey). It is
	// an app/build constant shared by all Stunner peers.
	Rendezvous []byte
}

// Event is an asynchronous notification surfaced to the app. It is flattened and
// JSON-tagged so the FFI layer can hand it across the boundary verbatim.
type Event struct {
	Kind   string `json:"kind"` // "message" | "connected" | "disconnected" | "error"
	ConvID string `json:"convId,omitempty"`
	PeerFP string `json:"peerFp,omitempty"`
	Text   string `json:"text,omitempty"`
	MsgID  string `json:"msgId,omitempty"`
	Err    string `json:"err,omitempty"`
}

// bundleRequest is sent by a dialer to ask a peer for its prekey bundle. It
// carries the requester's signaling peer ID so the responder knows where to
// reply (works uniformly across the DHT and in-memory signalers).
type bundleRequest struct {
	FromPeerID string `json:"fromPeerId"`
}

// Runtime is a running Stunner instance for one account.
//
// Concurrency note: the DHTSignaler exchanges SDP over a single shared channel
// (one negotiation at a time per node), so signalMu serializes outbound Connect
// negotiations. Simultaneous bidirectional dialing between the same two peers is
// not yet supported; established links run fully concurrently.
type Runtime struct {
	acc        *account.Account
	store      storage.Store
	node       *node.Node
	sig        signaling.Signaler
	bx         signaling.BundleExchanger
	tr         transport.Transport
	rendezvous []byte

	ctx    context.Context
	cancel context.CancelFunc

	signalMu sync.Mutex // serializes the single-negotiation-at-a-time signaler

	mu     sync.Mutex
	links  map[string]*node.Link // peerFP -> live link
	closed bool                  // set under mu once Stop has begun teardown

	events    chan Event
	wg        sync.WaitGroup
	stopOnce  sync.Once
	myBundleJ []byte
}

// Start opens the account, builds the production transport (pion/WebRTC) and
// signaler (libp2p DHT), advertises this node for discovery, and launches the
// accept and bundle-responder loops.
func Start(cfg Config) (*Runtime, error) {
	acc, err := account.LoadOrCreate(cfg.AccountDir, cfg.Key)
	if err != nil {
		return nil, err
	}
	store, err := storage.Open(storage.Options{
		Path: filepath.Join(cfg.AccountDir, "db.bin"),
		Key:  cfg.Key,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	// NewDHT returns the concrete *DHTSignaler, which implements both Signaler
	// and BundleExchanger.
	sig, err := signaling.NewDHT(ctx)
	if err != nil {
		cancel()
		_ = store.Close()
		return nil, err
	}
	tr, err := transport.New(transport.Config{ICEServers: cfg.Settings.EffectiveICEServers()})
	if err != nil {
		_ = sig.Close()
		cancel()
		_ = store.Close()
		return nil, err
	}

	return newRuntime(ctx, cancel, acc, store, sig, sig, tr, cfg.Rendezvous)
}

// newRuntime wires a runtime from already-constructed parts and starts its
// goroutines. Start uses it with production backends; tests use it with the
// in-memory signaler and an in-process transport.
func newRuntime(
	ctx context.Context, cancel context.CancelFunc,
	acc *account.Account, store storage.Store,
	sig signaling.Signaler, bx signaling.BundleExchanger, tr transport.Transport,
	rendezvous []byte,
) (*Runtime, error) {
	bundleJSON, err := json.Marshal(acc.PreKeys.Bundle())
	if err != nil {
		cancel()
		return nil, err
	}
	r := &Runtime{
		acc:        acc,
		store:      store,
		node:       node.New(acc, store),
		sig:        sig,
		bx:         bx,
		tr:         tr,
		rendezvous: rendezvous,
		ctx:        ctx,
		cancel:     cancel,
		links:      map[string]*node.Link{},
		events:     make(chan Event, 64),
		myBundleJ:  bundleJSON,
	}
	if err := sig.Advertise(r.discoveryKey(acc.Identity.SigningPub)); err != nil {
		cancel()
		return nil, err
	}
	r.wg.Add(2)
	go r.acceptLoop()
	go r.bundleResponder()
	return r, nil
}

// ContactURI returns this account's shareable "stunner:contact" URI (render as a
// QR code) for the persistent identity peers actually reach.
func (r *Runtime) ContactURI(handle string) string { return r.acc.ContactURI(handle) }

// Fingerprint returns this account's identity fingerprint.
func (r *Runtime) Fingerprint() string { return r.acc.Fingerprint() }

// Events returns the channel of asynchronous notifications. The FFI layer drains
// it (desktop polls; mobile pumps it into the EventHandler callback).
func (r *Runtime) Events() <-chan Event { return r.events }

// DrainEvents removes and returns all currently buffered events without
// blocking. The desktop FFI poll function calls this each tick; it always
// returns a non-nil (possibly empty) slice.
func (r *Runtime) DrainEvents() []Event {
	out := []Event{}
	for {
		select {
		case e := <-r.events:
			out = append(out, e)
		default:
			return out
		}
	}
}

// Connect discovers, dials, and establishes an encrypted link to the peer
// identified by a scanned "stunner:contact" URI, returning the peer fingerprint.
func (r *Runtime) Connect(contactURI string) (string, error) {
	c, err := contact.ParseURI(contactURI)
	if err != nil {
		return "", err
	}
	info, err := r.sig.Find(r.discoveryKey(c.IdentityKey))
	if err != nil {
		return "", err
	}

	r.signalMu.Lock()
	defer r.signalMu.Unlock()

	// The X3DH initiator needs the peer's prekey bundle, which the contact URI
	// does not carry — request it over the signaler first.
	bundle, err := r.requestBundle(info.PeerID)
	if err != nil {
		return "", err
	}
	// Bind the bundle's identity to the scanned identity (defeats a substituted
	// bundle) and apply trust-on-first-use for the handle.
	if !ed25519.PublicKey(bundle.IdentitySign).Equal(c.IdentityKey) {
		return "", errors.New("session: bundle identity does not match contact URI")
	}
	if _, err := r.acc.Contacts.SeenKey(c.Handle, bundle.IdentitySign); err != nil {
		return "", err
	}

	conn, err := r.tr.Dial(info.PeerID, r.sig)
	if err != nil {
		return "", err
	}
	link, err := r.node.Dial(conn, bundle)
	if err != nil {
		_ = conn.Close()
		return "", err
	}
	r.registerLink(link)
	r.emit(Event{Kind: "connected", PeerFP: link.PeerFingerprint()})
	return link.PeerFingerprint(), nil
}

// SendText sends a text message over the live link to peerFP.
func (r *Runtime) SendText(convID, peerFP, text string) (string, error) {
	r.mu.Lock()
	link, ok := r.links[peerFP]
	r.mu.Unlock()
	if !ok {
		return "", errors.New("session: no live link for peer " + peerFP)
	}
	env, err := link.SendText(r.node, convID, text)
	if err != nil {
		return "", err
	}
	return env.MsgID, nil
}

// Stop cancels the runtime, closes all links, the signaler, transport, and the
// store, and waits for every goroutine to exit.
func (r *Runtime) Stop() error {
	r.stopOnce.Do(func() {
		r.cancel()        // unblocks DHT signaler recvs
		_ = r.sig.Close() // unblocks in-memory signaler recvs
		_ = r.tr.Close()  // unblocks the accept loop

		r.mu.Lock()
		r.closed = true // reject any link registered after this point
		for _, l := range r.links {
			_ = l.Close() // unblocks each receive loop
		}
		r.links = map[string]*node.Link{}
		r.mu.Unlock()

		r.wg.Wait()
		_ = r.store.Close()
	})
	return nil
}

// requestBundle asks peerID for its prekey bundle and returns the decoded value.
func (r *Runtime) requestBundle(peerID string) (crypto.PreKeyBundle, error) {
	req, err := json.Marshal(bundleRequest{FromPeerID: r.bx.LocalID()})
	if err != nil {
		return crypto.PreKeyBundle{}, err
	}
	if err := r.bx.SendBundleRequest(peerID, req); err != nil {
		return crypto.PreKeyBundle{}, err
	}
	raw, err := r.bx.RecvBundleResponse()
	if err != nil {
		return crypto.PreKeyBundle{}, err
	}
	var bundle crypto.PreKeyBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return crypto.PreKeyBundle{}, err
	}
	return bundle, nil
}

// acceptLoop accepts inbound connections and establishes responder links.
func (r *Runtime) acceptLoop() {
	defer r.wg.Done()
	for {
		conn, err := r.tr.Accept(r.sig)
		if err != nil {
			if r.ctx.Err() != nil {
				return
			}
			r.emit(Event{Kind: "error", Err: "accept: " + err.Error()})
			return
		}
		link, err := r.node.Accept(conn)
		if err != nil {
			_ = conn.Close()
			r.emit(Event{Kind: "error", Err: "accept handshake: " + err.Error()})
			continue
		}
		r.registerLink(link)
		r.emit(Event{Kind: "connected", PeerFP: link.PeerFingerprint()})
	}
}

// bundleResponder answers inbound bundle requests with this node's prekey
// bundle. node.Accept reconstructs the X3DH secret from its own prekeys, so only
// the dialer ever needs the responder's bundle.
func (r *Runtime) bundleResponder() {
	defer r.wg.Done()
	for {
		raw, err := r.bx.RecvBundleRequest()
		if err != nil {
			return // ctx cancelled / signaler closed
		}
		var req bundleRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			continue
		}
		_ = r.bx.SendBundleResponse(req.FromPeerID, r.myBundleJ)
	}
}

// receiveLoop decrypts inbound envelopes on a link and emits events until the
// link closes.
func (r *Runtime) receiveLoop(link *node.Link) {
	defer r.wg.Done()
	peerFP := link.PeerFingerprint()
	for {
		env, err := link.Receive()
		if err != nil {
			r.deregisterLink(peerFP, link)
			if r.ctx.Err() == nil {
				r.emit(Event{Kind: "disconnected", PeerFP: peerFP})
			}
			return
		}
		if env.Type != messaging.TypeText {
			continue // first pass surfaces text only
		}
		body, err := env.Text()
		if err != nil {
			r.emit(Event{Kind: "error", PeerFP: peerFP, Err: err.Error()})
			continue
		}
		r.emit(Event{
			Kind:   "message",
			ConvID: env.ConvID,
			PeerFP: peerFP,
			Text:   body.Text,
			MsgID:  env.MsgID,
		})
	}
}

func (r *Runtime) registerLink(link *node.Link) {
	fp := link.PeerFingerprint()
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		_ = link.Close()
		return
	}
	if old, ok := r.links[fp]; ok {
		_ = old.Close()
	}
	r.links[fp] = link
	r.wg.Add(1) // under mu, so it is ordered before Stop's wg.Wait
	r.mu.Unlock()
	go r.receiveLoop(link)
}

// deregisterLink removes a link only if it is still the registered one, so a
// closing link never evicts a newer replacement for the same peer.
func (r *Runtime) deregisterLink(fp string, link *node.Link) {
	r.mu.Lock()
	if r.links[fp] == link {
		delete(r.links, fp)
	}
	r.mu.Unlock()
}

func (r *Runtime) discoveryKey(pub ed25519.PublicKey) []byte {
	return identity.DiscoveryKey(pub, r.rendezvous)
}

// emit delivers an event, dropping it if the runtime is shutting down.
func (r *Runtime) emit(e Event) {
	select {
	case r.events <- e:
	case <-r.ctx.Done():
	}
}
