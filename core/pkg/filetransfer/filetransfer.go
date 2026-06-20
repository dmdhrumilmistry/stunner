// Package filetransfer implements chunked, resumable, integrity-checked file
// transfer over a secure session.
//
// A sender emits a FileOffer (name, size, chunk size, SHA-256 of the plaintext,
// and a per-transfer key carried inside the E2E session). Each FileChunk is
// sealed with an AEAD (pkg/crypto.SealFile) and sent over the data channel.
// Missing chunks can be re-requested, enabling resume. See docs/PROTOCOL.md §5.
//
// This file defines the model and interfaces; implemented in roadmap phase 6.
package filetransfer

import "errors"

// ErrNotImplemented marks skeleton stubs awaiting a roadmap phase.
var ErrNotImplemented = errors.New("filetransfer: not implemented (see docs/ROADMAP.md phase 6)")

// DefaultChunkSize is the default plaintext chunk size before AEAD sealing.
const DefaultChunkSize = 16 * 1024 // 16 KiB

// Offer describes a file being offered to a peer (see docs/PROTOCOL.md §5).
type Offer struct {
	FileID      string `json:"fileId"`
	Name        string `json:"name"`
	Size        uint64 `json:"size"`
	MIME        string `json:"mime"`
	ChunkSize   uint32 `json:"chunkSize"`
	Hash        []byte `json:"hash"`        // SHA-256 of full plaintext
	TransferKey []byte `json:"transferKey"` // per-transfer symmetric key
}

// Chunk is one sealed slice of the file.
type Chunk struct {
	FileID string `json:"fileId"`
	Index  uint32 `json:"index"`
	Data   []byte `json:"data"` // AEAD-sealed
}

// Progress reports transfer progress to the UI.
type Progress struct {
	FileID     string
	BytesDone  uint64
	BytesTotal uint64
	Done       bool
	Err        error
}

// Sender drives an outgoing transfer.
type Sender interface {
	// Start begins sending the file at path to peerID, reporting progress.
	Start(peerID, path string, report func(Progress)) (Offer, error)
}

// Receiver drives an incoming transfer.
type Receiver interface {
	// Accept stores an offered file to destPath, verifying integrity on
	// completion. Declining is handled via a CONTROL message.
	Accept(offer Offer, destPath string, report func(Progress)) error
}
