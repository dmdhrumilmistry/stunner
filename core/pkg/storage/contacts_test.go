package storage

import (
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/contact"
	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
)

func TestContactsCRUD(t *testing.T) {
	s, _, _ := open(t)
	id, _ := identity.Generate()
	c := contact.New("alice", id.SigningPub)

	if err := s.SaveContact(c); err != nil {
		t.Fatalf("save contact: %v", err)
	}
	cs, _ := s.Contacts()
	if len(cs) != 1 || cs[0].Handle != "alice" {
		t.Fatalf("contacts = %+v", cs)
	}
	if err := s.DeleteContact("alice"); err != nil {
		t.Fatalf("delete contact: %v", err)
	}
	cs, _ = s.Contacts()
	if len(cs) != 0 {
		t.Errorf("expected no contacts, got %d", len(cs))
	}
}

func TestDeleteMessageAndConversation(t *testing.T) {
	s, _, _ := open(t)
	s.UpsertConversation(messaging.Conversation{ID: "conv1", DisplayName: "alice"})
	a, _ := messaging.NewText("conv1", "first")
	b, _ := messaging.NewText("conv1", "second")
	s.AppendMessage("conv1", a, messaging.StateSent)
	s.AppendMessage("conv1", b, messaging.StateSent)

	if err := s.DeleteMessage("conv1", a.MsgID); err != nil {
		t.Fatalf("delete message: %v", err)
	}
	msgs, _ := s.Messages("conv1", 0, 0)
	if len(msgs) != 1 || msgs[0].MsgID != b.MsgID {
		t.Fatalf("after delete, messages = %+v", msgs)
	}

	if err := s.DeleteConversation("conv1"); err != nil {
		t.Fatalf("delete conversation: %v", err)
	}
	convs, _ := s.Conversations()
	if len(convs) != 0 {
		t.Errorf("expected no conversations, got %d", len(convs))
	}
	if m, _ := s.Messages("conv1", 0, 0); len(m) != 0 {
		t.Errorf("expected no messages after conversation delete, got %d", len(m))
	}
}
