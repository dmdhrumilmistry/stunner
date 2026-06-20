// Package safetynumber computes Signal-style numeric "safety numbers" that two
// users compare out of band (in person, over a call, or by scanning a QR code)
// to verify they share authentic identity keys and defeat active MITM attacks.
//
// The algorithm mirrors Signal's NumericFingerprintGenerator: each party's key
// is hashed with 5200 iterations of SHA-512, the first 30 bytes are turned into
// six 5-digit chunks, and the two 30-digit halves are concatenated in sorted
// order so both sides display the same 60-digit number. Stdlib only.
package safetynumber

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	version    = 0
	iterations = 5200
)

// Compute returns the 60-digit safety number for the two identity keys,
// formatted in groups of five digits. The result is identical regardless of
// argument order.
func Compute(a, b ed25519.PublicKey) string {
	na := numeric(a)
	nb := numeric(b)
	var combined string
	if na <= nb {
		combined = na + nb
	} else {
		combined = nb + na
	}
	return group(combined)
}

// numeric returns the 30-digit fingerprint for a single public key. In Stunner
// the identity key is its own stable identifier.
func numeric(pub ed25519.PublicKey) string {
	var ver [2]byte
	binary.BigEndian.PutUint16(ver[:], version)
	data := append(append(append([]byte{}, ver[:]...), pub...), pub...)
	for i := 0; i < iterations; i++ {
		h := sha512.New()
		h.Write(data)
		h.Write(pub)
		data = h.Sum(nil)
	}
	var b strings.Builder
	for offset := 0; offset < 30; offset += 5 {
		b.WriteString(chunk(data, offset))
	}
	return b.String()
}

// chunk encodes 5 bytes at offset into a zero-padded 5-digit string.
func chunk(hash []byte, offset int) string {
	v := uint64(hash[offset])<<32 |
		uint64(hash[offset+1])<<24 |
		uint64(hash[offset+2])<<16 |
		uint64(hash[offset+3])<<8 |
		uint64(hash[offset+4])
	return fmt.Sprintf("%05d", v%100000)
}

func group(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += 5 {
		if i > 0 {
			b.WriteByte(' ')
		}
		end := i + 5
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[i:end])
	}
	return b.String()
}
