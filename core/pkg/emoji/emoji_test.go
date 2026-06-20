package emoji

import "testing"

func TestLookup(t *testing.T) {
	if e, ok := Lookup("fire"); !ok || e != "🔥" {
		t.Errorf("Lookup(fire) = %q, %v", e, ok)
	}
	if _, ok := Lookup("does-not-exist"); ok {
		t.Error("expected miss for unknown shortcode")
	}
}

func TestReplace(t *testing.T) {
	cases := map[string]string{
		"hello :wave:":         "hello 👋",
		"ship it :rocket:!":    "ship it 🚀!",
		"no emoji here":        "no emoji here",
		":fire: and :unknown:": "🔥 and :unknown:",
		"ratio 3:4 not emoji":  "ratio 3:4 not emoji",
	}
	for in, want := range cases {
		if got := Replace(in); got != want {
			t.Errorf("Replace(%q) = %q, want %q", in, got, want)
		}
	}
}
