# Security Policy

Stunner is privacy and security software. We take vulnerabilities seriously and
appreciate responsible disclosure.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, report privately via one of:

- GitHub Security Advisories: use the **"Report a vulnerability"** button under
  the repository's **Security** tab (preferred).
- Email the maintainers (see repository profile) with the subject
  `STUNNER SECURITY`.

Please include:

- A description of the issue and its impact.
- Steps to reproduce (proof-of-concept where possible).
- Affected version / commit and platform.
- Any suggested remediation.

We aim to acknowledge reports within **72 hours** and to provide a remediation
timeline after triage. Please give us a reasonable window to fix the issue
before any public disclosure (coordinated disclosure).

## Scope

In scope:

- The Go core (`core/`): cryptography, key handling, transport, storage.
- The Flutter app (`app/`): FFI boundary, local data handling, UI leaks of
  sensitive data.
- Build/release integrity.

Out of scope (for now):

- Denial of service against public STUN/TURN servers (these are third-party).
- Issues requiring a fully compromised device / rooted OS with the app unlocked.
- Social-engineering of users.

## Cryptography principles

- **Never hand-roll cryptographic primitives.** Use the vetted libraries
  referenced in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) (Signal protocol
  via a maintained library; standard AEAD constructions).
- All message and file content is end-to-end encrypted.
- Long-term keys never leave the device unencrypted; the local database is
  encrypted at rest with a key held in the platform secure store.

See [`docs/THREAT_MODEL.md`](docs/THREAT_MODEL.md) for the full threat model.

## Supported versions

During the pre-1.0 phase, only the latest `main` is supported. A formal support
matrix will be published with the first tagged release.
