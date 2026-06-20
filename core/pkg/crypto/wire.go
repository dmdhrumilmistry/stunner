package crypto

import "encoding/json"

// wireMessage is the serialized form of a ratchet message (header + ciphertext).
// JSON is used for clarity in this reference implementation; []byte fields are
// base64-encoded automatically.
type wireMessage struct {
	DH []byte `json:"dh"`
	PN uint32 `json:"pn"`
	N  uint32 `json:"n"`
	CT []byte `json:"ct"`
}

func encodeMessage(h ratchetHeader, ct []byte) []byte {
	b, _ := json.Marshal(wireMessage{DH: h.DH, PN: h.PN, N: h.N, CT: ct})
	return b
}

func decodeMessage(b []byte) (ratchetHeader, []byte, error) {
	var m wireMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return ratchetHeader{}, nil, err
	}
	return ratchetHeader{DH: m.DH, PN: m.PN, N: m.N}, m.CT, nil
}
