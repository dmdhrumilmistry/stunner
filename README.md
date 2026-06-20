# Stunner

> Open-source, cross-platform, privacy-first peer-to-peer messenger built on
> STUN/TURN + WebRTC, with end-to-end encryption (Signal protocol), file
> transfer, and emoji / animated-emoji support.

Stunner runs on **Android, iOS, macOS, and Windows**. The networking and
cryptography live in a single **Go core**; the UI is a **Flutter** app that
calls the core over FFI. Messages travel **directly between devices** over
encrypted WebRTC data channels — there is no central message server.

> ⚠️ **Status: working core with production backends.** The full pipeline —
> Signal-protocol E2E encryption, messaging, chunked file transfer,
> encrypted-at-rest storage, safety-number verification, and an optional offline
> mailbox — is implemented and unit-tested, and runs end-to-end in-process
> (`cd core && go run ./cmd/stunnerd`). Each networked concern has both an
> in-process reference backend and a production backend wired in behind the same
> interface: **`pion/webrtc`** transport over STUN/TURN, a **libp2p** Kademlia
> DHT signaler, and a **`go.mau.fi/libsignal`** crypto backend. See
> [`docs/ROADMAP.md`](docs/ROADMAP.md).
>
> 🔒 **Crypto:** use the `pkg/crypto/libsignal` (maintained Signal library)
> backend for production. The from-scratch X3DH/Double Ratchet in `pkg/crypto` is
> built on vetted stdlib primitives but is **not independently audited** — see
> [`docs/THREAT_MODEL.md`](docs/THREAT_MODEL.md).
>
> 📦 **Releases:** push a `v*` tag to build cross-platform `stunnerd` binaries,
> desktop `libstunner` libraries, and the Android `.aar`, attached to a GitHub
> Release (`.github/workflows/release.yml`).

## Why "Stunner"

The name nods to **STUN/TURN**, the NAT-traversal protocols that let two phones
behind home routers talk directly. Stunner uses public STUN/TURN servers by
default and lets you **override them in settings** (or point at your own
`coturn`).

## Design highlights

| Area | Choice |
|---|---|
| UI | Flutter (Dart) on all 4 platforms |
| Core | Go, compiled to a native lib (`gomobile` on mobile, `c-shared`/cgo on desktop) |
| Transport | WebRTC data channels via [`pion/webrtc`](https://github.com/pion/webrtc) |
| NAT traversal | STUN/TURN — public defaults, overridable in settings |
| Network model | **Pure P2P** (no central message server) |
| Discovery / signaling | Decentralized DHT rendezvous (libp2p Kademlia) |
| Encryption | **Signal protocol** (X3DH + Double Ratchet) |
| Local storage | SQLCipher (encrypted at rest), DB key in OS keystore |
| Emoji | Unicode emoji + animated emoji (Lottie / APNG) |

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full design and
[`docs/THREAT_MODEL.md`](docs/THREAT_MODEL.md) for the security model.

### The pure-P2P tradeoff (read this)

Because there is no store-and-forward server, **both peers must be online** to
exchange messages directly. Stunner queues and retries outgoing messages, and
offers an **optional, self-hostable relay/mailbox** (off by default) for offline
delivery. This is a deliberate privacy/availability tradeoff — see
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md#offline-delivery).

## Repository layout

```
core/   Go core: identity, crypto, transport, signaling, messaging, files, storage
app/    Flutter app (UI) — calls the Go core over FFI
docs/   Architecture, threat model, protocol, roadmap
```

## Building

### Go core

```bash
cd core
go build ./...
go vet ./...
go test ./...
go run ./cmd/stunnerd   # prints version + a generated identity fingerprint
```

### Flutter app

Requires the [Flutter SDK](https://docs.flutter.dev/get-started/install).

```bash
cd app
flutter pub get
flutter analyze
flutter run            # pick a connected device / desktop target
```

> The Flutter app expects the compiled Go core as a native library. Build
> instructions per platform (`gomobile bind`, `go build -buildmode=c-shared`)
> are tracked in [`docs/ROADMAP.md`](docs/ROADMAP.md).

## Testing locally

Requires **Go 1.25+** (a recent toolchain is fetched automatically if needed).
No services to stand up — the whole stack runs in-process.

### Run the end-to-end demo

`stunnerd` exercises the full pipeline between two in-process accounts: it
generates encrypted identities, runs the X3DH + Double Ratchet handshake, sends
an end-to-end-encrypted message and a file, prints the safety number, and
delivers an offline message via the mailbox.

```bash
cd core
go run ./cmd/stunnerd
```

Expected output (fingerprints/safety number vary per run):

```
Stunner core 0.2.0

identities:
  alice 4W7FN ZLHYK ...
  bob   K5F7A QTQMG ...

safety number (compare on both devices):
  03592 30276 99571 54331 22262 49477 83291 67067 56552 55226 60948 57461

alice -> bob (E2E): "hello bob 🎉🔒"
file transfer: "secret.bin" (40960 bytes) integrity=true
offline mailbox -> bob: "sent while you were offline 👋"

all pipeline stages OK
```

### Run the test suite

```bash
cd core
go test ./...                                   # all packages
go test -race ./pkg/crypto/ ./pkg/node/         # race detector on the hot paths
go vet ./... && gofmt -l .                       # vet + format check (no output = clean)
```

What the tests cover, by package:

- `pkg/crypto` & `pkg/crypto/libsignal` — handshake + ratchet round-trips,
  bidirectional traffic, out-of-order delivery, tamper rejection.
- `pkg/transport` — a real `pion/webrtc` data-channel round-trip over loopback.
- `pkg/signaling` — libp2p SDP exchange between two hosts.
- `pkg/node` — two nodes exchange an encrypted message and a file; offline
  delivery via the mailbox.
- `pkg/account`, `pkg/storage`, `pkg/vault` — encrypted identity/store reload
  with correct vs. wrong keys.
- `pkg/safetynumber`, `pkg/contact`, `pkg/filetransfer`, `pkg/emoji` — units.

The libp2p **DHT discovery** test is opt-in (its propagation timing is
environment-sensitive):

```bash
STUNNER_DHT_TEST=1 go test ./pkg/signaling/ -run TestLibp2pDHTDiscovery
```

### Smoke-test the desktop FFI library

```bash
cd core
go build -buildmode=c-shared -o /tmp/libstunner.so ./ffi   # .dylib on macOS, .dll on Windows
```

### Flutter app

```bash
cd app
flutter pub get
flutter analyze
flutter test
```

## Install & use the app

Each tagged release (see [Releases](https://github.com/dmdhrumilmistry/stunner/releases))
attaches the **Stunner app** with the Go core bundled in, alongside the
`stunnerd` CLI and libraries:

- `stunner_<version>_android.apk` — Android app
- `stunner_<version>_linux.tar.gz` — Linux desktop app
- `stunner_<version>_windows.zip` — Windows desktop app
- `stunner_<version>_macos.tar.gz` — macOS desktop app

### Android

1. Download `stunner_<version>_android.apk` from the release.
2. On your phone, enable **Install unknown apps** for your browser/file manager.
3. Open the APK to install, then launch **Stunner**.

> The APK is signed with a debug key (for sideloading/testing, not the Play
> Store). Open **Settings → My identity & safety number** to see your QR code and
> verify a contact.

### Desktop

- **Linux:** `tar xzf stunner_<version>_linux.tar.gz && ./stunner` (the bundled
  `lib/libstunner.so` is loaded automatically).
- **Windows:** unzip and run `stunner.exe` (`stunner.dll` sits beside it).
- **macOS:** extract and open `stunner.app`. It is unsigned, so first launch
  needs **right-click → Open** (or *System Settings → Privacy & Security →
  Open Anyway*).

### Run from source

No release needed — build the core and run the app directly:

```bash
# 1) build the Go core as a native library next to the app
cd core
go build -buildmode=c-shared -o ../app/libstunner.so ./ffi      # .dylib / stunner.dll on macOS / Windows

# 2) run the Flutter app (desktop or a connected device)
cd ../app
flutter pub get
flutter run
```

The app loads the core via `dart:ffi`; if the library isn't found it still runs
in a degraded mode (UI works, core-backed features show "core unavailable").

> Two devices talking to each other needs the production transport/signaling
> (`pion/webrtc` + libp2p DHT) wired into the app's runtime over FFI — that
> binding is the next integration step (see [`docs/ROADMAP.md`](docs/ROADMAP.md)).
> Today the app exercises the core locally (identity, QR, safety numbers); the
> full two-device message path is covered by the Go tests and `stunnerd`.

## Contributing

Contributions welcome. Please read [`SECURITY.md`](SECURITY.md) before reporting
vulnerabilities, and never hand-roll cryptographic primitives — use the vetted
libraries referenced in the docs.

## License

[MIT](LICENSE). Stunner is and will remain free and open source.
