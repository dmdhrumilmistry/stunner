# Stunner Threat Model

This document describes what Stunner protects, who it protects against, and the
limits of those protections. It is a living document and will tighten as the
implementation matures.

## Assets to protect

1. **Message content** — text, emoji, and files exchanged between users.
2. **Long-term identity keys** — the Ed25519/X25519 keys that define a user.
3. **Local data at rest** — conversation history, contacts, keys on the device.
4. **Communication metadata** — who talks to whom, when, and how often.

## Adversaries considered

| Adversary | Capability | Stunner's stance |
|---|---|---|
| **Network eavesdropper** | Observes traffic on the wire (Wi-Fi, ISP). | All content is E2E encrypted (Signal); transport adds DTLS. Eavesdropper sees ciphertext only. |
| **Malicious TURN/STUN operator** | Relays or observes connection traffic. | TURN relays only E2E ciphertext; cannot read content. Users can override with their own servers. |
| **DHT participant / signaling observer** | Sees discovery lookups. | Discovery keyed by *salted hash* of identity key; SDP/ICE exchanged over authenticated streams. Some connection metadata is inherently observable (see limitations). |
| **Optional relay operator** (if enabled) | Holds queued messages. | Only ever sees E2E ciphertext; off by default. |
| **Thief with the locked device** | Physical access, device locked. | Local DB encrypted at rest (SQLCipher); key in OS secure store; app-lock (biometric/PIN). |
| **Active MITM during contact add** | Tries to substitute keys. | Safety numbers + QR verification let users confirm keys out-of-band. |

## Explicitly out of scope

- **Fully compromised / rooted device** with the app unlocked and running. If
  the OS is owned by the attacker, no app can protect in-memory plaintext.
- **Targeted endpoint malware / keyloggers.**
- **Traffic-analysis by a global passive adversary.** Pure P2P reduces central
  metadata, but timing/volume correlation by an adversary watching both ends is
  not fully defeated.
- **Availability of third-party public STUN/TURN servers.** These are best
  effort; self-hosting is recommended for reliability.

## Protections and how they're achieved

### Confidentiality & integrity
- **End-to-end encryption** of all messages and files via the Signal protocol
  (X3DH + Double Ratchet): forward secrecy and post-compromise security.
- **AEAD** (XChaCha20-Poly1305) for file chunks under per-transfer keys.
- **Transport encryption** (WebRTC DTLS / SCTP-over-DTLS) as defense in depth.

### Authentication
- Each user is identified by their **public identity key**.
- **Safety numbers** (a stable fingerprint of both parties' keys) and **QR
  scanning** allow out-of-band verification, defeating active MITM.
- Key-change warnings surface when a contact's identity key changes.

### Metadata minimization
- **No central message server** that logs social graphs.
- DHT discovery uses a **salted hash** of the identity key rather than the raw
  key or any human identifier.
- Optional relay is **opt-in** and content-blind.

### Data at rest
- **SQLCipher**-encrypted local database.
- DB key stored in the **platform secure store** (Keychain, Android Keystore,
  Windows DPAPI, macOS Keychain), never in plaintext on disk.
- **App-lock** (biometric/PIN) gates decryption of the local store.

### Planned hardening (roadmap)
- **Disappearing messages** with per-conversation timers.
- **Sealed-sender**-style techniques to further reduce metadata.
- Reproducible builds and release signing for supply-chain integrity.

## Known limitations (be honest with users)

- **Online-presence leakage:** because delivery is direct P2P, establishing a
  connection reveals to the peer (and potentially to network observers) that you
  are online and reachable.
- **Offline delivery requires a tradeoff:** either both peers online, or the
  opt-in relay (which is a metadata-bearing component, though content-blind).
- **iOS background constraints:** reliable background wake on iOS effectively
  requires push (APNs), which introduces some metadata; this is future work and
  will be documented when implemented.
- **Public infrastructure:** default STUN/TURN servers are operated by third
  parties; for sensitive use, self-host.

## Status of the cryptography (important)

The current `pkg/crypto` implements X3DH and the Double Ratchet **from scratch**
over the Go standard library's vetted primitives (AES-256-GCM, HMAC-SHA256,
SHA-512, X25519 via `crypto/ecdh`). The *primitives* are not hand-rolled, but the
*protocol composition* is, and it has **not been independently audited**.

- Do not rely on this for high-risk use until it is reviewed.
- Swapping in a maintained implementation (e.g. `go.mau.fi/libsignal`) behind the
  existing `crypto.Session` / `crypto.SessionStore` interfaces is a tracked
  option and would not change the rest of the core.
- Message headers are authenticated as AEAD associated data, and every message
  binds both parties' identity keys (sorted) to resist unknown-key-share.

## Cryptographic discipline

- Never implement primitives by hand; use vetted libraries (see
  [`ARCHITECTURE.md`](ARCHITECTURE.md#libraries-planned)).
- Keep secret key material out of logs and crash reports.
- Zero/minimize lifetime of plaintext and key material in memory where feasible.

Report security issues per [`../SECURITY.md`](../SECURITY.md).
