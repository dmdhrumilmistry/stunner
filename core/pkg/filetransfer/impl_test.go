package filetransfer

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestSplitReassemble(t *testing.T) {
	data := make([]byte, 100*1024+123) // not a chunk multiple
	rand.Read(data)

	offer, chunks, err := Split("photo.jpg", "image/jpeg", data, 16*1024)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if offer.Size != uint64(len(data)) {
		t.Errorf("offer size = %d", offer.Size)
	}

	got, err := Reassemble(offer, chunks)
	if err != nil {
		t.Fatalf("reassemble: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Error("reassembled data does not match original")
	}
}

func TestReassembleOutOfOrder(t *testing.T) {
	data := bytes.Repeat([]byte("stunner"), 5000)
	offer, chunks, _ := Split("f.bin", "application/octet-stream", data, 1024)

	// Reverse the chunk order.
	for i, j := 0, len(chunks)-1; i < j; i, j = i+1, j-1 {
		chunks[i], chunks[j] = chunks[j], chunks[i]
	}
	got, err := Reassemble(offer, chunks)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("out-of-order reassemble failed: %v", err)
	}
}

func TestTamperedChunkFails(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 4096)
	offer, chunks, _ := Split("f", "text/plain", data, 1024)
	chunks[1].Data[0] ^= 0xFF
	if _, err := Reassemble(offer, chunks); err == nil {
		t.Error("expected failure for tampered chunk")
	}
}

func TestMissingChunkFails(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 4096)
	offer, chunks, _ := Split("f", "text/plain", data, 1024)
	if _, err := Reassemble(offer, chunks[:len(chunks)-1]); err == nil {
		t.Error("expected failure for missing chunk")
	}
}
