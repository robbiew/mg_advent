# Memory Bank: MiSTiGRiS Advent Calendar door

Working notes for people (and AI assistants) changing this code. They
describe the door as it is now; the repository README is the sysop-facing
manual. Last reviewed: November 2026 (preparing the 2026 season).

| File | What it covers |
| --- | --- |
| [architecture.md](architecture.md) | Packages, data flow, and the invariants the code relies on |
| [features.md](features.md) | What callers and sysops see: calendar rules, keys, flags, logon mode |
| [season-runbook.md](season-runbook.md) | Adding a year: placeholders, dropping in art, previewing, releasing |
| [progress.md](progress.md) | History of the code, recent changes, and open items |

## Status

- Seasons 2023, 2024 and 2025 are released and open all year as an archive.
- **2026** is installed as generated placeholder art (`internal/embedded/art/2026/`).
  It stays hidden until 1 December 2026. Real art replaces the placeholders
  file by file; `scripts/build.sh` and `scripts/release.sh` refuse to run
  while any placeholder is still embedded.
- Open GitHub issue: #2 (Windows telnet carrier drop): fixed in code by the
  GoDoors move, awaiting confirmation on a live Windows board.

## Ground rules

- Go 1.26+ (`go.mod` pins toolchain go1.26.10). `go test -race ./...`, `go vet`
  and gofmt must pass; CI runs all three on Linux, macOS and Windows.
- BBS transport, terminal probes and key input belong to
  [GoDoors](https://github.com/robbiew/godoors). Don't reimplement them here.
- Never write caller-facing output to `os.Stdout` directly; use the display
  engine (or `gd.Output()` before it exists).
- Keep these notes true. If a change makes a statement here wrong, fix it in
  the same commit.
