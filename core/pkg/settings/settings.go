// Package settings holds user-configurable application configuration.
//
// The most important responsibility here is the STUN/TURN (ICE) server list:
// Stunner ships sensible public defaults but lets the user override them in the
// Settings UI (e.g. to point at a self-hosted coturn). The transport package
// feeds ICEServers straight into the WebRTC configuration.
//
// See ../../docs/ARCHITECTURE.md (NAT traversal) and ../../docs/THREAT_MODEL.md.
package settings

import "encoding/json"

// ICEServer describes a single STUN or TURN server, mirroring the fields WebRTC
// expects. Credentials are only used for TURN.
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// Settings is the full set of user preferences. It is serialized to the
// encrypted local store (pkg/storage).
type Settings struct {
	// ICEServers is the STUN/TURN list used for NAT traversal. If empty, the
	// transport falls back to DefaultICEServers().
	ICEServers []ICEServer `json:"iceServers"`

	// RelayEnabled turns on the optional, self-hostable offline mailbox/relay.
	// Off by default to preserve the pure-P2P, low-metadata posture.
	RelayEnabled bool   `json:"relayEnabled"`
	RelayAddress string `json:"relayAddress,omitempty"`

	// AppLock controls the biometric/PIN gate before the local store is
	// decrypted. ("none" | "pin" | "biometric")
	AppLock string `json:"appLock"`

	// DisappearingDefaultSeconds is the default disappearing-message timer for
	// new conversations (0 = disabled).
	DisappearingDefaultSeconds int `json:"disappearingDefaultSeconds"`
}

// DefaultICEServers returns the built-in public STUN/TURN defaults.
//
// Public TURN is best-effort; operators should self-host coturn for reliability.
// These defaults are always overridable in settings.
func DefaultICEServers() []ICEServer {
	return []ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
		{URLs: []string{"stun:stun1.l.google.com:19302"}},
		// TODO(roadmap phase 4+): document/add a reliable default TURN option,
		// e.g. a community openrelay endpoint, while strongly recommending a
		// self-hosted coturn for sensitive use.
	}
}

// Default returns a fresh Settings populated with safe defaults.
func Default() Settings {
	return Settings{
		ICEServers:                 DefaultICEServers(),
		RelayEnabled:               false,
		AppLock:                    "none",
		DisappearingDefaultSeconds: 0,
	}
}

// EffectiveICEServers returns the configured servers, or the defaults when none
// are set.
func (s Settings) EffectiveICEServers() []ICEServer {
	if len(s.ICEServers) == 0 {
		return DefaultICEServers()
	}
	return s.ICEServers
}

// MarshalJSON / UnmarshalJSON are provided via the struct tags above; helpers
// below keep the FFI layer simple.

// JSON serializes the settings for transport across the FFI boundary.
func (s Settings) JSON() ([]byte, error) { return json.Marshal(s) }

// FromJSON parses settings received across the FFI boundary.
func FromJSON(b []byte) (Settings, error) {
	var s Settings
	err := json.Unmarshal(b, &s)
	return s, err
}
