// Package runtime is the long-lived messaging engine the app drives over FFI.
//
// It owns a persistent account, a WebRTC transport and a libp2p-DHT signaler,
// and ties them together with pkg/node: it continuously accepts inbound links
// (listenLoop), connects to peers on demand to send (sendWorker), and reads
// incoming messages off every link (recvLoop). All of this runs on Go
// goroutines; the FFI surface stays non-blocking by enqueueing sends and
// draining an event queue (Poll), which the Dart side polls and turns into UI
// updates.
//
// Discovery and SDP exchange go through the pluggable signaling.Signaler, so the
// same engine runs hermetically in tests over an in-process Registry + pion
// loopback, and in production over the Kademlia DHT.
package runtime

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dmdhrumilmistry/stunner/core/pkg/account"
	"github.com/dmdhrumilmistry/stunner/core/pkg/contact"
	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/node"
	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
	"github.com/dmdhrumilmistry/stunner/core/pkg/signaling"
	"github.com/dmdhrumilmistry/stunner/core/pkg/storage"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
	"github.com/dmdhrumilmistry/stunner/core/pkg/vault"
)

// RendezvousSalt namespaces this app's discovery keys in the DHT.
var RendezvousSalt = []byte("stunner/rendezvous/1")

// connectAttempts bounds how many times a send retries discovery/connect before
// giving up (covers DHT propagation delay and a peer coming online).
const connectAttempts = 5

// Event is a runtime notification drained by Poll and surfaced in the UI.
type Event struct {
	Kind    string `json:"kind"` // message | file | presence | sent | sendFailed | receipt | error
	PeerFP  string `json:"peerFp,omitempty"`
	PeerURI string `json:"peerUri,omitempty"` // lets the UI save/reply to an inbound peer
	Text    string `json:"text,omitempty"`
	Name    string `json:"name,omitempty"` // file name (kind=file)
	Path    string `json:"path,omitempty"` // local path of a received file (kind=file)
	MsgID   string `json:"msgId,omitempty"`
	Online  bool   `json:"online,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Time    int64  `json:"time"`
}

// Runtime is a running messaging engine for one account.
type Runtime struct {
	node     *node.Node
	tr       transport.Transport
	sig      signaling.Signaler
	salt     []byte
	uri      string
	fp       string
	filesDir string // where received files are saved

	mu      sync.Mutex
	links   map[string]*node.Link
	lastIn  map[string]string // peerFP -> last inbound msgID (for read receipts)
	events  []Event
	closed  bool
	outbox  chan outReq
	closeCh chan struct{}
	wg      sync.WaitGroup
}

type outReq struct {
	peerURI  string
	text     string
	msgID    string
	filePath string // non-empty => send this file instead of text
}

// Start loads (or creates) the persistent account at dataDir and wires the
// production stack: WebRTC transport + libp2p Kademlia DHT (bootstrapped against
// public peers). handle is the display name embedded in the shareable contact
// URI. The data-at-rest key is stored at dataDir/key.bin (OS secure-store
// integration is a follow-up).
func Start(dataDir, handle string) (*Runtime, error) {
	key, err := loadOrCreateKey(dataDir)
	if err != nil {
		return nil, err
	}
	acc, err := account.LoadOrCreate(filepath.Join(dataDir, "account"), key)
	if err != nil {
		return nil, err
	}
	store, err := storage.Open(storage.Options{Path: filepath.Join(dataDir, "store.bin"), Key: key})
	if err != nil {
		return nil, err
	}
	tr, err := transport.New(transport.Config{ICEServers: settings.DefaultICEServers()})
	if err != nil {
		return nil, err
	}
	sig, err := signaling.NewDHT(context.Background(),
		"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic-v1")
	if err != nil {
		return nil, err
	}
	go sig.BootstrapPublic() // best-effort, async; discovery improves once connected

	rt := StartWith(node.New(acc, store), tr, sig, handle)
	rt.filesDir = filepath.Join(dataDir, "files")
	return rt, nil
}

// StartWith builds a runtime over an already-constructed node, transport and
// signaler. Production uses Start; tests inject an in-process Registry + pion
// loopback so the full engine runs without networking.
func StartWith(n *node.Node, tr transport.Transport, sig signaling.Signaler, handle string) *Runtime {
	r := &Runtime{
		node:     n,
		tr:       tr,
		sig:      sig,
		salt:     RendezvousSalt,
		uri:      n.Account.ContactURI(handle),
		fp:       n.Account.Fingerprint(),
		filesDir: filepath.Join(os.TempDir(), "stunner-recv", n.Account.Fingerprint()),
		links:    map[string]*node.Link{},
		lastIn:   map[string]string{},
		outbox:   make(chan outReq, 64),
		closeCh:  make(chan struct{}),
	}
	r.wg.Add(2)
	go r.listenLoop()
	go r.sendWorker()
	return r
}

// MyURI returns this account's shareable contact URI (share it so peers can add
// you). Fingerprint returns the identity fingerprint.
func (r *Runtime) MyURI() string       { return r.uri }
func (r *Runtime) Fingerprint() string { return r.fp }

// Send enqueues a text message to the peer identified by their contact URI. It
// returns immediately; delivery happens on a worker goroutine and surfaces a
// "sent" or "sendFailed" event (correlated by msgID) plus, on the peer, a
// "message" event.
func (r *Runtime) Send(peerURI, text, msgID string) {
	select {
	case r.outbox <- outReq{peerURI: peerURI, text: text, msgID: msgID}:
	case <-r.closeCh:
	}
}

// SendFile enqueues the file at path to the peer identified by their contact
// URI. Like Send it returns immediately and reports a "sent"/"sendFailed" event.
func (r *Runtime) SendFile(peerURI, path, msgID string) {
	select {
	case r.outbox <- outReq{peerURI: peerURI, filePath: path, msgID: msgID}:
	case <-r.closeCh:
	}
}

// Poll atomically drains and returns the pending events.
func (r *Runtime) Poll() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	ev := r.events
	r.events = nil
	return ev
}

// Stop tears down the engine: stops the loops, closes all links, and shuts down
// the transport and signaler. Idempotent.
func (r *Runtime) Stop() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	close(r.closeCh)
	links := r.links
	r.links = map[string]*node.Link{}
	r.mu.Unlock()

	for _, l := range links {
		_ = l.Close()
	}
	_ = r.sig.Close()
	_ = r.tr.Close()
	return nil
}

func (r *Runtime) listenLoop() {
	defer r.wg.Done()
	for {
		if r.isClosed() {
			return
		}
		link, err := r.node.Listen(r.tr, r.sig, r.salt)
		if err != nil {
			if r.isClosed() {
				return
			}
			r.push(Event{Kind: "error", Detail: "listen: " + err.Error()})
			select {
			case <-r.closeCh:
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		r.adopt(link)
	}
}

func (r *Runtime) sendWorker() {
	defer r.wg.Done()
	for {
		select {
		case <-r.closeCh:
			return
		case req := <-r.outbox:
			r.deliver(req)
		}
	}
}

func (r *Runtime) deliver(req outReq) {
	c, err := contact.ParseURI(req.peerURI)
	if err != nil {
		r.push(Event{Kind: "sendFailed", MsgID: req.msgID, Detail: "invalid contact URI"})
		return
	}
	fp := identity.Fingerprint(c.IdentityKey)

	link := r.getLink(fp)
	if link == nil {
		for attempt := 0; attempt < connectAttempts; attempt++ {
			if r.isClosed() {
				return
			}
			link, err = r.node.Connect(r.tr, r.sig, c.IdentityKey, r.salt)
			if err == nil {
				break
			}
			select {
			case <-r.closeCh:
				return
			case <-time.After(time.Duration(attempt+1) * 300 * time.Millisecond):
			}
		}
		if link == nil {
			r.push(Event{Kind: "sendFailed", PeerFP: fp, MsgID: req.msgID, Detail: err.Error()})
			return
		}
		r.adopt(link)
	}

	if req.filePath != "" {
		data, ferr := os.ReadFile(req.filePath)
		if ferr != nil {
			r.push(Event{Kind: "sendFailed", PeerFP: fp, MsgID: req.msgID, Detail: ferr.Error()})
			return
		}
		name := filepath.Base(req.filePath)
		if _, ferr := link.SendFile(r.node, fp, name, mimeOf(name), data); ferr != nil {
			r.dropLink(fp, link)
			r.push(Event{Kind: "sendFailed", PeerFP: fp, MsgID: req.msgID, Detail: ferr.Error()})
			return
		}
		r.push(Event{Kind: "sent", PeerFP: fp, MsgID: req.msgID})
		return
	}

	// Build the envelope with the caller's msgID so delivery receipts (which
	// reference it) correlate back to the UI's message.
	env, err := messaging.NewText(fp, req.text)
	if err != nil {
		r.push(Event{Kind: "sendFailed", PeerFP: fp, MsgID: req.msgID, Detail: err.Error()})
		return
	}
	if req.msgID != "" {
		env.MsgID = req.msgID
	}
	if err := link.SendEnvelope(r.node, env); err != nil {
		r.dropLink(fp, link)
		r.push(Event{Kind: "sendFailed", PeerFP: fp, MsgID: req.msgID, Detail: err.Error()})
		return
	}
	r.push(Event{Kind: "sent", PeerFP: fp, MsgID: req.msgID})
}

// adopt registers a link, announces presence, and starts reading from it.
func (r *Runtime) adopt(link *node.Link) {
	fp := link.PeerFingerprint()
	r.mu.Lock()
	if old, ok := r.links[fp]; ok && old != link {
		_ = old.Close()
	}
	r.links[fp] = link
	r.mu.Unlock()
	r.push(Event{Kind: "presence", PeerFP: fp, PeerURI: peerURI(link), Online: true})
	go r.recvLoop(link)
}

func (r *Runtime) recvLoop(link *node.Link) {
	fp := link.PeerFingerprint()
	uri := peerURI(link)
	for {
		env, err := link.Receive()
		if err != nil {
			r.dropLink(fp, link)
			r.push(Event{Kind: "presence", PeerFP: fp, PeerURI: uri, Online: false})
			return
		}
		switch env.Type {
		case messaging.TypeText:
			body, _ := env.Text()
			r.setLastInbound(fp, env.MsgID)
			r.push(Event{Kind: "message", PeerFP: fp, PeerURI: uri, MsgID: env.MsgID, Text: body.Text})
			// Acknowledge receipt so the sender sees a "delivered" tick.
			_ = link.SendReceipt(fp, env.MsgID, messaging.StateDelivered)
		case messaging.TypeFileOffer:
			offer, perr := node.ParseFileOffer(env)
			if perr != nil {
				r.push(Event{Kind: "error", Detail: "file offer: " + perr.Error()})
				continue
			}
			data, ferr := link.ReceiveFileBody(offer)
			if ferr != nil {
				r.push(Event{Kind: "error", Detail: "file recv: " + ferr.Error()})
				continue
			}
			path, serr := r.saveFile(offer.Name, data)
			if serr != nil {
				r.push(Event{Kind: "error", Detail: "file save: " + serr.Error()})
				continue
			}
			r.setLastInbound(fp, env.MsgID)
			r.push(Event{Kind: "file", PeerFP: fp, PeerURI: uri, MsgID: env.MsgID,
				Name: offer.Name, Path: path, Detail: offer.MIME})
			_ = link.SendReceipt(fp, env.MsgID, messaging.StateDelivered)
		case messaging.TypeReceipt:
			if rb, rerr := env.Receipt(); rerr == nil {
				r.push(Event{Kind: "receipt", PeerFP: fp, MsgID: rb.RefMsgID, Detail: string(rb.State)})
			}
		}
	}
}

// MarkRead sends a read receipt for the latest message received from the peer
// (call when the user opens the conversation), turning the peer's last sent
// message(s) "read".
func (r *Runtime) MarkRead(peerURI string) {
	c, err := contact.ParseURI(peerURI)
	if err != nil {
		return
	}
	fp := identity.Fingerprint(c.IdentityKey)
	link := r.getLink(fp)
	r.mu.Lock()
	msgID := r.lastIn[fp]
	r.mu.Unlock()
	if link == nil || msgID == "" {
		return
	}
	_ = link.SendReceipt(fp, msgID, messaging.StateRead)
}

// saveFile writes received file bytes under filesDir with a unique name and
// returns the path.
func (r *Runtime) saveFile(name string, data []byte) (string, error) {
	if err := os.MkdirAll(r.filesDir, 0o700); err != nil {
		return "", err
	}
	base := filepath.Base(name)
	if base == "." || base == "/" || base == "" {
		base = "file"
	}
	p := filepath.Join(r.filesDir, fmt.Sprintf("%d-%s", time.Now().UnixNano(), base))
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return "", err
	}
	return p, nil
}

// mimeOf guesses a content type from a filename extension.
func mimeOf(name string) string {
	if t := mime.TypeByExtension(filepath.Ext(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}

func (r *Runtime) setLastInbound(fp, msgID string) {
	r.mu.Lock()
	r.lastIn[fp] = msgID
	r.mu.Unlock()
}

// peerURI reconstructs the peer's shareable contact URI from the identity key
// learned during the handshake, so the UI can save/reply to an inbound peer.
func peerURI(link *node.Link) string {
	key := link.PeerIdentityKey()
	if len(key) == 0 {
		return ""
	}
	return contact.URI("", key)
}

func (r *Runtime) getLink(fp string) *node.Link {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.links[fp]
}

func (r *Runtime) dropLink(fp string, link *node.Link) {
	r.mu.Lock()
	if r.links[fp] == link {
		delete(r.links, fp)
	}
	r.mu.Unlock()
	_ = link.Close()
}

func (r *Runtime) push(e Event) {
	e.Time = time.Now().Unix()
	r.mu.Lock()
	r.events = append(r.events, e)
	r.mu.Unlock()
}

func (r *Runtime) isClosed() bool {
	select {
	case <-r.closeCh:
		return true
	default:
		return false
	}
}

func loadOrCreateKey(dir string) ([]byte, error) {
	if dir == "" {
		return nil, errors.New("runtime: empty data dir")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "key.bin")
	if b, err := os.ReadFile(path); err == nil && len(b) == vault.KeySize {
		return b, nil
	}
	key := make([]byte, vault.KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}
