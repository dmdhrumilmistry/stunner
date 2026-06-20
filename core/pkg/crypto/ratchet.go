package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

// maxSkip bounds how many missed messages we will derive keys for in one chain,
// preventing a malicious header from forcing unbounded work.
const maxSkip = 1000

// ratchet holds Double Ratchet state for one peer. It is not safe for
// concurrent use; the owning session serializes access.
type ratchet struct {
	dhs *ecdh.PrivateKey // our current ratchet keypair
	dhr []byte           // their current ratchet public key (32 bytes), or nil

	rk  []byte // root key
	cks []byte // sending chain key
	ckr []byte // receiving chain key

	ns uint32 // messages sent in current sending chain
	nr uint32 // messages received in current receiving chain
	pn uint32 // messages in previous sending chain

	skipped map[string][]byte // "hex(dhr):n" -> message key
	ad      []byte            // associated data bound into every message
}

// ratchetHeader is the per-message routing info, authenticated as AEAD AD.
type ratchetHeader struct {
	DH []byte `json:"dh"`
	PN uint32 `json:"pn"`
	N  uint32 `json:"n"`
}

func newRatchetInitiator(sk, remoteRatchetPub, ad []byte) (*ratchet, error) {
	dhs, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	r := &ratchet{dhr: remoteRatchetPub, ad: ad, skipped: map[string][]byte{}}
	r.dhs = dhs
	out, err := r.dh(remoteRatchetPub)
	if err != nil {
		return nil, err
	}
	r.rk, r.cks = kdfRoot(sk, out)
	return r, nil
}

func newRatchetResponder(sk []byte, signedPre *ecdh.PrivateKey, ad []byte) *ratchet {
	return &ratchet{dhs: signedPre, rk: sk, ad: ad, skipped: map[string][]byte{}}
}

func (r *ratchet) dh(remotePub []byte) ([]byte, error) {
	pub, err := ecdh.X25519().NewPublicKey(remotePub)
	if err != nil {
		return nil, err
	}
	return r.dhs.ECDH(pub)
}

// encrypt advances the sending chain and seals plaintext.
func (r *ratchet) encrypt(plaintext []byte) ([]byte, error) {
	mk := chainStep(&r.cks)
	h := ratchetHeader{DH: r.dhs.PublicKey().Bytes(), PN: r.pn, N: r.ns}
	r.ns++
	ad := append(append([]byte{}, r.ad...), h.bytes()...)
	ct, err := aeadSeal(mk, plaintext, ad)
	if err != nil {
		return nil, err
	}
	return encodeMessage(h, ct), nil
}

// decrypt opens a wire message, performing DH ratchet steps as needed.
func (r *ratchet) decrypt(wire []byte) ([]byte, error) {
	h, ct, err := decodeMessage(wire)
	if err != nil {
		return nil, err
	}
	if pt, ok, err := r.trySkipped(h, ct); err != nil || ok {
		return pt, err
	}
	if r.dhr == nil || !equal(h.DH, r.dhr) {
		if err := r.skip(h.PN); err != nil {
			return nil, err
		}
		if err := r.dhRatchet(h.DH); err != nil {
			return nil, err
		}
	}
	if err := r.skip(h.N); err != nil {
		return nil, err
	}
	mk := chainStep(&r.ckr)
	r.nr++
	ad := append(append([]byte{}, r.ad...), h.bytes()...)
	return aeadOpen(mk, ct, ad)
}

func (r *ratchet) trySkipped(h ratchetHeader, ct []byte) ([]byte, bool, error) {
	key := hex.EncodeToString(h.DH) + ":" + itoa(h.N)
	mk, ok := r.skipped[key]
	if !ok {
		return nil, false, nil
	}
	ad := append(append([]byte{}, r.ad...), h.bytes()...)
	pt, err := aeadOpen(mk, ct, ad)
	if err != nil {
		return nil, false, err
	}
	delete(r.skipped, key)
	return pt, true, nil
}

func (r *ratchet) skip(until uint32) error {
	if r.ckr == nil {
		return nil
	}
	if until > r.nr+maxSkip {
		return errors.New("crypto: too many skipped messages")
	}
	for r.nr < until {
		mk := chainStep(&r.ckr)
		r.skipped[hex.EncodeToString(r.dhr)+":"+itoa(r.nr)] = mk
		r.nr++
	}
	return nil
}

func (r *ratchet) dhRatchet(remotePub []byte) error {
	r.pn = r.ns
	r.ns, r.nr = 0, 0
	r.dhr = remotePub

	out, err := r.dh(remotePub)
	if err != nil {
		return err
	}
	r.rk, r.ckr = kdfRoot(r.rk, out)

	dhs, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	r.dhs = dhs
	out, err = r.dh(remotePub)
	if err != nil {
		return err
	}
	r.rk, r.cks = kdfRoot(r.rk, out)
	return nil
}

// --- key derivation steps ---

// kdfRoot derives a new root key and chain key from the old root key and a DH
// output.
func kdfRoot(rk, dhOut []byte) (newRK, ck []byte) {
	out := hkdf(rk, dhOut, []byte("stunner-root"), 64)
	return out[:32], out[32:]
}

// chainStep advances a chain key in place and returns the next message key.
func chainStep(ck *[]byte) []byte {
	mk := hmacSHA256(*ck, []byte{0x01})
	*ck = hmacSHA256(*ck, []byte{0x02})
	return mk
}

// --- AEAD over a message key ---

func messageKeys(mk []byte) (encKey, nonce []byte) {
	km := hkdf(nil, mk, []byte("stunner-msg"), 32+12)
	return km[:32], km[32:]
}

func aeadSeal(mk, plaintext, ad []byte) ([]byte, error) {
	gcm, nonce, err := msgGCM(mk)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, ad), nil
}

func aeadOpen(mk, ct, ad []byte) ([]byte, error) {
	gcm, nonce, err := msgGCM(mk)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, ad)
}

func msgGCM(mk []byte) (cipher.AEAD, []byte, error) {
	encKey, nonce := messageKeys(mk)
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	return gcm, nonce, nil
}

// --- helpers ---

func (h ratchetHeader) bytes() []byte {
	b := make([]byte, 0, len(h.DH)+8)
	b = append(b, h.DH...)
	var n [8]byte
	binary.BigEndian.PutUint32(n[0:4], h.PN)
	binary.BigEndian.PutUint32(n[4:8], h.N)
	return append(b, n[:]...)
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func itoa(n uint32) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
