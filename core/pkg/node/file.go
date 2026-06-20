package node

import (
	"encoding/json"
	"errors"

	"github.com/dmdhrumilmistry/stunner/core/pkg/filetransfer"
	"github.com/dmdhrumilmistry/stunner/core/pkg/messaging"
)

// ReceiveFile reads a FILE_OFFER envelope followed by its FILE_CHUNK envelopes
// and returns the reassembled, integrity-verified file along with the offer
// metadata. It assumes the sender used SendFile.
func (l *Link) ReceiveFile() (filetransfer.Offer, []byte, error) {
	env, err := l.Receive()
	if err != nil {
		return filetransfer.Offer{}, nil, err
	}
	if env.Type != messaging.TypeFileOffer {
		return filetransfer.Offer{}, nil, errors.New("node: expected file offer")
	}
	var offer filetransfer.Offer
	if err := json.Unmarshal(env.Body, &offer); err != nil {
		return filetransfer.Offer{}, nil, err
	}

	num := 0
	if offer.ChunkSize > 0 {
		num = int((offer.Size + uint64(offer.ChunkSize) - 1) / uint64(offer.ChunkSize))
	}
	chunks := make([]filetransfer.Chunk, 0, num)
	for i := 0; i < num; i++ {
		cenv, err := l.Receive()
		if err != nil {
			return filetransfer.Offer{}, nil, err
		}
		var c filetransfer.Chunk
		if err := json.Unmarshal(cenv.Body, &c); err != nil {
			return filetransfer.Offer{}, nil, err
		}
		chunks = append(chunks, c)
	}
	data, err := filetransfer.Reassemble(offer, chunks)
	return offer, data, err
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
