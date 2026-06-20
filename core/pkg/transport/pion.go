package transport

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"

	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
)

// connectTimeout bounds how long Dial/Accept wait for the data channel to open.
const connectTimeout = 30 * time.Second

// pionTransport is the production Transport: WebRTC data channels via pion, using
// the configured STUN/TURN ICE servers. It implements Transport.
//
// Signaling uses non-trickle ICE: each side waits for ICE gathering to complete
// so the full candidate set is embedded in the SDP, and only the SDP offer/answer
// is exchanged over the SignalingExchange (SendSDP/RecvSDP). This keeps the
// exchange to a single round trip and works over any Signaler (DHT or relay).
type pionTransport struct {
	cfg webrtc.Configuration
}

// New constructs a pion-backed Transport from settings. ICE servers (STUN/TURN)
// are taken from cfg and passed straight into the WebRTC configuration, so users
// can override NAT-traversal infrastructure (see pkg/settings).
func New(cfg Config) (Transport, error) {
	return &pionTransport{cfg: webrtc.Configuration{ICEServers: toICEServers(cfg.ICEServers)}}, nil
}

func toICEServers(in []settings.ICEServer) []webrtc.ICEServer {
	out := make([]webrtc.ICEServer, 0, len(in))
	for _, s := range in {
		srv := webrtc.ICEServer{URLs: s.URLs}
		if s.Username != "" {
			srv.Username = s.Username
			srv.Credential = s.Credential
		}
		out = append(out, srv)
	}
	return out
}

// Dial is the offerer: it creates the data channel, sends an SDP offer, and
// applies the peer's answer.
func (t *pionTransport) Dial(peerID string, sig SignalingExchange) (Conn, error) {
	pc, err := webrtc.NewPeerConnection(t.cfg)
	if err != nil {
		return nil, err
	}
	dc, err := pc.CreateDataChannel("stunner", nil)
	if err != nil {
		pc.Close()
		return nil, err
	}
	conn := newPionConn(pc, dc)

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		pc.Close()
		return nil, err
	}
	if err := setLocalAndGather(pc, offer); err != nil {
		pc.Close()
		return nil, err
	}
	local, _ := json.Marshal(pc.LocalDescription())
	if err := sig.SendSDP(peerID, local); err != nil {
		pc.Close()
		return nil, err
	}
	answerRaw, err := sig.RecvSDP(peerID)
	if err != nil {
		pc.Close()
		return nil, err
	}
	var answer webrtc.SessionDescription
	if err := json.Unmarshal(answerRaw, &answer); err != nil {
		pc.Close()
		return nil, err
	}
	if err := pc.SetRemoteDescription(answer); err != nil {
		pc.Close()
		return nil, err
	}
	if err := conn.waitOpen(); err != nil {
		pc.Close()
		return nil, err
	}
	return conn, nil
}

// Accept is the answerer: it waits for the peer's offer, replies with an answer,
// and adopts the inbound data channel.
func (t *pionTransport) Accept(sig SignalingExchange) (Conn, error) {
	pc, err := webrtc.NewPeerConnection(t.cfg)
	if err != nil {
		return nil, err
	}
	dcCh := make(chan *webrtc.DataChannel, 1)
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		select {
		case dcCh <- dc:
		default:
		}
	})

	offerRaw, err := sig.RecvSDP("")
	if err != nil {
		pc.Close()
		return nil, err
	}
	var offer webrtc.SessionDescription
	if err := json.Unmarshal(offerRaw, &offer); err != nil {
		pc.Close()
		return nil, err
	}
	if err := pc.SetRemoteDescription(offer); err != nil {
		pc.Close()
		return nil, err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		pc.Close()
		return nil, err
	}
	if err := setLocalAndGather(pc, answer); err != nil {
		pc.Close()
		return nil, err
	}
	local, _ := json.Marshal(pc.LocalDescription())
	if err := sig.SendSDP("", local); err != nil {
		pc.Close()
		return nil, err
	}

	select {
	case dc := <-dcCh:
		conn := newPionConn(pc, dc)
		if err := conn.waitOpen(); err != nil {
			pc.Close()
			return nil, err
		}
		return conn, nil
	case <-time.After(connectTimeout):
		pc.Close()
		return nil, errors.New("transport: timed out waiting for data channel")
	}
}

func (t *pionTransport) Close() error { return nil }

// setLocalAndGather sets the local description and blocks until ICE gathering is
// complete, so the returned LocalDescription contains all candidates.
func setLocalAndGather(pc *webrtc.PeerConnection, desc webrtc.SessionDescription) error {
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(desc); err != nil {
		return err
	}
	<-gatherComplete
	return nil
}

// pionConn adapts a WebRTC DataChannel to the Conn interface.
type pionConn struct {
	pc   *webrtc.PeerConnection
	dc   *webrtc.DataChannel
	in   chan []byte
	open chan struct{}
	done chan struct{}
	once sync.Once
}

func newPionConn(pc *webrtc.PeerConnection, dc *webrtc.DataChannel) *pionConn {
	c := &pionConn{
		pc:   pc,
		dc:   dc,
		in:   make(chan []byte, 64),
		open: make(chan struct{}),
		done: make(chan struct{}),
	}
	var openOnce sync.Once
	dc.OnOpen(func() { openOnce.Do(func() { close(c.open) }) })
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		select {
		case c.in <- append([]byte(nil), msg.Data...):
		case <-c.done:
		}
	})
	dc.OnClose(func() { c.closeDone() })
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateClosed {
			c.closeDone()
		}
	})
	return c
}

func (c *pionConn) waitOpen() error {
	select {
	case <-c.open:
		return nil
	case <-c.done:
		return ErrClosed
	case <-time.After(connectTimeout):
		return errors.New("transport: timed out opening data channel")
	}
}

func (c *pionConn) Send(b []byte) error {
	select {
	case <-c.done:
		return ErrClosed
	default:
	}
	return c.dc.Send(b)
}

func (c *pionConn) Recv() ([]byte, error) {
	select {
	case b := <-c.in:
		return b, nil
	case <-c.done:
		select {
		case b := <-c.in:
			return b, nil
		default:
			return nil, ErrClosed
		}
	}
}

func (c *pionConn) Close() error {
	c.closeDone()
	return c.pc.Close()
}

func (c *pionConn) closeDone() {
	c.once.Do(func() { close(c.done) })
}
