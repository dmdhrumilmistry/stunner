# Stunner Architecture

This document describes the architecture of Stunner: a cross-platform,
privacy-first, peer-to-peer messenger.

## Goals

- **Cross-platform:** Android, iOS, macOS, Windows from one codebase.
- **Private by default:** no central message server; end-to-end encryption.
- **Direct connectivity:** NAT traversal via STUN/TURN + WebRTC.
- **Rich messaging:** text, emoji, animated emoji, file transfer.
- **Open source:** MIT licensed, auditable, self-hostable infrastructure.

## High-level shape

Stunner is split into two layers:

1. **Go core** — all networking and cryptography. Compiled to a native library
   and embedded into the app. Has no UI.
2. **Flutter app** — the UI for every platform. Talks to the Go core over FFI.

```
┌───────────────────────────────────────────────────────────┐
│ Flutter UI (Dart)         Android · iOS · macOS · Windows   │
│   ChatsScreen · ConversationScreen · SettingsScreen         │
│   emoji picker · animated emoji (Lottie) · file picker      │
└───────────────────────────▲──────────────────────────────-─┘
                            │ dart:ffi
                            │  - synchronous calls (Version, SendMessage…)
                            │  - async event stream (incoming msgs, presence)
┌───────────────────────────┴───────────────────────────────┐
│ Go core                                                     │
│                                                             │
│  settings ── identity ── crypto(Signal) ── storage(SQLCipher)│
│      │           │            │                              │
│      └────► transport(WebRTC/pion) ◄── signaling(DHT)        │
│                   │                                          │
│            messaging · filetransfer · emoji                 │
└───────────────────────────▲───────────────────────────────┘
                            │ STUN / TURN (configurable)
                    ┌────────┴────────┐
                    │   Peer device    │
                    └─────────────────┘
```

## The Go core packages

| Package | Responsibility |
|---|---|
| `pkg/identity` | Long-term Ed25519 identity key + X25519 key agreement key; device IDs; fingerprints / "safety numbers". |
| `pkg/crypto` | Signal protocol session management (X3DH + Double Ratchet) wrapping a vetted library; AEAD helpers for files; in-memory key material handling. |
| `pkg/transport` | WebRTC peer connections and data channels (`pion/webrtc`); applies ICE (STUN/TURN) configuration. |
| `pkg/signaling` | `Signaler` interface for exchanging SDP/ICE; default decentralized DHT (libp2p Kademlia) rendezvous; optional relay impl. |
| `pkg/messaging` | Message model, conversations, send/receive pipeline, outbox + retry, delivery/read receipts. |
| `pkg/filetransfer` | Chunked, resumable, integrity-checked file transfer over a data channel. |
| `pkg/storage` | Encrypted local persistence (SQLCipher); DB key sourced from the OS secure store. |
| `pkg/emoji` | Emoji catalog, shortcode lookup, and animated-emoji manifest metadata. |
| `pkg/settings` | App configuration: ICE server overrides, relay toggle, app-lock options. |
| `mobile` | `gomobile`-friendly API surface for Android/iOS bindings. |
| `ffi` | `cgo` `//export` functions for the desktop C-shared library. |
| `cmd/stunnerd` | Headless dev/test node. |

## The FFI boundary

The Go core is consumed two ways:

- **Mobile (Android/iOS):** `gomobile bind` generates an `.aar` (Android) and
  `.xcframework` (iOS) from `core/mobile`. Only gobind-safe types are exposed
  (strings, ints, byte slices, simple structs, interface callbacks).
- **Desktop (macOS/Windows/Linux):** `go build -buildmode=c-shared` from
  `core/ffi` produces a `.dylib`/`.dll`/`.so` + header, called from Dart via
  `dart:ffi`.

To keep the surface stable and language-agnostic, calls cross the boundary as
**JSON** (or protobuf later) payloads. Asynchronous events (incoming messages,
presence, transfer progress) are delivered through a callback / `SendPort` so the
UI can subscribe to a stream.

The skeleton ships a minimal proof of the boundary: `Version()` and `Ping()`.

## Network model: pure P2P

There is **no central message server**. Two devices establish a direct,
encrypted WebRTC data channel and exchange Signal-encrypted payloads over it.

### NAT traversal (STUN/TURN)

WebRTC uses ICE to find a working path between peers:

- **STUN** discovers each peer's public address (for direct/hole-punched paths).
- **TURN** relays traffic when a direct path is impossible (symmetric NATs,
  restrictive firewalls). TURN only relays *encrypted* bytes; it cannot read
  message content.

Default ICE servers are public (see `pkg/settings`), and **fully overridable in
the Settings UI**. Operators are encouraged to run their own `coturn`.

### Signaling & discovery (still serverless)

WebRTC needs a side channel to exchange SDP offers/answers and ICE candidates,
and a way to find a peer's current endpoint. Stunner keeps this decentralized:

- A **`Signaler` interface** abstracts this step.
- The **default implementation** is a libp2p host participating in a Kademlia
  **DHT**. A user advertises and is discovered under a salted hash of their
  identity public key; SDP/ICE is exchanged over an authenticated libp2p stream.
- An **optional relay implementation** (a tiny self-hostable signaling server)
  is available for users who prefer it or who are on networks hostile to DHT
  traffic.

## Offline delivery

This is the central tradeoff of pure P2P and is called out explicitly:

- With no store-and-forward server, **both peers must be online** for a message
  to be delivered directly.
- The `messaging` package keeps an **outbox** and retries delivery when the peer
  reappears (presence via the signaling layer).
- For users who need true offline delivery, Stunner offers an **optional,
  self-hostable encrypted mailbox/relay** that holds ciphertext until the
  recipient fetches it. It is **off by default** because it reintroduces a
  metadata-bearing component; when enabled it still only ever sees E2E
  ciphertext.

## Cryptography overview

- **Identity:** each install generates an Ed25519 long-term identity keypair and
  X25519 prekeys. The public identity key is the user's cryptographic address.
- **Sessions:** the Signal protocol (X3DH handshake + Double Ratchet) provides
  end-to-end encryption with **forward secrecy** and **deniability**. Implemented
  by wrapping a maintained Signal library — **never hand-rolled**.
- **Files:** large files are chunked and each chunk sealed with an AEAD
  (XChaCha20-Poly1305) under a per-transfer key exchanged inside the secure
  session.
- **At rest:** the local database is SQLCipher-encrypted; the DB key lives in the
  platform secure store (Keychain / Android Keystore / Windows DPAPI / macOS
  Keychain) and is unlocked via biometric/PIN app-lock.

See [`THREAT_MODEL.md`](THREAT_MODEL.md) and [`PROTOCOL.md`](PROTOCOL.md).

## Libraries (planned)

| Concern | Library |
|---|---|
| WebRTC | `github.com/pion/webrtc/v4` |
| STUN/TURN client | `github.com/pion/stun`, `github.com/pion/turn` |
| Signal protocol | `go.mau.fi/libsignal` |
| Discovery | `github.com/libp2p/go-libp2p`, `go-libp2p-kad-dht` |
| Encrypted DB | `github.com/mutecomm/go-sqlcipher` |
| Mobile binding | `golang.org/x/mobile/cmd/gomobile` |
| AEAD / curves | Go stdlib (`crypto/ed25519`, `crypto/ecdh`), `golang.org/x/crypto` |

> The current skeleton uses **only the Go standard library** so it builds
> offline. External libraries above are introduced per phase in
> [`ROADMAP.md`](ROADMAP.md).
