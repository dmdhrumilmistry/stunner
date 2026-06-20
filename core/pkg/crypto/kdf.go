package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
)

// hkdf implements RFC 5869 (extract-and-expand) over HMAC-SHA256. It is used to
// derive root, chain, and message keys in the Double Ratchet. Built on stdlib
// primitives only.
func hkdf(salt, ikm, info []byte, length int) []byte {
	if len(salt) == 0 {
		salt = make([]byte, sha256.Size)
	}
	// Extract.
	ext := hmac.New(sha256.New, salt)
	ext.Write(ikm)
	prk := ext.Sum(nil)

	// Expand.
	var out, t []byte
	for counter := byte(1); len(out) < length; counter++ {
		exp := hmac.New(sha256.New, prk)
		exp.Write(t)
		exp.Write(info)
		exp.Write([]byte{counter})
		t = exp.Sum(nil)
		out = append(out, t...)
	}
	return out[:length]
}

// hmacSHA256 returns HMAC-SHA256(key, data). Used for the symmetric-key ratchet.
func hmacSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}
