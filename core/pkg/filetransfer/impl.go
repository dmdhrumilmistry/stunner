package filetransfer

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sort"

	"github.com/dmdhrumilmistry/stunner/core/pkg/crypto"
)

// Split chops data into AEAD-sealed chunks and produces the matching Offer
// (including a fresh per-transfer key and the SHA-256 of the plaintext). The
// Offer travels inside the E2E session; chunks travel over the data channel.
func Split(name, mime string, data []byte, chunkSize uint32) (Offer, []Chunk, error) {
	if chunkSize == 0 {
		chunkSize = DefaultChunkSize
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return Offer{}, nil, err
	}
	sum := sha256.Sum256(data)
	offer := Offer{
		FileID:      newFileID(),
		Name:        name,
		Size:        uint64(len(data)),
		MIME:        mime,
		ChunkSize:   chunkSize,
		Hash:        sum[:],
		TransferKey: key,
	}

	var chunks []Chunk
	for i, off := uint32(0), 0; off < len(data); i++ {
		end := off + int(chunkSize)
		if end > len(data) {
			end = len(data)
		}
		sealed, err := crypto.SealFile(key, nonceFor(i), data[off:end])
		if err != nil {
			return Offer{}, nil, err
		}
		chunks = append(chunks, Chunk{FileID: offer.FileID, Index: i, Data: sealed})
		off = end
	}
	return offer, chunks, nil
}

// Reassemble decrypts and concatenates chunks in index order and verifies the
// result against the offer's hash. Chunks may arrive out of order.
func Reassemble(offer Offer, chunks []Chunk) ([]byte, error) {
	ordered := append([]Chunk(nil), chunks...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Index < ordered[j].Index })

	var out []byte
	for i, c := range ordered {
		if c.Index != uint32(i) {
			return nil, errors.New("filetransfer: missing or duplicate chunk")
		}
		pt, err := crypto.OpenFile(offer.TransferKey, nonceFor(c.Index), c.Data)
		if err != nil {
			return nil, err
		}
		out = append(out, pt...)
	}
	if uint64(len(out)) != offer.Size {
		return nil, errors.New("filetransfer: size mismatch")
	}
	sum := sha256.Sum256(out)
	if !equalHash(sum[:], offer.Hash) {
		return nil, errors.New("filetransfer: integrity check failed")
	}
	return out, nil
}

// nonceFor derives a unique 12-byte GCM nonce from a chunk index. Each transfer
// uses a fresh key, so a per-transfer counter nonce is safe.
func nonceFor(index uint32) []byte {
	n := make([]byte, 12)
	binary.BigEndian.PutUint32(n[8:], index)
	return n
}

func newFileID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	const hexd = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexd[v>>4]
		out[i*2+1] = hexd[v&0x0f]
	}
	return string(out)
}

func equalHash(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
