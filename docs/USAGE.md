# Usage

## Status and scope

The v1 workflow transfers task context only. It does not synchronize repositories, assets, diffs, or conversation history. Use Git or another deliberate file-transfer mechanism separately.

## Install

Install `skills/codex-handoff` with the Codex skill installer, then run the bundled OS installer. Installers are pinned to `v1.0.1`, download the matching release archive, and verify `checksums.txt`.

Run `codex-lan-handoff doctor` to inspect the version, platform, configuration directory, pairing status, addresses, default port, and inbox state.

## Pair

Run `pair init` on either machine. On the peer, run `pair import`, paste the key into the hidden prompt, and press Enter once. No Ctrl+Z/Ctrl+D is required. Piped stdin remains supported for automation and consumes only its first line. The key is stored below the operating system user configuration directory. Never send it in a handoff.

## Receive and send

Start the receiver first:

```sh
codex-lan-handoff receive --from 192.168.0.19 --timeout 10m
```

Then send a Schema v1 JSON document:

```sh
codex-lan-handoff send --to 192.168.0.20 --file handoff.json
```

The default TCP port is `47128`. Override it with `--port` on both sides. A successful receiver exits after one accepted document. Invalid connections are rejected while the receiver continues until its timeout.

Use `codex-lan-handoff inbox latest` after an interrupted Codex task.

## Firewall and network boundary

The program never changes firewall settings. If required, allow inbound TCP 47128 only on the private network profile and ideally only from the peer address. Do not forward this port through a router or expose it publicly. A private overlay such as ZeroTier is acceptable when both peers and routes are trusted.

## Rollback and removal

Remove the installed binary and the `codex-handoff` skill directory. User configuration and received summaries remain in the OS user configuration directory until manually removed. Back up desired summaries before removal. Deleting the shared key requires pairing again.

## Troubleshooting

- `connection refused`: start the receiver first and verify address/port.
- `receive timed out`: verify routing and private firewall rules.
- `authentication failed`: pair both machines with the same key.
- `source IP not allowed`: use the address the receiver actually observes.
- `possible secret detected`: remove the credential from the summary; never bypass the check.
- Repository mismatch after receipt is expected when Git/file synchronization has not occurred.
