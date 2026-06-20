// Package emoji provides emoji support: shortcode lookup for static Unicode
// emoji and a manifest model for animated emoji.
//
// Static emoji are ordinary Unicode carried inline in message text and rendered
// natively by the platform; this package supplies the shortcode<->emoji mapping
// the picker and text entry use. Animated emoji (Lottie / APNG / WebP) are
// referenced by id from messages and resolved by the Flutter renderer.
// See docs/PROTOCOL.md §6.
package emoji

import "strings"

// Format enumerates animated-emoji asset formats.
type Format string

const (
	FormatLottie Format = "LOTTIE"
	FormatAPNG   Format = "APNG"
	FormatWebP   Format = "WEBP"
)

// Animated describes one animated emoji in a pack manifest.
type Animated struct {
	ID        string `json:"id"`        // stable id referenced from messages
	Shortcode string `json:"shortcode"` // e.g. ":party_parrot:"
	Format    Format `json:"format"`
	AssetRef  string `json:"assetRef"` // bundled asset id or content hash
}

// shortcodes is a small built-in catalog of static Unicode emoji. The full
// Unicode set is loaded from data on the client; this seed keeps the core
// self-contained and testable.
var shortcodes = map[string]string{
	"smile":      "😄",
	"grin":       "😁",
	"joy":        "😂",
	"heart":      "❤️",
	"thumbsup":   "👍",
	"thumbsdown": "👎",
	"fire":       "🔥",
	"tada":       "🎉",
	"rocket":     "🚀",
	"eyes":       "👀",
	"wave":       "👋",
	"lock":       "🔒",
}

// Lookup returns the Unicode emoji for a shortcode (without surrounding colons),
// e.g. Lookup("fire") -> "🔥". The bool reports whether it was found.
func Lookup(shortcode string) (string, bool) {
	e, ok := shortcodes[strings.ToLower(shortcode)]
	return e, ok
}

// Replace substitutes :shortcode: tokens in s with their Unicode emoji. Unknown
// shortcodes are left untouched.
func Replace(s string) string {
	var b strings.Builder
	for {
		start := strings.IndexByte(s, ':')
		if start < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:start])
		end := strings.IndexByte(s[start+1:], ':')
		if end < 0 {
			b.WriteString(s[start:])
			break
		}
		name := s[start+1 : start+1+end]
		if e, ok := Lookup(name); ok {
			b.WriteString(e)
		} else {
			b.WriteByte(':')
			b.WriteString(name)
			b.WriteByte(':')
		}
		s = s[start+1+end+1:]
	}
	return b.String()
}
