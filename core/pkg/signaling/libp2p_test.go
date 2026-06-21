package signaling

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLibp2pSDPExchange connects two libp2p signalers directly and verifies the
// SDP offer/answer round trip over libp2p streams (the exchange the WebRTC
// transport relies on).
func TestLibp2pSDPExchange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, err := NewDHT(ctx)
	if err != nil {
		t.Fatalf("new a: %v", err)
	}
	defer a.Close()
	b, err := NewDHT(ctx)
	if err != nil {
		t.Fatalf("new b: %v", err)
	}
	defer b.Close()

	if err := a.Connect(b.AddrInfo()); err != nil {
		t.Fatalf("connect: %v", err)
	}

	// A (offerer) -> B.
	errc := make(chan error, 1)
	go func() { errc <- a.SendSDP(b.ID(), []byte("offer-sdp")) }()
	offer, err := b.RecvSDP("")
	if err != nil || string(offer) != "offer-sdp" {
		t.Fatalf("recv offer: %v %q", err, offer)
	}
	if err := <-errc; err != nil {
		t.Fatalf("send offer: %v", err)
	}

	// B replies to the most recent remote (A) with an empty peerID; A (the
	// offerer) reads the answer addressed to B's peer ID, as the transport does.
	go func() { errc <- b.SendSDP("", []byte("answer-sdp")) }()
	answer, err := a.RecvSDP(b.ID())
	if err != nil || string(answer) != "answer-sdp" {
		t.Fatalf("recv answer: %v %q", err, answer)
	}
	if err := <-errc; err != nil {
		t.Fatalf("send answer: %v", err)
	}

	if ok, _ := a.Presence(b.ID()); !ok {
		t.Error("expected b present to a")
	}
}

// TestLibp2pDHTDiscovery exercises advertise/find over the Kademlia DHT. It is
// opt-in (set STUNNER_DHT_TEST=1) because content-routing propagation timing in
// a tiny two-node DHT is environment-sensitive and can be slow.
func TestLibp2pDHTDiscovery(t *testing.T) {
	if os.Getenv("STUNNER_DHT_TEST") == "" {
		t.Skip("set STUNNER_DHT_TEST=1 to run the DHT discovery test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	a, _ := NewDHT(ctx)
	defer a.Close()
	b, _ := NewDHT(ctx)
	defer b.Close()
	if err := a.Connect(b.AddrInfo()); err != nil {
		t.Fatalf("connect: %v", err)
	}

	key := []byte("discovery-key-123")
	if err := b.Advertise(key); err != nil {
		t.Fatalf("advertise: %v", err)
	}

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := a.Find(key); err == nil && info.PeerID == b.ID() {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("did not discover advertised peer via DHT")
}
