# Stunner Roadmap

A phased plan from the current skeleton to a working messenger. Each phase is
designed to be independently buildable and reviewable. Phases introduce external
dependencies gradually — the skeleton itself uses only the Go standard library.

## Phase 1 — Skeleton ✅ (this repo)

- Repository layout, docs (architecture, threat model, protocol, roadmap).
- Go core module with stub packages and stable interfaces.
- Working FFI proof: `Version()` / `Ping()` exported via `core/mobile` and
  `core/ffi`; callable from the headless `cmd/stunnerd` and from Dart.
- Real Ed25519/X25519 identity key generation + fingerprints (stdlib only).
- Flutter app shell: chats list, conversation, and settings screens; Dart FFI
  binding stub.
- CI: `go build/vet/test` + `flutter analyze`.

**Done when:** `go build ./... && go vet ./... && go test ./...` is green and
`go run ./cmd/stunnerd` prints the version and a generated fingerprint.

## Phase 2 — Identity & verification

- Persist identity in `pkg/storage` (encrypted).
- Safety numbers + QR encode/decode.
- Contact model and key-change detection.

## Phase 3 — Secure sessions (Signal)

- Integrate `go.mau.fi/libsignal`: X3DH handshake, Double Ratchet.
- Prekey generation/management.
- Encrypt/decrypt application envelopes over an in-memory loopback transport
  (no network yet) to validate the crypto pipeline end-to-end.

## Phase 4 — Transport & signaling

- `pkg/transport`: `pion/webrtc` data channels with configurable ICE servers.
- `pkg/signaling`: `Signaler` interface + libp2p Kademlia DHT implementation.
- Goal: two `stunnerd` instances discover each other and exchange an encrypted
  message over a real WebRTC data channel using STUN (TURN when needed).

## Phase 5 — Storage & history

- `pkg/storage`: SQLCipher schema for conversations, messages, contacts, keys.
- DB key sourced from OS secure store (per-platform shims via the app).
- Outbox + retry; delivery/read receipts.

## Phase 6 — File transfer

- `pkg/filetransfer`: chunked, AEAD-sealed, resumable transfers with integrity
  verification, wired to data-channel backpressure.

## Phase 7 — UX: emoji, animated emoji, settings

- Emoji picker (Unicode) and animated emoji rendering (Lottie/APNG) in Flutter.
- Settings UI: STUN/TURN override, optional-relay toggle, app-lock
  (biometric/PIN), disappearing-message timers.

## Phase 8 — Optional offline relay

- Self-hostable, content-blind encrypted mailbox for offline delivery.
- Off by default; documented metadata tradeoffs.

## Cross-cutting / later

- Group messaging (sender-keys or MLS).
- Reproducible builds + release signing.
- External security audit before 1.0.

---

## Build notes for native libraries (phases 4+)

These commands are documented here so future contributors can wire the Go core
into the Flutter app.

### Mobile (gomobile)

```bash
# one-time
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init

# Android (.aar) and iOS (.xcframework) from the mobile binding package
cd core
gomobile bind -target=android -o ../app/android/stunnercore.aar ./mobile
gomobile bind -target=ios     -o ../app/ios/Stunnercore.xcframework ./mobile
```

### Desktop (c-shared)

```bash
cd core
# produces libstunner.{so,dylib,dll} + libstunner.h
go build -buildmode=c-shared -o ../app/native/libstunner.so ./ffi
```

The Flutter app loads the appropriate library per platform and calls it via
`dart:ffi` (see `app/lib/src/ffi/`).
