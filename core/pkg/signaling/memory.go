package signaling

import (
	"encoding/hex"
	"sync"
)

// Registry is an in-process Signaler fabric for tests and the headless harness.
// It models discovery, SDP/ICE exchange, and presence without networking; the
// libp2p DHT (NewDHT) is the production path. Nodes sharing the same *Registry
// can discover and signal each other.
//
// SDP routing mirrors the transport's non-trickle convention: SendSDP with a
// non-empty peerID is an OFFER to that peer; SendSDP("") is an ANSWER to the
// peer we last received an offer from. RecvSDP("") returns the next inbound
// offer (the answerer side); RecvSDP(peerID) returns the answer from peerID (the
// offerer side). Offers and answers travel on separate queues so a node that
// both listens and connects never consumes the wrong one.
type Registry struct {
	mu     sync.Mutex
	advert map[string]string // discoveryKey(hex) -> peerID
	online map[string]bool   // peerID -> online
	boxes  map[string]*inbox // peerID -> inbox
}

type inbox struct {
	offers  chan offerFrame
	cands   chan []byte
	mu      sync.Mutex
	answers map[string]chan []byte // fromPeerID -> answer
}

type offerFrame struct {
	from string
	data []byte
}

func newInbox() *inbox {
	return &inbox{
		offers:  make(chan offerFrame, 16),
		cands:   make(chan []byte, 16),
		answers: map[string]chan []byte{},
	}
}

func (b *inbox) answerChan(from string) chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch, ok := b.answers[from]
	if !ok {
		ch = make(chan []byte, 8)
		b.answers[from] = ch
	}
	return ch
}

// NewRegistry creates a shared in-process signaling fabric.
func NewRegistry() *Registry {
	return &Registry{
		advert: map[string]string{},
		online: map[string]bool{},
		boxes:  map[string]*inbox{},
	}
}

func (r *Registry) box(peerID string) *inbox {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.boxes[peerID]
	if !ok {
		b = newInbox()
		r.boxes[peerID] = b
	}
	return b
}

// Join returns a Signaler for peerID bound to this registry.
func (r *Registry) Join(peerID string) Signaler {
	r.mu.Lock()
	if _, ok := r.boxes[peerID]; !ok {
		r.boxes[peerID] = newInbox()
	}
	r.online[peerID] = true
	r.mu.Unlock()
	return &memSignaler{reg: r, peerID: peerID}
}

type memSignaler struct {
	reg        *Registry
	peerID     string
	mu         sync.Mutex
	lastRemote string // peer we last received an offer from; reply target for ""
}

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

// SendSDP routes an offer (peerID != "") or an answer (peerID == "").
func (s *memSignaler) SendSDP(peerID string, sdp []byte) error {
	data := append([]byte(nil), sdp...)
	if peerID != "" {
		s.reg.box(peerID).offers <- offerFrame{from: s.peerID, data: data}
		return nil
	}
	s.mu.Lock()
	to := s.lastRemote
	s.mu.Unlock()
	if to == "" {
		return ErrNotImplemented
	}
	s.reg.box(to).answerChan(s.peerID) <- data
	return nil
}

// RecvSDP returns the next inbound offer (peerID == "") or the answer from a
// specific peer (peerID != "").
func (s *memSignaler) RecvSDP(peerID string) ([]byte, error) {
	if peerID == "" {
		f := <-s.reg.box(s.peerID).offers
		s.mu.Lock()
		s.lastRemote = f.from
		s.mu.Unlock()
		return f.data, nil
	}
	return <-s.reg.box(s.peerID).answerChan(peerID), nil
}

func (s *memSignaler) SendCandidate(peerID string, c []byte) error {
	data := append([]byte(nil), c...)
	to := peerID
	if to == "" {
		s.mu.Lock()
		to = s.lastRemote
		s.mu.Unlock()
		if to == "" {
			return ErrNotImplemented
		}
	}
	s.reg.box(to).cands <- data
	return nil
}

func (s *memSignaler) RecvCandidate(string) ([]byte, error) {
	return <-s.reg.box(s.peerID).cands, nil
}

func (s *memSignaler) Presence(peerID string) (bool, error) {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()
	return s.reg.online[peerID], nil
}

func (s *memSignaler) Close() error {
	s.reg.mu.Lock()
	defer s.reg.mu.Unlock()
	s.reg.online[s.peerID] = false
	return nil
}
