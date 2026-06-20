// Package messaging defines the message model and the send/receive pipeline.
//
// Application payloads are wrapped in an Envelope (see docs/PROTOCOL.md §4),
// encrypted by pkg/crypto, and carried by pkg/transport. Because Stunner is pure
// P2P with no store-and-forward server, outgoing messages are queued in an
// outbox and retried when the peer becomes reachable (presence from
// pkg/signaling). An optional relay can provide true offline delivery.
//
// This file defines the model and interfaces; the pipeline is implemented across
// roadmap phases 3–5.
package messaging

import (
	"errors"
	"time"
)

// ErrNotImplemented marks skeleton stubs awaiting a roadmap phase.
var ErrNotImplemented = errors.New("messaging: not implemented (see docs/ROADMAP.md)")

// Type enumerates application payload types (see docs/PROTOCOL.md §4).
type Type string

const (
	TypeText      Type = "TEXT"
	TypeFileOffer Type = "FILE_OFFER"
	TypeFileChunk Type = "FILE_CHUNK"
	TypeReceipt   Type = "RECEIPT"
	TypeTyping    Type = "TYPING"
	TypeControl   Type = "CONTROL"
)

// Envelope is the type-tagged container for every application payload. The whole
// envelope is serialized and then encrypted by the secure session.
type Envelope struct {
	Version   int       `json:"version"`
	Type      Type      `json:"type"`
	MsgID     string    `json:"msgId"`
	ConvID    string    `json:"convId"`
	Timestamp time.Time `json:"timestamp"`
	Body      []byte    `json:"body"` // type-specific payload
}

// TextBody is the payload for TypeText. Text is UTF-8 and may contain Unicode
// emoji; animated emoji are referenced by id (see pkg/emoji), not embedded.
type TextBody struct {
	Text          string   `json:"text"`
	AnimatedEmoji []string `json:"animatedEmoji,omitempty"` // emoji ids
}

// DeliveryState tracks how far an outgoing message has progressed.
type DeliveryState string

const (
	StateQueued    DeliveryState = "QUEUED"
	StateSent      DeliveryState = "SENT"
	StateDelivered DeliveryState = "DELIVERED"
	StateRead      DeliveryState = "READ"
	StateFailed    DeliveryState = "FAILED"
)

// Conversation is a 1:1 thread (group chat is future work).
type Conversation struct {
	ID          string
	PeerID      string // peer identity (public key, see pkg/identity)
	DisplayName string
}

// Service is the high-level messaging API the FFI layer exposes to the UI.
type Service interface {
	// SendText queues a text message to a conversation, returning its msgId.
	SendText(convID, text string) (msgID string, err error)
	// Subscribe delivers inbound envelopes and delivery-state changes to the UI.
	Subscribe(handler func(Event)) (cancel func())
}

// Event is pushed to UI subscribers (inbound messages, receipts, presence).
type Event struct {
	Kind     string    // "message" | "receipt" | "presence"
	ConvID   string    `json:"convId,omitempty"`
	Envelope *Envelope `json:"envelope,omitempty"`
	State    DeliveryState
}
