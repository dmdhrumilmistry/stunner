package signaling

import (
	"encoding/hex"
	"sync"
)

// Memory is an in-process Signaler used for tests and the headless harness. It
// models discovery, SDP/ICE exchange, bundle exchange, and presence without
// networking; the libp2p DHT (NewDHT) is the production path. Nodes sharing the
// same *Registry can discover and signal each other.
type Registry struct {
	mu     sync.Mutex
	advert map[string]string     // discoveryKey(hex) -> peerID
	online map[string]bool       // peerID -> online
	inbox  map[string]*peerInbox // peerID -> per-kind inboxes
}

// peerInbox holds one channel per frame kind so concurrent readers (e.g. the
// bundle-responder loop and an in-flight requester) never consume each other's
// frames. This mirrors the dedicated channels the DHT signaler uses.
type peerInbox struct {
	sdp  chan []byte
	cand chan []byte
	breq chan []byte
	brsp chan []byte
	done chan struct{} // closed by Close() to unblock pending recvs
	once sync.Once
}

func newPeerInbox() *peerInbox {
	return &peerInbox{
		sdp:  make(chan []byte, 64),
		cand: make(chan []byte, 64),
		breq: make(chan []byte, 64),
		brsp: make(chan []byte, 64),
		done: make(chan struct{}),
	}
}

// NewRegistry creates a shared in-process signaling fabric.
func NewRegistry() *Registry {
	return &Registry{
		advert: map[string]string{},
		online: map[string]bool{},
		inbox:  map[string]*peerInbox{},
	}
}

// Join returns a Signaler for peerID bound to this registry. The returned value
// also implements BundleExchanger.
func (r *Registry) Join(peerID string) Signaler {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.inbox[peerID]; !ok {
		r.inbox[peerID] = newPeerInbox()
	}
	r.online[peerID] = true
	return &memSignaler{reg: r, peerID: peerID}
}

type memSignaler struct {
	reg    *Registry
	peerID string
}

// memSignaler implements both Signaler and BundleExchanger.
var (
	_ Signaler        = (*memSignaler)(nil)
	_ BundleExchanger = (*memSignaler)(nil)
)

func (s *memSignaler) Advertise(discoveryKey []byte) error {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()
	s.reg.advert[hex.EncodeToString(discoveryKey)] = s.peerID
	return nil
}

func (s *memSignaler) Find(discoveryKey []byte) (PeerInfo, error) {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()
	peerID, ok := s.reg.advert[hex.EncodeToString(discoveryKey)]
	if !ok {
		return PeerInfo{}, ErrNotImplemented
	}
	return PeerInfo{PeerID: peerID, Addresses: []string{"inproc://" + peerID}}, nil
}

// chanFor returns the destination peer's channel for a kind, or nil if the peer
// is unknown.
func (s *memSignaler) chanFor(peerID, kind string) chan []byte {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()
	box, ok := s.reg.inbox[peerID]
	if !ok {
		return nil
	}
	switch kind {
	case "sdp":
		return box.sdp
	case "candidate":
		return box.cand
	case "bundle-req":
		return box.breq
	case "bundle-resp":
		return box.brsp
	default:
		return nil
	}
}

func (s *memSignaler) send(to, kind string, data []byte) error {
	ch := s.chanFor(to, kind)
	if ch == nil {
		return ErrNotImplemented
	}
	ch <- append([]byte(nil), data...)
	return nil
}

func (s *memSignaler) recv(kind string) ([]byte, error) {
	s.reg.mu.Lock()
	box := s.reg.inbox[s.peerID]
	s.reg.mu.Unlock()
	if box == nil {
		return nil, ErrNotImplemented
	}
	ch := s.chanFor(s.peerID, kind)
	if ch == nil {
		return nil, ErrNotImplemented
	}
	select {
	case b := <-ch:
		return b, nil
	case <-box.done:
		return nil, ErrClosed
	}
}

// ErrClosed is returned by recv when the signaler has been closed.
var ErrClosed = errorString("signaling: closed")

type errorString string

func (e errorString) Error() string { return string(e) }

func (s *memSignaler) SendSDP(peerID string, sdp []byte) error { return s.send(peerID, "sdp", sdp) }
func (s *memSignaler) RecvSDP(peerID string) ([]byte, error)   { return s.recv("sdp") }
func (s *memSignaler) SendCandidate(peerID string, c []byte) error {
	return s.send(peerID, "candidate", c)
}
func (s *memSignaler) RecvCandidate(peerID string) ([]byte, error) { return s.recv("candidate") }

func (s *memSignaler) LocalID() string { return s.peerID }

func (s *memSignaler) SendBundleRequest(peerID string, req []byte) error {
	return s.send(peerID, "bundle-req", req)
}
func (s *memSignaler) RecvBundleRequest() ([]byte, error) { return s.recv("bundle-req") }
func (s *memSignaler) SendBundleResponse(peerID string, bundle []byte) error {
	return s.send(peerID, "bundle-resp", bundle)
}
func (s *memSignaler) RecvBundleResponse() ([]byte, error) { return s.recv("bundle-resp") }

func (s *memSignaler) Presence(peerID string) (bool, error) {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()
	return s.reg.online[peerID], nil
}

func (s *memSignaler) Close() error {
	s.reg.mu.Lock()
	s.reg.online[s.peerID] = false
	box := s.reg.inbox[s.peerID]
	s.reg.mu.Unlock()
	if box != nil {
		box.once.Do(func() { close(box.done) })
	}
	return nil
}
