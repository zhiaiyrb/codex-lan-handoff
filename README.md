# Codex LAN Handoff

Securely transfer a structured task-context summary between two Codex sessions on computers in the same trusted LAN. It does **not** copy source files, uncommitted changes, raw chats, or Codex's internal conversation database.

This is an independent community project and is not an OpenAI product.

## Install the Codex skill

Ask Codex to install:

```text
https://github.com/zhiaiyrb/codex-lan-handoff/tree/v1.0.1/skills/codex-handoff
```

Then ask the installed skill to set up the CLI. The installer downloads the matching single-file binary from the pinned release and verifies its SHA-256 checksum.

## Pair once

On the first machine:

```sh
codex-lan-handoff pair init
```

Transfer the displayed key privately. On the second machine, start the hidden prompt:

```sh
codex-lan-handoff pair import
```

Paste the key and press Enter once. The key is not echoed. Piped input remains supported for automation.

Never paste the key into a Codex conversation or store it in a repository.

## Daily use

On the receiving machine, tell Codex:

```text
从 192.168.0.19 接手，等待交接。
```

On the sending machine:

```text
把当前工作交接到 192.168.0.20。
```

The receiver verifies the encrypted document, saves it to its user-level inbox, checks the actual local repository state, and continues. See [Usage](docs/USAGE.md) and [Protocol](docs/PROTOCOL.md).

## Security model

- AES-256-GCM authenticated encryption and HMAC-SHA256 acknowledgements with a pre-shared 32-byte key.
- One-shot listener restricted to an explicitly allowed source IP.
- 15-minute envelope lifetime, unique IDs, replay tracking, bounded frames, strict JSON fields, and atomic inbox writes.
- Basic secret detection rejects likely tokens, passwords, private keys, and credential-bearing URLs.
- No automatic firewall changes and no public-Internet exposure.

## Development

Requires Go 1.23 or newer.

```sh
go test ./...
go vet ./...
```

Licensed under the [MIT License](LICENSE).
