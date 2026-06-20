package node

import (
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/transport"
)

func TestReadReceipts(t *testing.T) {
	alice, bob := newNode(t), newNode(t)
	ca, cb := transport.Pipe()
	linkA, err := alice.Dial(ca, bob.Bundle())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	linkB, err := bob.Accept(cb)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}

	sent, err := linkA.SendText(alice, "c1", "hello")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	got, err := linkB.Receive()
	if err != nil || got.Type != messaging.TypeText {
		t.Fatalf("receive text: %v %s", err, got.Type)
	}

	// Bob acknowledges delivery, then read.
	for _, state := range []messaging.DeliveryState{messaging.StateDelivered, messaging.StateRead} {
		if err := linkB.SendReceipt("c1", got.MsgID, state); err != nil {
			t.Fatalf("send receipt %s: %v", state, err)
		}
		ack, err := linkA.Receive()
		if err != nil {
			t.Fatalf("receive receipt: %v", err)
		}
		if ack.Type != messaging.TypeReceipt {
			t.Fatalf("expected RECEIPT, got %s", ack.Type)
		}
		rb, err := ack.Receipt()
		if err != nil {
			t.Fatalf("decode receipt: %v", err)
		}
		if rb.RefMsgID != sent.MsgID {
			t.Errorf("receipt refMsgId = %q, want %q", rb.RefMsgID, sent.MsgID)
		}
		if rb.State != state {
			t.Errorf("receipt state = %q, want %q", rb.State, state)
		}
	}
}
