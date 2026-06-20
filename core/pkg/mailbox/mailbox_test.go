package mailbox

import "testing"

func TestPutFetchClears(t *testing.T) {
	m := NewMemory()
	m.Put("fp1", []byte("a"))
	m.Put("fp1", []byte("b"))
	m.Put("fp2", []byte("c"))

	if m.Pending("fp1") != 2 {
		t.Errorf("fp1 pending = %d", m.Pending("fp1"))
	}
	got, _ := m.Fetch("fp1")
	if len(got) != 2 || string(got[0]) != "a" || string(got[1]) != "b" {
		t.Errorf("unexpected fetch: %q", got)
	}
	if m.Pending("fp1") != 0 {
		t.Error("fetch should clear the mailbox")
	}
	if m.Pending("fp2") != 1 {
		t.Error("fp2 should be untouched")
	}
}
