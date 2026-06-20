package emoji

import (
	"encoding/json"
	"strings"
)

// Pack is a named collection of animated emoji (a manifest). Packs are bundled
// with the app or exchanged as files and registered locally; messages reference
// entries by id, and the Flutter renderer resolves AssetRef (Lottie/APNG/WebP).
// See docs/PROTOCOL.md §6.
type Pack struct {
	Name    string     `json:"name"`
	Emoji   []Animated `json:"emoji"`
	byID    map[string]Animated
	byShort map[string]Animated
}

// LoadPack parses a manifest (JSON) and indexes it for lookup.
func LoadPack(manifest []byte) (*Pack, error) {
	var p Pack
	if err := json.Unmarshal(manifest, &p); err != nil {
		return nil, err
	}
	p.index()
	return &p, nil
}

// NewPack builds a Pack from entries.
func NewPack(name string, emoji ...Animated) *Pack {
	p := &Pack{Name: name, Emoji: emoji}
	p.index()
	return p
}

func (p *Pack) index() {
	p.byID = make(map[string]Animated, len(p.Emoji))
	p.byShort = make(map[string]Animated, len(p.Emoji))
	for _, e := range p.Emoji {
		p.byID[e.ID] = e
		p.byShort[strings.Trim(e.Shortcode, ":")] = e
	}
}

// ByID returns the animated emoji with the given id.
func (p *Pack) ByID(id string) (Animated, bool) {
	e, ok := p.byID[id]
	return e, ok
}

// ByShortcode returns the animated emoji for a :shortcode: (colons optional).
func (p *Pack) ByShortcode(shortcode string) (Animated, bool) {
	e, ok := p.byShort[strings.Trim(shortcode, ":")]
	return e, ok
}
