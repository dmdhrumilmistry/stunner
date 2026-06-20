package safetynumber

import (
	"strings"
	"testing"

	"github.com/dmdhrumilmistry/stunner/core/pkg/identity"
)

func TestSymmetric(t *testing.T) {
	a, _ := identity.Generate()
	b, _ := identity.Generate()
	if Compute(a.SigningPub, b.SigningPub) != Compute(b.SigningPub, a.SigningPub) {
		t.Error("safety number must be identical regardless of order")
	}
}

func TestFormatIs60Digits(t *testing.T) {
	a, _ := identity.Generate()
	b, _ := identity.Generate()
	sn := Compute(a.SigningPub, b.SigningPub)
	digits := strings.ReplaceAll(sn, " ", "")
	if len(digits) != 60 {
		t.Errorf("got %d digits, want 60 (%q)", len(digits), sn)
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			t.Fatalf("non-digit in safety number: %q", sn)
		}
	}
}

func TestDistinctPairsDiffer(t *testing.T) {
	a, _ := identity.Generate()
	b, _ := identity.Generate()
	c, _ := identity.Generate()
	if Compute(a.SigningPub, b.SigningPub) == Compute(a.SigningPub, c.SigningPub) {
		t.Error("different peers should yield different safety numbers")
	}
}
