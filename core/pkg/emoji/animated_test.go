package emoji

import "testing"

func TestPackLookup(t *testing.T) {
	p := NewPack("default",
		Animated{ID: "pp", Shortcode: ":party_parrot:", Format: FormatLottie, AssetRef: "pp.json"},
		Animated{ID: "blob", Shortcode: ":blob_wave:", Format: FormatAPNG, AssetRef: "blob.png"},
	)
	if e, ok := p.ByID("pp"); !ok || e.Format != FormatLottie {
		t.Errorf("ByID(pp) = %+v %v", e, ok)
	}
	if e, ok := p.ByShortcode("blob_wave"); !ok || e.AssetRef != "blob.png" {
		t.Errorf("ByShortcode(blob_wave) = %+v %v", e, ok)
	}
	if _, ok := p.ByShortcode(":missing:"); ok {
		t.Error("expected miss")
	}
}

func TestLoadPack(t *testing.T) {
	manifest := []byte(`{"name":"x","emoji":[{"id":"a","shortcode":":a:","format":"WEBP","assetRef":"a.webp"}]}`)
	p, err := LoadPack(manifest)
	if err != nil {
		t.Fatalf("LoadPack: %v", err)
	}
	if e, ok := p.ByID("a"); !ok || e.Format != FormatWebP {
		t.Errorf("loaded pack lookup failed: %+v %v", e, ok)
	}
}
