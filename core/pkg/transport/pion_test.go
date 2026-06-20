package transport

import (
	"testing"
	"time"
)

// memExchange is an in-memory SignalingExchange endpoint for tests: SDP written
// to one endpoint is read from its peer.
type memExchange struct {
	send chan []byte
	recv chan []byte
}

func newExchangePair() (*memExchange, *memExchange) {
	a := make(chan []byte, 1)
	b := make(chan []byte, 1)
	return &memExchange{send: a, recv: b}, &memExchange{send: b, recv: a}
}

func (m *memExchange) SendSDP(_ string, sdp []byte) error { m.send <- sdp; return nil }
func (m *memExchange) RecvSDP(_ string) ([]byte, error)   { return <-m.recv, nil }
func (m *memExchange) SendCandidate(string, []byte) error { return nil }
func (m *memExchange) RecvCandidate(string) ([]byte, error) {
	select {} // unused: non-trickle ICE
}

func TestPionDataChannelRoundTrip(t *testing.T) {
	// No ICE servers: loopback uses host candidates, keeping the test hermetic.
	tr, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dialerSig, accepterSig := newExchangePair()

	type result struct {
		conn Conn
		err  error
	}
	accCh := make(chan result, 1)
	go func() {
		c, err := tr.Accept(accepterSig)
		accCh <- result{c, err}
	}()

	dialConn, err := tr.Dial("peer", dialerSig)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer dialConn.Close()

	var acc result
	select {
	case acc = <-accCh:
	case <-time.After(connectTimeout):
		t.Fatal("accept timed out")
	}
	if acc.err != nil {
		t.Fatalf("accept: %v", acc.err)
	}
	defer acc.conn.Close()

	// Dialer -> accepter.
	if err := dialConn.Send([]byte("ping 🔒")); err != nil {
		t.Fatalf("send: %v", err)
	}
	got, err := acc.conn.Recv()
	if err != nil || string(got) != "ping 🔒" {
		t.Fatalf("recv: %v %q", err, got)
	}

	// Accepter -> dialer.
	if err := acc.conn.Send([]byte("pong")); err != nil {
		t.Fatalf("send back: %v", err)
	}
	got, err = dialConn.Recv()
	if err != nil || string(got) != "pong" {
		t.Fatalf("recv back: %v %q", err, got)
	}
}
