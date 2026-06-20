# Stunner

> Open-source, cross-platform, privacy-first peer-to-peer messenger built on
> STUN/TURN + WebRTC, with end-to-end encryption (Signal protocol), file
> transfer, and emoji / animated-emoji support.

Stunner runs on **Android, iOS, macOS, and Windows**. The networking and
cryptography live in a single **Go core**; the UI is a **Flutter** app that
calls the core over FFI. Messages travel **directly between devices** over
encrypted WebRTC data channels — there is no central message server.

> ⚠️ **Status: working reference core.** The full pipeline — X3DH + Double
> Ratchet E2E encryption, messaging, chunked file transfer, encrypted-at-rest
> storage, safety-number verification, and an optional offline mailbox — is
> implemented and unit-tested in Go (standard library only) and runs end-to-end
> in-process (`cd core && go run ./cmd/stunnerd`). What remains is swapping the
> in-process/reference backends for production ones (pion/webrtc, libp2p,
> SQLCipher) behind the same interfaces. See [`docs/ROADMAP.md`](docs/ROADMAP.md).
>
> 🔒 **Crypto not yet audited.** The X3DH/Double Ratchet composition is built
> from vetted stdlib primitives but must receive an independent review before
> production use — see [`docs/THREAT_MODEL.md`](docs/THREAT_MODEL.md).

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

## Contributing

Contributions welcome. Please read [`SECURITY.md`](SECURITY.md) before reporting
vulnerabilities, and never hand-roll cryptographic primitives — use the vetted
libraries referenced in the docs.

## License

[MIT](LICENSE). Stunner is and will remain free and open source.
