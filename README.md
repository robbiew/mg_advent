# MiSTiGRiS Advent Calendar

<img src="images/WELCOME.png" alt="Welcome Screen" width="100%">

An interactive BBS door program that displays ANSI Christmas art, a new one each day in December. Browse past collections as well. Compatible with all modern BBS systems that support Door32 dropfiles -- Windows and Linux.

## Unified Repository Structure (2025+)

This repository now contains all versions of the Advent Calendar:

- **Go/Modern (Linux, Windows, Mac):** Main Go codebase in root (`cmd/`, `internal/`, etc.)
- **DOS (Legacy):** All Turbo Pascal and DOS-specific files are in `dos/` (NOT 100% functional!! see below)

### Directory Layout

```text
mg_advent/
├── cmd/advent/              # the door (entry point, main loop, logon mode)
├── internal/                # Go packages: art, display, navigation, session, ...
│   └── embedded/art/        # all art, compiled into the binary
│       ├── 2023/ 2024/ 2025/    # released seasons
│       ├── 2026/                # this season (placeholder art until delivered)
│       └── common/              # fallback screens
├── tools/mkplaceholders/    # generates placeholder art for a new year
├── scripts/                 # build, package and release scripts
├── memory-bank/             # maintainer notes: architecture, runbook, history
└── dos/                     # legacy Turbo Pascal version (src/, build/)
```

## Quick Setup

Point `-path` at the node's `door32.sys`, or at the directory that contains it
(the file name is matched case-insensitively). No wrapper script is needed; use
whichever your BBS's door command can pass directly:

```text
mg-advent.exe -path C:\bbs\temp\%1             (Windows: node directory)
mg-advent.exe -path C:\bbs\temp\%1\door32.sys  (Windows: drop file)
./advent -path /opt/bbs/temp/%1                 (Linux)
```

Here `%1` stands for whatever node number or drop-file path your BBS
substitutes. All art is built into the executable, so the working directory
doesn't matter.

On Linux, make the binary executable: `chmod +x advent`

To show today's door to callers as they log on, add a second entry to your
logon event with `-logon` added. It shows the day's art for 30 seconds or until
a key is pressed, then returns to the BBS. Outside the December season it exits
immediately without output, so it can stay enabled all year.

### DOS (Legacy)

All DOS code and assets are in `dos/`. You need Turbo Pascal 7 or compatible.

**To build/run in DOSBox or real DOS:**

1. Enter the `dos/` directory:

 ```text
 cd dos
 ```

1. Compile `src/advent.pas` with Turbo Pascal 7 / Free Pascal, or use
   `build/dos_build.sh` (`build/dos_build.bat` on DOS).
2. Run the resulting executable in DOS or DOSBox.

## Command Line Options

```text
-path string           door32.sys, or the directory containing it
-logon                 Logon mode: show today's door during the season, then exit
-debug-date string     Pretend today is this date (YYYY-MM-DD)
-debug-disable-date    Unlock every installed year and day
-local                 Run locally with UTF-8 art conversion
-nodetect              Use 80x25 without terminal probes
-noice                 Don't send the blink/iCE colour mode sequence
-log file              Append a detailed log to this file
-version               Show version information
```

Without `-log`, only errors are written, to stderr. Some BBSes connect a
door's stderr to the caller's terminal; use `-log` to keep logs off-screen.

Sessions end after 5 minutes without a key, or when the caller's BBS time
(from `door32.sys`, at most 2 hours) runs out.

If a terminal reports an implausible screen size (narrower than 80 columns or
shorter than 24 rows) the door uses 80x25. Use `-nodetect` for terminals that
answer the size probe badly.

## How the Calendar Opens

- **Past seasons are open all year.** Every day of every earlier year can be
  browsed at any time, including outside December.
- **The current year opens one door per day in December**, up to the 25th.
  Paging past today's door shows `COMEBACK.ANS`.
- **Future years stay hidden.** A year's art can be installed early; callers
  can't see it until December 1st of that year.
- The door opens on the newest year that can be viewed: the live season in
  December, otherwise the most recent past season. `NOTYET.ANS` appears only
  if no installed year can be opened yet.

## Adding a New Year

The year's art lives in `internal/embedded/art/<year>/`. Start from
placeholders, then replace them as the pack is delivered:

1. Generate placeholders for every file the door needs:

   ```bash
   go run ./tools/mkplaceholders -year 2027
   ```

   This writes `WELCOME.ANS` (the year menu), `COMEBACK.ANS`, `GOODBYE.ANS`,
   `INFOFILE.ANS`, `MEMBERS.ANS` and `01_DEC27.ANS` … `25_DEC27.ANS` (2025
   onward use two-digit days). It never overwrites a file that already
   exists. 2026 placeholders are already in the repository.
2. Drop each real file in under the same name. The welcome and come-back
   screens are the year menu: `1` is the oldest year, counting up; `I`, `M`
   and `Q` are info, members and quit. List what is left with
   `grep -rl ADVENT-PLACEHOLDER internal/embedded/art`.
3. Preview with `./advent -local -debug-date=2026-12-03` (as callers see it
   on the 3rd) or `-debug-date=2026-12-25` (the whole season). The new year
   stays hidden from callers until 1 December, so it can ship early.
4. Release. `scripts/build.sh` and `scripts/release.sh` refuse to run while
   any placeholder is embedded (`ALLOW_PLACEHOLDERS=1` for a test build).
   Update the release-package files in `internal/embedded/` (FILE_ID, INFOFILE,
   MEMBERS, LICENSE) for the new pack.

A year missing its own WELCOME/COMEBACK/GOODBYE uses `art/common`, and a
missing day shows `MISSING.ANS`, so a partly delivered year still runs. The
full checklist is in [memory-bank/season-runbook.md](memory-bank/season-runbook.md).

## Building from Source

### For Modern Systems (Windows 10+, Linux, Mac)

Prerequisites: Go 1.26 or later, Git

```bash
# Clone repository
git clone https://github.com/robbiew/mg_advent.git
cd mg_advent

# Linux/Mac build (builds Linux amd64, arm64, and Windows 386)
./scripts/build.sh

# Windows build
go build -o mg-advent.exe ./cmd/advent
```

### BBS transport and Windows support

BBS connections, terminal probes, single-key input, and raw terminal setup now
use [GoDoors](https://github.com/robbiew/godoors). The Advent display engine keeps
its CP437 art processing, local UTF-8 conversion, overlays, scrolling, and ICE
settings. Navigation keys, local mode, logon mode, and the session limits
remain application features.

`-path` still accepts the full path to `door32.sys` (case-insensitive filename);
a directory containing it is also accepted. Windows uses the inherited socket
when supplied, while Linux/macOS and redirected Windows standard I/O use the
standard streams. No connection is made to a port inferred from a socket handle.
`-nodetect` opens the connection without issuing terminal queries. Detection
shares its buffered input with navigation so typed keys are retained.

Go 1.26 or later is required. Current Windows builds require Windows 10 / Server
2016 or later. The former Go 1.20 / Windows 7 build path is retired; the DOS tree
is unchanged. `WINDOWS7-FIX.md` is historical documentation for older releases.

Build directly on Windows with `go build -o mg-advent.exe ./cmd/advent`. The Unix
build script still produces Linux amd64/arm64 binaries and `mg-advent.exe` for
Windows 386, using the installed Go toolchain. Windows resources are optional.

When developing companion GoDoors API changes in a sibling checkout, create an
ignored local workspace with `go work init . ../godoors`. The module dependency
is pinned in `go.mod`; no local filesystem replacement is required in that file.
The companion revision must be published before a fresh clone or CI can download
it. Until then, use the sibling `codex/mg-advent-integration-api` branch in that
local workspace.

## Testing

```bash
go test -race ./...

# Run locally (no BBS required)
./advent -local                          # today's real calendar
./advent -local -debug-date=2026-12-15   # as callers will see it on Dec 15, 2026
./advent -local -debug-disable-date      # everything unlocked
```

## Usage

| Screen | Keys |
| --- | --- |
| Welcome | `→` or Enter: open the calendar · `1`–`9`: pick a year (1 = oldest) · `[` `]` or PgUp/PgDn: previous / next year · `I` info · `M` members · `Q`/Esc: quit |
| Day | `←` `→`: previous / next door · `↑` `↓`: scroll tall art · `Q`/Esc: back to Welcome |
| Info, Members | `↑` `↓`, PgUp/PgDn, Home/End: scroll · `Q`/Esc: back |

## License

This project is released under the terms specified in the LICENSE file.

---

### Maintainer notes

`memory-bank/` describes the architecture, the rules the code relies on,
the yearly runbook and the project history. The `dos/` directory is a work in
progress and is not fully functional; the focus is the Go version.
