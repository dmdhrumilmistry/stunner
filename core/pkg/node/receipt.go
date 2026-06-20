package node

import "github.com/dmdhrumilmistry/stunner/core/pkg/messaging"

// SendReceipt acknowledges a received message to the peer with a delivery state
// (messaging.StateDelivered when it arrives, messaging.StateRead when the user
// opens the conversation). Receipts are control messages and are not stored as
// conversation messages.
func (l *Link) SendReceipt(convID, refMsgID string, state messaging.DeliveryState) error {
	env, err := messaging.NewReceipt(convID, refMsgID, state)
	if err != nil {
		return err
	}
	pt, err := env.Encode()
	if err != nil {
		return err
	}
	ct, err := l.session.Encrypt(pt)
	if err != nil {
		return err
	}
	frame, err := messaging.EncodeFrame(messaging.Frame{Payload: ct})
	if err != nil {
		return err
	}
	return l.conn.Send(frame)
}
