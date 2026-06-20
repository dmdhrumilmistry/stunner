# Stunner Roadmap

A phased plan from skeleton to a working messenger. Each phase is independently
buildable and reviewable.

## Status

Phases 1–8 are implemented as a **stdlib-only reference core** whose entire
pipeline runs and is unit-tested in-process. What remains is swapping the
in-process/reference backends for production ones behind the same interfaces:

| Concern | Implemented now (stdlib) | Production backend (pending) |
|---|---|---|
| E2E crypto | from-scratch X3DH + Double Ratchet (`pkg/crypto`) | optionally `go.mau.fi/libsignal`; **needs audit** |
| Transport | in-process pipe (`transport.Pipe`) | `pion/webrtc` data channels over STUN/TURN |
| Signaling | in-memory registry (`signaling.Registry`) | libp2p Kademlia DHT |
| Storage | vault-sealed JSON file (`storage` file store) | SQLCipher |
| Offline relay | in-memory mailbox (`pkg/mailbox`) | networked self-hostable relay |
| Mobile FFI | gomobile-ready surface | `gomobile bind` artifacts + stateful runtime binding |

> ⚠️ **Crypto audit required.** The X3DH/Double Ratchet composition is a
> from-scratch implementation over vetted stdlib primitives (AES-GCM,
> HMAC-SHA256, SHA-512, X25519). It must receive an independent review before any
> production use. See `docs/THREAT_MODEL.md`.

Run the whole pipeline end-to-end: `cd core && go run ./cmd/stunnerd`.

## Phase 1 — Skeleton ✅

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

## Phase 2 — Identity & verification ✅

- Encrypted-at-rest identity (`pkg/account` + `pkg/vault`), reloads across runs.
- Safety numbers (`pkg/safetynumber`, Signal-style 60-digit, symmetric).
- Contact model with TOFU key-change detection and `stunner:contact` QR/URI
  exchange (`pkg/contact`).
- Flutter: My-identity screen (QR + safety-number verification).

## Phase 3 — Secure sessions (Signal) ✅

- X3DH handshake + Double Ratchet (`pkg/crypto`): forward secrecy, out-of-order
  delivery, skipped-key handling, AEAD-bound headers and identity binding.
- Prekey generation/bundles; in-memory session store.
- _Pending:_ optional swap to `go.mau.fi/libsignal`; **independent audit**.

## Phase 4 — Transport & signaling ✅ (reference)

- `Signaler` interface + in-memory `signaling.Registry` (discovery, SDP/ICE,
  presence); in-process `transport.Pipe` data channel.
- `pkg/node` ties account + sessions + transport into a working link; two nodes
  exchange an E2E message in-process (see `cmd/stunnerd`).
- _Pending:_ `pion/webrtc` over STUN/TURN + libp2p Kademlia DHT backends.

## Phase 5 — Storage & history ✅ (reference)

- `pkg/storage` encrypted file store (vault/AES-256-GCM): settings,
  conversations, messages, outbox, blobs; key from the app's secure store.
- _Pending:_ SQLCipher backend for indexed/incremental access.

## Phase 6 — File transfer ✅

- `pkg/filetransfer`: chunked, AEAD-sealed, out-of-order-tolerant transfers with
  SHA-256 integrity; wired through `node.Link.SendFile`/`ReceiveFile`.
- _Pending:_ resume-on-reconnect + data-channel backpressure tuning.

## Phase 7 — UX: emoji, animated emoji, settings ✅

- `pkg/emoji`: shortcode expansion + animated-emoji pack manifest.
- Flutter: emoji shortcode expansion in the composer; settings for STUN/TURN
  override, optional-relay toggle, app-lock; QR identity screen.
- _Pending:_ full emoji picker sheet + Lottie rendering wired to packs.

## Phase 8 — Optional offline relay ✅ (reference)

- `pkg/mailbox`: content-blind store-and-forward; `node.SendOffline` /
  `FetchOffline` perform offline X3DH + ratchet. Off by default.
- _Pending:_ networked, self-hostable relay deployment.

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
