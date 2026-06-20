# CLAUDE.md

Context for AI assistants (and humans) working in this repo.

## What Stunner is

An open-source, cross-platform, privacy-first **peer-to-peer messenger**:
Signal-protocol end-to-end encryption, NAT traversal via STUN/TURN + WebRTC,
file transfer, emoji/animated emoji. Targets Android, iOS, macOS, Windows
(+ Linux desktop). MIT licensed.

## Layout

```
core/   Go module (github.com/dmdhrumilmistry/stunner/core) — all networking & crypto
app/    Flutter app (Dart UI) — calls the Go core over dart:ffi
docs/   ARCHITECTURE.md, THREAT_MODEL.md, PROTOCOL.md, ROADMAP.md
Makefile  local dev + release helpers (run `make help`)
```

### Core packages (`core/pkg/`)
- `identity` — Ed25519 identity + X25519 agreement keys, fingerprints (stdlib).
- `crypto` — from-scratch X3DH + Double Ratchet (reference). **Unaudited.**
- `crypto/libsignal` — `go.mau.fi/libsignal` backend implementing `crypto.Session` (production crypto).
- `transport` — `Transport`/`Conn` interfaces; `pion/webrtc` impl (`New`) + in-process `Pipe` (tests).
- `signaling` — `Signaler` interface; libp2p Kademlia DHT (`NewDHT`) + in-memory `Registry` (tests).
- `storage` — encrypted-at-rest `Store`; vault-sealed file store (SQLCipher is a future swap).
- `vault` — AES-256-GCM seal/open + PBKDF2; `account` — persistent encrypted identity.
- `messaging`, `filetransfer`, `mailbox`, `node` — envelopes/frames, chunked files, offline relay, runtime that ties it together.
- `safetynumber`, `contact`, `emoji`, `settings` — verification, contacts/QR, emoji, STUN/TURN config.
- `core` — version + FFI-facing helpers; `mobile` (gomobile) and `ffi` (cgo c-shared) are the FFI surfaces; `cmd/stunnerd` is the headless end-to-end demo.

## Architecture notes
- Two backends per networked concern: an in-process **reference** backend (carries logic, used in tests/`stunnerd`) and a **production** backend behind the same interface. See `docs/ROADMAP.md`.
- The app loads the core via `dart:ffi` (`app/lib/src/ffi/stunner_ffi.dart`): Android `dlopen`s `libstunner.so` from jniLibs; desktop searches bundle-relative paths.

## Conventions / gotchas
- **Go 1.25+** (libp2p requires it). Module has `tool` directives for gomobile/gobind.
- Keep `gofmt` clean and `go vet`/`go test ./...` green before pushing (`make check`).
- **Crypto:** never hand-roll primitives; prefer the `crypto/libsignal` backend for production. The from-scratch ratchet needs an audit.
- **gomobile (`core/mobile`)**: exported funcs may return at most **one value + error** — wrap multiple returns in a struct (see `ContactInfo`).
- **gomobile bind** needs `-androidapi 21` (NDK r26 dropped API 16).
- **Android APK** build needs `compileSdk 36` (plugins require it); the release workflow patches the generated project.
- Tag pushes may be blocked in some sandboxes; cut releases from a real checkout.

## Common commands (see `make help`)
```bash
make check        # build + vet + fmt-check + test
make demo         # run the full pipeline in-process (cmd/stunnerd)
make test-race    # race detector on crypto/node
make lib          # build the desktop c-shared library
make app-run      # build core lib + flutter run
make release-tag TAG=v0.3.1   # (re)create & push a release tag at origin/main
```

## CI / releases
- `.github/workflows/ci.yml` — Go build/vet/test + Flutter analyze/test on push/PR.
- `.github/workflows/release.yml` — on `v*` tags: builds the **app** (Android APK + Linux/Windows/macOS desktop), `stunnerd` CLIs, `libstunner` libraries, and the `stunnercore.aar`, attaching all to a GitHub Release. App/aar jobs are best-effort (`continue-on-error`).

## Workflow
- Develop on a `claude/*` branch; open a PR to `main`; keep PRs focused.
- The two-device **live** messaging path in the GUI still needs the production transport/signaling wired into the app runtime over FFI (next integration step). Today the GUI exercises the core locally; the full path is covered by Go tests and `stunnerd`.
