package signaling

import (
	"encoding/hex"
	"sync"
)

// Memory is an in-process Signaler used for tests and the headless harness. It
// models discovery, SDP/ICE exchange, and presence without networking; the
// libp2p DHT (NewDHT) is the production path. Nodes sharing the same *Registry
// can discover and signal each other.
type Registry struct {
	mu      sync.Mutex
	advert  map[string]string           // discoveryKey(hex) -> peerID
	online  map[string]bool             // peerID -> online
	mailbox map[string]chan signalFrame // peerID -> inbox
}

type signalFrame struct {
	from string
	kind string // "sdp" | "candidate"
	data []byte
}

// NewRegistry creates a shared in-process signaling fabric.
func NewRegistry() *Registry {
	return &Registry{
		advert:  map[string]string{},
		online:  map[string]bool{},
		mailbox: map[string]chan signalFrame{},
	}
}

// Join returns a Signaler for peerID bound to this registry.
func (r *Registry) Join(peerID string) Signaler {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.mailbox[peerID]; !ok {
		r.mailbox[peerID] = make(chan signalFrame, 64)
	}
	r.online[peerID] = true
	return &memSignaler{reg: r, peerID: peerID}
}

type memSignaler struct {
	reg    *Registry
	peerID string
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

func (s *memSignaler) send(to, kind string, data []byte) error {
	s.reg.mu.Lock()
	ch, ok := s.reg.mailbox[to]
	s.reg.mu.Unlock()
	if !ok {
		return ErrNotImplemented
	}
	ch <- signalFrame{from: s.peerID, kind: kind, data: append([]byte(nil), data...)}
	return nil
}

func (s *memSignaler) recv(kind string) ([]byte, error) {
	s.reg.mu.Lock()
	ch := s.reg.mailbox[s.peerID]
	s.reg.mu.Unlock()
	for f := range ch {
		if f.kind == kind {
			return f.data, nil
		}
	}
	return nil, ErrNotImplemented
}

func (s *memSignaler) SendSDP(peerID string, sdp []byte) error { return s.send(peerID, "sdp", sdp) }
func (s *memSignaler) RecvSDP(peerID string) ([]byte, error)   { return s.recv("sdp") }
func (s *memSignaler) SendCandidate(peerID string, c []byte) error {
	return s.send(peerID, "candidate", c)
}
func (s *memSignaler) RecvCandidate(peerID string) ([]byte, error) { return s.recv("candidate") }

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
