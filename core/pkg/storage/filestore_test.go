package storage

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
)

func open(t *testing.T) (Store, string, []byte) {
	t.Helper()
	key := bytes.Repeat([]byte{4}, 32)
	path := filepath.Join(t.TempDir(), "db.bin")
	s, err := Open(Options{Path: path, Key: key})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return s, path, key
}

func TestSettingsAndMessagesPersist(t *testing.T) {
	s, path, key := open(t)

	set, _ := s.LoadSettings()
	set.RelayEnabled = true
	if err := s.SaveSettings(set); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	env, _ := messaging.NewText("c1", "hi")
	if err := s.AppendMessage("c1", env, messaging.StateSent); err != nil {
		t.Fatalf("append: %v", err)
	}
	s.Close()

	// Reopen and confirm encrypted-at-rest data round-trips.
	s2, err := Open(Options{Path: path, Key: key})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	set2, _ := s2.LoadSettings()
	if !set2.RelayEnabled {
		t.Error("settings not persisted")
	}
	msgs, _ := s2.Messages("c1", 0, 0)
	if len(msgs) != 1 || msgs[0].MsgID != env.MsgID {
		t.Errorf("messages not persisted: %+v", msgs)
	}
}

func TestWrongKeyCannotOpen(t *testing.T) {
	s, path, _ := open(t)
	s.Close()
	if _, err := Open(Options{Path: path, Key: bytes.Repeat([]byte{9}, 32)}); err == nil {
		t.Error("expected failure opening with wrong key")
	}
}

func TestBlobs(t *testing.T) {
	s, _, _ := open(t)
	if err := s.SaveBlob("session", "peer1", []byte("state")); err != nil {
		t.Fatalf("save blob: %v", err)
	}
	v, err := s.LoadBlob("session", "peer1")
	if err != nil || string(v) != "state" {
		t.Fatalf("load blob: %v %q", err, v)
	}
	if _, err := s.LoadBlob("session", "missing"); err == nil {
		t.Error("expected error for missing blob")
	}
}
