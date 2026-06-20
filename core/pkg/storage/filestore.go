package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
	"github.com/dmdhrumilmistry/stunner/core/pkg/vault"
)

// fileStore is a Store backed by a single vault-sealed JSON file. Every mutation
// re-seals and writes the whole document; this is simple and fully encrypted at
// rest, suitable as the reference implementation.
type fileStore struct {
	mu   sync.Mutex
	path string
	key  []byte
	data dbData
}

type dbData struct {
	Settings      *settings.Settings                `json:"settings,omitempty"`
	Conversations map[string]messaging.Conversation `json:"conversations"`
	Messages      map[string][]storedMessage        `json:"messages"`
	Outbox        []storedMessage                   `json:"outbox"`
	Blobs         map[string][]byte                 `json:"blobs"`
}

type storedMessage struct {
	Envelope messaging.Envelope      `json:"envelope"`
	State    messaging.DeliveryState `json:"state"`
	ConvID   string                  `json:"convId"`
}

func openFileStore(opts Options) (Store, error) {
	if len(opts.Key) != vault.KeySize {
		return nil, errors.New("storage: key must be 32 bytes")
	}
	if opts.Path == "" {
		return nil, errors.New("storage: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o700); err != nil {
		return nil, err
	}
	s := &fileStore{
		path: opts.Path,
		key:  opts.Key,
		data: dbData{
			Conversations: map[string]messaging.Conversation{},
			Messages:      map[string][]storedMessage{},
			Blobs:         map[string][]byte{},
		},
	}
	sealed, err := os.ReadFile(opts.Path)
	if err == nil {
		blob, oerr := vault.Open(opts.Key, sealed)
		if oerr != nil {
			return nil, oerr
		}
		if err := json.Unmarshal(blob, &s.data); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

// flush must be called with s.mu held.
func (s *fileStore) flush() error {
	blob, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	sealed, err := vault.Seal(s.key, blob)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, sealed, 0o600)
}

func (s *fileStore) LoadSettings() (settings.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Settings == nil {
		return settings.Default(), nil
	}
	return *s.data.Settings, nil
}

func (s *fileStore) SaveSettings(set settings.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Settings = &set
	return s.flush()
}

func (s *fileStore) Conversations() ([]messaging.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]messaging.Conversation, 0, len(s.data.Conversations))
	for _, c := range s.data.Conversations {
		out = append(out, c)
	}
	return out, nil
}

func (s *fileStore) UpsertConversation(c messaging.Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Conversations[c.ID] = c
	return s.flush()
}

func (s *fileStore) AppendMessage(convID string, env messaging.Envelope, state messaging.DeliveryState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Messages[convID] = append(s.data.Messages[convID], storedMessage{Envelope: env, State: state, ConvID: convID})
	return s.flush()
}

func (s *fileStore) Messages(convID string, limit, offset int) ([]messaging.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.data.Messages[convID]
	if offset > len(all) {
		offset = len(all)
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	out := make([]messaging.Envelope, len(all))
	for i, m := range all {
		out[i] = m.Envelope
	}
	return out, nil
}

func (s *fileStore) EnqueueOutbox(convID string, env messaging.Envelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Outbox = append(s.data.Outbox, storedMessage{Envelope: env, State: messaging.StateQueued, ConvID: convID})
	return s.flush()
}

func (s *fileStore) PendingOutbox() ([]messaging.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]messaging.Envelope, len(s.data.Outbox))
	for i, m := range s.data.Outbox {
		out[i] = m.Envelope
	}
	return out, nil
}

func (s *fileStore) SaveBlob(namespace, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Blobs[namespace+"/"+key] = append([]byte(nil), value...)
	return s.flush()
}

func (s *fileStore) LoadBlob(namespace, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data.Blobs[namespace+"/"+key]
	if !ok {
		return nil, errors.New("storage: blob not found")
	}
	return append([]byte(nil), v...), nil
}

func (s *fileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flush()
}
