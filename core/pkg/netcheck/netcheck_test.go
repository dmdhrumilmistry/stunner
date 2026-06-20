package netcheck

import (
	"testing"
	"time"
)

// TestSTUNNoServers is hermetic: with no STUN servers configured, the probe must
// complete without error and report failure (only host candidates, no srflx).
func TestSTUNNoServers(t *testing.T) {
	res, err := STUN(nil, 3*time.Second)
	if err != nil {
		t.Fatalf("STUN: %v", err)
	}
	if res.OK {
		t.Errorf("expected OK=false with no STUN servers, got srflx %q", res.ReflexiveAddr)
	}
	if res.Detail() == "" {
		t.Error("expected a non-empty detail string")
	}
}

// TestResultDetail checks the human-readable summaries.
func TestResultDetail(t *testing.T) {
	ok := Result{OK: true, ReflexiveAddr: "203.0.113.5:54321", CandidateTypes: []string{"host", "srflx"}}
	if got := ok.Detail(); got == "" || got[:4] != "STUN" {
		t.Errorf("ok detail = %q", got)
	}
	none := Result{}
	if none.Detail() == "" {
		t.Error("empty result should still have a detail")
	}
}
