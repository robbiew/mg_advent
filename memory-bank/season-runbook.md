# Season runbook: adding a year

The current example is 2026. Placeholders for it are already committed.

## 1. Placeholders (done for 2026)

```bash
go run ./tools/mkplaceholders -year 2027
```

Writes all 30 files into `internal/embedded/art/2027/`: WELCOME (year menu),
COMEBACK, GOODBYE, INFOFILE, MEMBERS and `01_DEC27.ANS` … `25_DEC27.ANS`.
Screens are framed 80x25 CP437 ANSI with SAUCE; day placeholders show the day
number large so navigation is easy to check. The welcome placeholder lists
the earlier years on the keys the door gives them. Existing files are never
overwritten (rerun safely after dropping some art in); `-force` regenerates
everything.

The year stays hidden until 1 December, so placeholders can be committed
early without callers seeing them.

## 2. Drop in the real art

Replace each placeholder with the artist's file **under the same name**.
Checklist for the artists:

- 80 columns, CP437, ideally 25 rows (taller day art scrolls).
- Avoid the 80th column; the door cuts lines to 79 cells anyway.
- WELCOME (and COMEBACK, which callers see after today's door) is the year
  menu: show `1`/`2`/`3`… for the earlier years (oldest = 1), `I` info,
  `M` member list, `Q` quit, arrows/Enter to browse. Optionally `A` credits,
  `S` slideshow and `?` help, which work on every screen.
- Fill in the SAUCE title, artist and group: `A` shows them to callers.
- INFOFILE and MEMBERS may be any height; they scroll above the footer.
- SAUCE records are fine; they are stripped.

Track what is left:

```bash
grep -rla ADVENT-PLACEHOLDER internal/embedded/art/2026   # files still to replace (-a: art looks binary)
```

The door also logs the remaining placeholders (`-log file`) when it opens the year.

## 3. Preview

```bash
./advent -local -debug-date=2026-12-03    # as callers see it on 3 December
./advent -local -debug-date=2026-12-25    # the whole season
./advent -local                           # today (2026 hidden before December)
```

`go test ./...` includes an end-to-end check that the new year is hidden in
November and opens on 3 December.

### On the local Vision/3 test board (`/opt/v3`)

The door is installed at `/opt/v3/doors/advent/advent` with three records in
`configs/doors.json` (category `HOLIDAY`, shown in the generated door menu on
the main menu's `D` key):

| Code | Command | Purpose |
| --- | --- | --- |
| `ADVENT` | `./advent -path {NODEDIR}` | the door on the real clock |
| `ADVENTDEC` | … `-debug-date=2026-12-03` | preview of the live 2026 season |
| `ADVENTLOGON` | `./advent -logon -path {NODEDIR} -debug-date=2026-12-03` | logon-mode preview; also the `DOOR:ADVENTLOGON` step in `configs/login.json` |

Redeploy after changing art: `go build -o /opt/v3/doors/advent/advent ./cmd/advent`.
Vision/3 reloads `doors.json`, `login.json` and `config.json` by itself. The
Holiday category header placeholder is `menus.d/v3/templates/DOORMENU_HOLIDAY.TOP`.
Before the real season, drop `-debug-date` from `ADVENTLOGON` (and remove
the preview records). Backups of the edited configs are the `*.bak-*` files
next to them.

## 4. Release

1. `grep -rla ADVENT-PLACEHOLDER internal/embedded/art` prints nothing.
2. Update the release-package files in `internal/embedded/` (FILE_ID.ANS,
   FILE_ID.DIZ, INFOFILE.ANS, MEMBERS.ANS, LICENSE.TXT) for the new pack.
3. Build, package and publish:

   ```bash
   VERSION=vX.Y.Z ./scripts/build.sh && ./scripts/package.sh
   ./scripts/release.sh vX.Y.Z --notes-file notes.md --dry-run
   ./scripts/release.sh vX.Y.Z --notes-file notes.md
   ```

   `build.sh` stamps `VERSION` (default: `git describe`) and the date into
   every binary; `package.sh` writes `checksums.sha256`. `release.sh` checks
   the tree is clean and pushed, the binaries carry the version, and no
   placeholders remain, then tags the commit and uploads the archives and
   checksums. `--yes` skips the prompts but never replaces an existing tag or
   release; `--dry-run` changes nothing.

   **Test builds** with placeholder art: `ALLOW_PLACEHOLDERS=1` for both
   scripts, a `-beta.N` version and `--prerelease` (v2.0.0-beta.1 was made this
   way). A suffixed version without `--prerelease` is refused under `--yes`.
