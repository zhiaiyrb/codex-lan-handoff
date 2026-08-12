# Repository instructions

- Keep dependencies minimal. `golang.org/x/term` is approved for hidden, cross-platform key entry; other dependencies require a reviewed security or portability need.
- Never log, test-fixture, commit, or print stored pre-shared keys. Tests must use synthetic keys.
- Preserve strict protocol/schema decoding and bounded input sizes.
- Update `docs/PROTOCOL.md`, `docs/USAGE.md`, and the skill when commands, wire format, storage, security, or workflow changes.
- Run `go test ./...`, `go vet ./...`, formatting checks, and the skill validator before delivery.
- Do not claim that this project migrates Codex conversation history or synchronizes files.
