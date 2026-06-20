# Stunner Roadmap

A phased plan from skeleton to a working messenger. Each phase is independently
buildable and reviewable.

## Status

Phases 1–8 are implemented, with both a **stdlib reference backend** (carrying
the core logic, runnable/testable in-process) and a **production backend** wired
in behind the same interface for each networked concern:

| Concern | Reference backend (default, in-process) | Production backend (wired in) |
|---|---|---|
| E2E crypto | from-scratch X3DH + Double Ratchet (`pkg/crypto`) | `go.mau.fi/libsignal` (`pkg/crypto/libsignal`) ✅ |
| Transport | in-process pipe (`transport.Pipe`) | `pion/webrtc` data channels over STUN/TURN (`transport.New`) ✅ |
| Signaling | in-memory registry (`signaling.Registry`) | libp2p Kademlia DHT (`signaling.NewDHT`) ✅ |
| Storage | vault-sealed file store (`storage.Open`) | SQLCipher (future swap behind `storage.Store`) |
| Offline relay | in-memory mailbox (`pkg/mailbox`) | networked self-hostable relay (future) |
| Mobile FFI | gomobile-ready surface | `gomobile bind` artifacts (built in release CI) |

> ⚠️ **Crypto audit.** The from-scratch X3DH/Double Ratchet in `pkg/crypto` is
> built on vetted stdlib primitives but is **not independently audited** — use
> the `pkg/crypto/libsignal` backend (maintained Signal implementation) for
> production, or commission an audit. See `docs/THREAT_MODEL.md`.

Run the whole pipeline end-to-end: `cd core && go run ./cmd/stunnerd`.

## Releases

Pushing a tag matching `v*` triggers `.github/workflows/release.yml`, which
builds and attaches to a GitHub Release:

- the **Stunner app** with the Go core bundled in: Android `.apk` and desktop
  bundles for Linux/Windows/macOS (best-effort),
- `stunnerd` CLI binaries for linux/darwin/windows × amd64/arm64 (tar.gz/zip +
  SHA-256 checksums),
- desktop `libstunner` c-shared libraries (`.so`/`.dylib`/`.dll`) with headers,
- the Android `stunnercore.aar` (gomobile, best-effort).

See the [README](../README.md#install--use-the-app) for install/usage steps.

```bash
git tag v0.3.0 && git push origin v0.3.0
```

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

- From-scratch X3DH + Double Ratchet (`pkg/crypto`): forward secrecy,
  out-of-order delivery, skipped-key handling, AEAD-bound headers, identity
  binding.
- Production: `go.mau.fi/libsignal` backend (`pkg/crypto/libsignal`) implementing
  the same `crypto.Session` interface ✅ — recommended for production pending an
  audit of the from-scratch path.

## Phase 4 — Transport & signaling ✅

- `Signaler`/`Transport` interfaces with in-process reference backends
  (`signaling.Registry`, `transport.Pipe`).
- Production: `pion/webrtc` data channels over the configurable STUN/TURN ICE
  servers (`transport.New`) ✅, and a libp2p Kademlia DHT signaler
  (`signaling.NewDHT`) exchanging SDP over authenticated streams ✅.
- `pkg/node` ties account + sessions + transport into a working link; two nodes
  exchange an E2E message (see `cmd/stunnerd`).

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

gomobile is pinned via a `tool` directive in `core/go.mod`, so it runs through
`go tool` (no `@latest` install) and `golang.org/x/mobile` stays in the module
graph (required by `gomobile bind`).

```bash
cd core
go tool gomobile init

# Android (.aar) and iOS (.xcframework) from the mobile binding package
go tool gomobile bind -target=android -o ../app/android/stunnercore.aar ./mobile
go tool gomobile bind -target=ios     -o ../app/ios/Stunnercore.xcframework ./mobile
```

### Desktop (c-shared)

```bash
cd core
# produces libstunner.{so,dylib,dll} + libstunner.h
go build -buildmode=c-shared -o ../app/native/libstunner.so ./ffi
```

The Flutter app loads the appropriate library per platform and calls it via
`dart:ffi` (see `app/lib/src/ffi/`).
