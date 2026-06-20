package messaging

import "encoding/json"

// ReceiptBody is the payload for a TypeReceipt envelope: it acknowledges a
// previously received message by id and reports its new delivery state
// (DELIVERED when it reaches the peer, READ when the peer opens it).
// See docs/PROTOCOL.md §4.2.
type ReceiptBody struct {
	RefMsgID string        `json:"refMsgId"`
	State    DeliveryState `json:"state"`
}

// NewReceipt builds a RECEIPT envelope acknowledging refMsgID with state, which
// must be StateDelivered or StateRead.
func NewReceipt(convID, refMsgID string, state DeliveryState) (Envelope, error) {
	body, err := json.Marshal(ReceiptBody{RefMsgID: refMsgID, State: state})
	if err != nil {
		return Envelope{}, err
	}
	return NewEnvelope(TypeReceipt, convID, body), nil
}

// Receipt extracts the ReceiptBody from a RECEIPT envelope.
func (e Envelope) Receipt() (ReceiptBody, error) {
	var r ReceiptBody
	err := json.Unmarshal(e.Body, &r)
	return r, err
}
