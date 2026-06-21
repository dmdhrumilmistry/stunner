package signaling

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

// signalProtocol is the libp2p stream protocol used to exchange SDP/ICE.
const signalProtocol protocol.ID = "/stunner/signal/1.0.0"

// DHTSignaler implements Signaler.
var _ Signaler = (*DHTSignaler)(nil)

// DHTSignaler is the production Signaler: a libp2p host participating in a
// Kademlia DHT for decentralized discovery, exchanging WebRTC SDP/ICE over
// authenticated libp2p streams (transport-encrypted by libp2p). It implements
// Signaler. The in-memory Registry remains for tests and the headless harness.
type DHTSignaler struct {
	ctx    context.Context
	cancel context.CancelFunc
	host   host.Host
	kad    *dht.IpfsDHT
	disc   *drouting.RoutingDiscovery

	mu         sync.Mutex
	lastRemote peer.ID
	answers    map[peer.ID]chan []byte // per-peer answer queues (offerer side)

	offerIn chan dhtOffer // inbound offers (answerer side)
	candIn  chan []byte
}

type dhtOffer struct {
	from peer.ID
	data []byte
}

// NewDHT creates a libp2p host (listening on listenAddrs, or an ephemeral
// localhost TCP port if none are given) and starts a Kademlia DHT in server
// mode. Bootstrap by connecting to known peers via Connect.
func NewDHT(ctx context.Context, listenAddrs ...string) (*DHTSignaler, error) {
	cctx, cancel := context.WithCancel(ctx)
	opts := []libp2p.Option{}
	if len(listenAddrs) == 0 {
		listenAddrs = []string{"/ip4/127.0.0.1/tcp/0"}
	}
	opts = append(opts, libp2p.ListenAddrStrings(listenAddrs...))

	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, err
	}
	kad, err := dht.New(cctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		h.Close()
		cancel()
		return nil, err
	}
	if err := kad.Bootstrap(cctx); err != nil {
		kad.Close()
		h.Close()
		cancel()
		return nil, err
	}
	s := &DHTSignaler{
		ctx:     cctx,
		cancel:  cancel,
		host:    h,
		kad:     kad,
		disc:    drouting.NewRoutingDiscovery(kad),
		answers: map[peer.ID]chan []byte{},
		offerIn: make(chan dhtOffer, 8),
		candIn:  make(chan []byte, 8),
	}
	h.SetStreamHandler(signalProtocol, s.onStream)
	return s, nil
}

// BootstrapPublic connects to the public IPFS DHT bootstrap peers so discovery
// works across the internet. Best-effort: individual dial failures are ignored.
// Call after NewDHT for production (real cross-device) use.
func (s *DHTSignaler) BootstrapPublic() {
	var wg sync.WaitGroup
	for _, ai := range dht.GetDefaultBootstrapPeerAddrInfos() {
		wg.Add(1)
		go func(info peer.AddrInfo) {
			defer wg.Done()
			_ = s.host.Connect(s.ctx, info)
		}(ai)
	}
	wg.Wait()
}

func (s *DHTSignaler) answerChan(from peer.ID) chan []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.answers[from]
	if !ok {
		ch = make(chan []byte, 8)
		s.answers[from] = ch
	}
	return ch
}

// ID returns this node's libp2p peer ID (the peerID used by Send*/Recv*).
func (s *DHTSignaler) ID() string { return s.host.ID().String() }

// AddrInfo returns this node's dialable address info, for bootstrapping peers.
func (s *DHTSignaler) AddrInfo() peer.AddrInfo {
	return peer.AddrInfo{ID: s.host.ID(), Addrs: s.host.Addrs()}
}

// Connect dials another node directly (used for bootstrapping / tests).
func (s *DHTSignaler) Connect(info peer.AddrInfo) error {
	s.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	return s.host.Connect(s.ctx, info)
}

func (s *DHTSignaler) Advertise(discoveryKey []byte) error {
	dutil.Advertise(s.ctx, s.disc, hex.EncodeToString(discoveryKey))
	return nil
}

func (s *DHTSignaler) Find(discoveryKey []byte) (PeerInfo, error) {
	ch, err := s.disc.FindPeers(s.ctx, hex.EncodeToString(discoveryKey))
	if err != nil {
		return PeerInfo{}, err
	}
	for ai := range ch {
		if ai.ID == s.host.ID() || len(ai.Addrs) == 0 {
			continue
		}
		s.host.Peerstore().AddAddrs(ai.ID, ai.Addrs, peerstore.PermanentAddrTTL)
		addrs := make([]string, 0, len(ai.Addrs))
		for _, a := range ai.Addrs {
			addrs = append(addrs, a.String())
		}
		return PeerInfo{PeerID: ai.ID.String(), Addresses: addrs}, nil
	}
	return PeerInfo{}, errors.New("signaling: peer not found in DHT")
}

// SendSDP routes an offer (peerID != "") or an answer (peerID == "", to the peer
// we last received an offer from), mirroring the transport's convention.
func (s *DHTSignaler) SendSDP(peerID string, sdp []byte) error {
	if peerID != "" {
		return s.sendStream(peerID, 'o', sdp)
	}
	return s.sendStream("", 'a', sdp)
}

// RecvSDP returns the next inbound offer (peerID == "") or the answer from a
// specific peer (peerID != ""). Separate queues keep a node that both listens
// and connects from consuming the wrong message.
func (s *DHTSignaler) RecvSDP(peerID string) ([]byte, error) {
	if peerID == "" {
		select {
		case f := <-s.offerIn:
			s.mu.Lock()
			s.lastRemote = f.from
			s.mu.Unlock()
			return f.data, nil
		case <-s.ctx.Done():
			return nil, s.ctx.Err()
		}
	}
	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, err
	}
	select {
	case b := <-s.answerChan(pid):
		return b, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func (s *DHTSignaler) SendCandidate(peerID string, candidate []byte) error {
	return s.sendStream(peerID, 'c', candidate)
}

func (s *DHTSignaler) RecvCandidate(string) ([]byte, error) {
	select {
	case b := <-s.candIn:
		return b, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func (s *DHTSignaler) Presence(peerID string) (bool, error) {
	pid, err := peer.Decode(peerID)
	if err != nil {
		return false, err
	}
	return s.host.Network().Connectedness(pid) == network.Connected, nil
}

func (s *DHTSignaler) Close() error {
	s.cancel()
	_ = s.kad.Close()
	return s.host.Close()
}

// sendStream opens a one-shot stream to the target and writes a single framed
// message. An empty peerID replies to the most recent inbound peer, which is how
// an answerer routes its SDP answer back to the offerer.
func (s *DHTSignaler) sendStream(peerID string, kind byte, data []byte) error {
	var pid peer.ID
	if peerID == "" {
		s.mu.Lock()
		pid = s.lastRemote
		s.mu.Unlock()
		if pid == "" {
			return errors.New("signaling: no peer to reply to")
		}
	} else {
		decoded, err := peer.Decode(peerID)
		if err != nil {
			return err
		}
		pid = decoded
	}
	stream, err := s.host.NewStream(s.ctx, pid, signalProtocol)
	if err != nil {
		return err
	}
	defer stream.Close()
	return writeFrame(stream, kind, data)
}

func (s *DHTSignaler) onStream(stream network.Stream) {
	defer stream.Close()
	remote := stream.Conn().RemotePeer()

	kind, data, err := readFrame(stream)
	if err != nil {
		return
	}
	switch kind {
	case 'o': // offer
		s.mu.Lock()
		s.lastRemote = remote
		s.mu.Unlock()
		select {
		case s.offerIn <- dhtOffer{from: remote, data: data}:
		case <-s.ctx.Done():
		}
	case 'a': // answer
		select {
		case s.answerChan(remote) <- data:
		case <-s.ctx.Done():
		}
	case 'c':
		select {
		case s.candIn <- data:
		case <-s.ctx.Done():
		}
	}
}

// writeFrame writes kind(1) || len(4, big-endian) || data.
func writeFrame(w io.Writer, kind byte, data []byte) error {
	var hdr [5]byte
	hdr[0] = kind
	binary.BigEndian.PutUint32(hdr[1:], uint32(len(data)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func readFrame(r io.Reader) (byte, []byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	n := binary.BigEndian.Uint32(hdr[1:])
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, nil, err
	}
	return hdr[0], buf, nil
}
