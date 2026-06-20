package core

import (
	"strings"
	"testing"
)

func TestNewContactURIAndValidate(t *testing.T) {
	uri, err := NewContactURI("alice")
	if err != nil {
		t.Fatalf("NewContactURI: %v", err)
	}
	handle, fp, err := ValidateContactURI(uri)
	if err != nil {
		t.Fatalf("ValidateContactURI: %v", err)
	}
	if handle != "alice" || fp == "" {
		t.Errorf("handle=%q fp=%q", handle, fp)
	}
}

func TestSafetyNumberSymmetricViaURIs(t *testing.T) {
	a, _ := NewContactURI("a")
	b, _ := NewContactURI("b")
	snAB, err := SafetyNumber(a, b)
	if err != nil {
		t.Fatalf("SafetyNumber: %v", err)
	}
	snBA, _ := SafetyNumber(b, a)
	if snAB != snBA {
		t.Error("safety number should be order-independent")
	}
	if len(strings.ReplaceAll(snAB, " ", "")) != 60 {
		t.Errorf("expected 60 digits: %q", snAB)
	}
}

func TestValidateRejectsJunk(t *testing.T) {
	if _, _, err := ValidateContactURI("not-a-uri"); err == nil {
		t.Error("expected error")
	}
}
