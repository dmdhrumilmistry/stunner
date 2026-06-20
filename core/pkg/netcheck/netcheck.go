// Package netcheck probes NAT-traversal reachability: it asks whether the
// configured STUN servers can be reached and a public (server-reflexive)
// address discovered. This powers the app's "Test STUN connection" diagnostic.
//
// It uses pion/webrtc to run ICE gathering exactly as a real connection would,
// then inspects the gathered candidates: a server-reflexive ("srflx") candidate
// means STUN succeeded (the peer learned its public address via the STUN
// server); only host candidates means STUN did not respond (blocked/unreachable).
package netcheck

import (
	"fmt"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"

	"github.com/dmdhrumilmistry/stunner/core/pkg/settings"
)

// Result reports the outcome of a STUN reachability probe.
type Result struct {
	// OK is true when a server-reflexive candidate was discovered (STUN worked).
	OK bool
	// ReflexiveAddr is the public address STUN reported, when OK.
	ReflexiveAddr string
	// CandidateTypes lists the ICE candidate types that were gathered
	// (e.g. "host", "srflx", "relay").
	CandidateTypes []string
}

// Detail renders a short human-readable summary for the UI.
func (r Result) Detail() string {
	if r.OK {
		return "STUN reachable — public address " + r.ReflexiveAddr
	}
	if len(r.CandidateTypes) == 0 {
		return "No ICE candidates gathered; check your network."
	}
	return "STUN did not respond (only local candidates). A direct P2P path may " +
		"require a TURN server on restrictive networks."
}

// STUN runs an ICE-gathering probe against the given ICE servers and reports
// whether a server-reflexive candidate was obtained within timeout.
func STUN(servers []settings.ICEServer, timeout time.Duration) (Result, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{ICEServers: toICEServers(servers)})
	if err != nil {
		return Result{}, err
	}
	defer pc.Close()

	var mu sync.Mutex
	types := map[string]bool{}
	var res Result
	done := make(chan struct{})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil { // nil candidate signals end of gathering
			select {
			case <-done:
			default:
				close(done)
			}
			return
		}
		mu.Lock()
		defer mu.Unlock()
		typ := c.Typ.String()
		types[typ] = true
		if c.Typ == webrtc.ICECandidateTypeSrflx && res.ReflexiveAddr == "" {
			res.OK = true
			res.ReflexiveAddr = fmt.Sprintf("%s:%d", c.Address, c.Port)
		}
	})

	// A data channel + offer kick off ICE gathering.
	if _, err := pc.CreateDataChannel("probe", nil); err != nil {
		return Result{}, err
	}
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return Result{}, err
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return Result{}, err
	}

	select {
	case <-done:
	case <-time.After(timeout):
	}

	mu.Lock()
	defer mu.Unlock()
	for t := range types {
		res.CandidateTypes = append(res.CandidateTypes, t)
	}
	return res, nil
}

func toICEServers(in []settings.ICEServer) []webrtc.ICEServer {
	out := make([]webrtc.ICEServer, 0, len(in))
	for _, s := range in {
		srv := webrtc.ICEServer{URLs: s.URLs}
		if s.Username != "" {
			srv.Username = s.Username
			srv.Credential = s.Credential
		}
		out = append(out, srv)
	}
	return out
}
