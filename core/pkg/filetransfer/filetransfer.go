// Package filetransfer implements chunked, integrity-checked file transfer over
// a secure session.
//
// A sender produces a FileOffer (name, size, chunk size, SHA-256 of the
// plaintext, and a per-transfer key carried inside the E2E session) plus a set
// of FileChunks, each sealed with an AEAD (pkg/crypto.SealFile). Chunks may
// arrive out of order and are reassembled and integrity-checked. See
// Split/Reassemble in impl.go and docs/PROTOCOL.md §5.
package filetransfer

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

// Progress reports transfer progress to the UI as chunks are sent/received.
type Progress struct {
	FileID     string
	BytesDone  uint64
	BytesTotal uint64
	Done       bool
	Err        error
}
