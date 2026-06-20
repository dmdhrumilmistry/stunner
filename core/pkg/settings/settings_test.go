package settings

import "testing"

func TestDefaultHasStunServers(t *testing.T) {
	s := Default()
	if len(s.EffectiveICEServers()) == 0 {
		t.Fatal("expected default ICE servers")
	}
	if s.RelayEnabled {
		t.Error("relay should be off by default")
	}
}

func TestEffectiveICEServersFallsBackToDefaults(t *testing.T) {
	s := Settings{} // no ICE servers configured
	if len(s.EffectiveICEServers()) == 0 {
		t.Error("expected fallback to default ICE servers")
	}
}

func TestEffectiveICEServersHonorsOverride(t *testing.T) {
	custom := ICEServer{URLs: []string{"turn:turn.example.org:3478"}, Username: "u", Credential: "p"}
	s := Settings{ICEServers: []ICEServer{custom}}
	got := s.EffectiveICEServers()
	if len(got) != 1 || got[0].URLs[0] != "turn:turn.example.org:3478" {
		t.Errorf("override not honored, got %+v", got)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	s := Default()
	s.RelayEnabled = true
	s.RelayAddress = "relay.example.org:9000"
	b, err := s.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	back, err := FromJSON(b)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}
	if !back.RelayEnabled || back.RelayAddress != "relay.example.org:9000" {
		t.Errorf("round trip mismatch: %+v", back)
	}
}
