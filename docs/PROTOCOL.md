# Protocol v1

## Transport

TCP carries one request envelope and one authenticated acknowledgement. Every JSON frame starts with a four-byte unsigned big-endian length. Frames are limited to 1 MiB. The default port is 47128.

## Envelope

The envelope contains `version`, a random 128-bit hexadecimal `id`, an RFC3339 UTC `created_at`, an AEAD `nonce`, and `ciphertext`. The authenticated additional data is:

```text
codex-lan-handoff|<version>|<id>|<created_at>
```

Encryption is AES-256-GCM with the paired 32-byte key. Envelopes older than 15 minutes or more than one minute in the future are rejected. Accepted IDs are persisted to prevent replay.

## Plaintext

Plaintext is the strict handoff Schema v1 JSON documented with the skill. Unknown fields, missing core fields, documents over 512 KiB, and likely credentials are rejected before sending and again after decryption.

## Commit and acknowledgement

The receiver authenticates, validates, checks replay state, writes a restricted temporary file, syncs it, atomically renames it into the inbox, records the ID, and then sends an acknowledgement authenticated with HMAC-SHA256 under the paired key. A lost acknowledgement may make the sender report failure even though the receiver durably accepted the handoff; check `inbox latest` before retrying.

## Compatibility

Protocol and schema versions are explicit. v1 implementations reject unknown envelope or document fields and unsupported versions instead of guessing. Future incompatible changes require a new protocol version; releases follow semantic versioning.

## Threat boundary

The design protects confidentiality and integrity from passive LAN observation and unauthenticated modification when the pre-shared key remains secret. Source-IP restriction reduces accidental or unauthorized peers but is not authentication. It does not protect a compromised endpoint, leaked key, malicious content deliberately written by a paired peer, traffic analysis, or denial of service.
