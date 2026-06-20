// Package mailbox is the optional, self-hostable, content-blind store-and-forward
// relay that enables offline delivery in Stunner's otherwise pure-P2P model.
//
// It only ever holds end-to-end ciphertext keyed by recipient fingerprint; it
// cannot read messages. It is OFF by default (see pkg/settings.RelayEnabled)
// because it reintroduces a metadata-bearing component — see
// docs/THREAT_MODEL.md. Memory is an in-process implementation used by tests and
// the harness; a networked relay is the production deployment.
package mailbox

import "sync"

// Mailbox stores ciphertext for recipients who are offline.
type Mailbox interface {
	// Put stores one ciphertext message for a recipient.
	Put(recipientFP string, ciphertext []byte) error
	// Fetch returns and removes all queued messages for a recipient.
	Fetch(recipientFP string) ([][]byte, error)
	// Pending reports how many messages are queued for a recipient.
	Pending(recipientFP string) int
}

// Memory is an in-process Mailbox.
type Memory struct {
	mu  sync.Mutex
	box map[string][][]byte
}

// NewMemory creates an empty in-memory mailbox.
func NewMemory() *Memory { return &Memory{box: map[string][][]byte{}} }

func (m *Memory) Put(recipientFP string, ciphertext []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.box[recipientFP] = append(m.box[recipientFP], append([]byte(nil), ciphertext...))
	return nil
}

func (m *Memory) Fetch(recipientFP string) ([][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msgs := m.box[recipientFP]
	delete(m.box, recipientFP)
	return msgs, nil
}

func (m *Memory) Pending(recipientFP string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.box[recipientFP])
}
