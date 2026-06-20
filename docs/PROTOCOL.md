# Stunner Protocol (Draft)

This document specifies the wire-level behavior of Stunner. It is a **draft**
that tracks the skeleton; field formats will be pinned (and versioned) as the
implementation lands. Where concrete encodings are not yet final, they are
marked _TBD_.

## 1. Identifiers

- **Identity key:** Ed25519 public key (32 bytes). The canonical user address.
- **Key-agreement key:** X25519 public key, used by X3DH.
- **Device fingerprint:** `base32(SHA-256(identityPubKey))`, truncated and
  grouped for display. Used in QR codes and safety numbers.
- **Discovery key:** `SHA-256(identityPubKey || rendezvousSalt)` — what a peer
  advertises/looks up in the DHT, so the raw identity key is not the lookup key.

## 2. Layers

```
Application messages (JSON/protobuf)         ← pkg/messaging, pkg/filetransfer
  └─ Signal session (X3DH + Double Ratchet)  ← pkg/crypto      [E2E encryption]
       └─ WebRTC DataChannel (SCTP/DTLS)     ← pkg/transport   [transport security]
            └─ ICE path via STUN/TURN        ← pkg/settings    [NAT traversal]

Out of band:
  Signaling: SDP/ICE exchange + discovery    ← pkg/signaling   [DHT or relay]
```

Every application payload is encrypted by the Signal session **before** it is
handed to the data channel. The data channel's own DTLS is defense-in-depth, not
the primary confidentiality guarantee.

## 3. Connection establishment

1. **Discovery.** Initiator looks up the recipient's *discovery key* via the
   `Signaler` (DHT by default) to find a reachable signaling path.
2. **Signaling handshake.** Peers exchange WebRTC **SDP offer/answer** and **ICE
   candidates** over an authenticated signaling stream. ICE servers come from
   `pkg/settings` (STUN/TURN, overridable).
3. **Transport.** ICE negotiates a path (direct, hole-punched, or TURN-relayed);
   a DTLS-secured SCTP **data channel** opens.
4. **Secure session.** If no Signal session exists, peers run **X3DH** using
   published prekeys, then communicate via the **Double Ratchet**. Existing
   sessions resume.
5. **Verification (optional, recommended).** Users compare **safety numbers** or
   scan a **QR code** to authenticate identity keys out of band.

## 4. Application message framing

All application payloads share an envelope (encoding _TBD_: JSON first, protobuf
later). Conceptually:

```
Envelope {
  version:    uint     // protocol version
  type:       enum     // TEXT | FILE_OFFER | FILE_CHUNK | RECEIPT | TYPING | CONTROL
  msgId:      bytes    // unique per message (for receipts/dedup)
  convId:     bytes    // conversation identifier
  timestamp:  int64    // sender clock (advisory)
  body:       bytes    // type-specific payload (below)
}
```

The whole envelope is serialized, then encrypted by the Signal session.

### 4.1 TEXT
```
TextBody {
  text:     string         // UTF-8, may contain Unicode emoji
  entities: []Entity       // optional: mentions, links, emoji shortcodes
}
```
Animated emoji are referenced by id from the emoji manifest (see §6), not
embedded inline.

### 4.2 Receipts
```
ReceiptBody {
  refMsgId: bytes
  state:    enum   // DELIVERED | READ
}
```

### 4.3 Typing / presence
Ephemeral `TYPING` and `CONTROL` messages are not persisted.

## 5. File transfer

Files are transferred over the same secure session, chunked for flow control and
resumability.

1. **Offer.** Sender emits `FILE_OFFER`:
   ```
   FileOffer {
     fileId:    bytes
     name:      string
     size:      uint64
     mime:      string
     chunkSize: uint32        // e.g. 16 KiB
     hash:      bytes         // SHA-256 of full plaintext, for integrity
     transferKey: bytes       // per-transfer symmetric key (carried inside E2E session)
   }
   ```
2. **Accept / reject.** Receiver responds with a `CONTROL` accept/decline.
3. **Chunks.** Sender streams `FILE_CHUNK`:
   ```
   FileChunk {
     fileId: bytes
     index:  uint32
     data:   bytes     // AEAD(XChaCha20-Poly1305) sealed with transferKey + nonce(index)
   }
   ```
4. **Integrity & resume.** Receiver verifies the final SHA-256 against `hash`.
   Missing chunk indices can be re-requested via `CONTROL`, enabling resume.

Backpressure follows the data channel's buffered-amount thresholds.

## 6. Emoji & animated emoji

- **Static emoji** are ordinary Unicode in `TextBody.text`; rendered natively by
  the platform. `pkg/emoji` provides shortcode↔emoji lookup for the picker.
- **Animated emoji** are described by a manifest entry:
  ```
  AnimatedEmoji {
    id:       string        // stable id referenced from messages
    shortcode:string        // e.g. ":party_parrot:"
    format:   enum          // LOTTIE | APNG | WEBP
    assetRef: string        // bundled asset id or content hash
  }
  ```
  The renderer (Flutter `lottie` / image codecs) resolves `assetRef`. Custom
  animated emoji packs may be exchanged as files (§5) and registered locally.

## 7. Versioning

- `Envelope.version` gates parsing. Unknown future fields are ignored where the
  encoding allows (protobuf) to support forward compatibility.
- Breaking changes bump the major protocol version and are negotiated during the
  signaling handshake.

## 8. Open items (_TBD_)

- Final envelope encoding (JSON vs protobuf) and exact field tags.
- Group messaging (sender-keys vs MLS) — out of scope for the skeleton.
- Exact DHT record format and rendezvous salt rotation policy.
- Optional-relay mailbox fetch protocol.
