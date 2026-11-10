# Progress and history

## History

- **Pre-2025**: single-file Go door, December-only, one year of art.
- **2025 modernization** (Nov 2025): split into `internal/` packages,
  embedded art for 2023–2025, multi-year browsing, Windows socket support,
  logon mode. The planning documents from that work (requirements,
  implementation plan, code analysis) are in git history before
  November 2026; much of what they planned (themes, animation, plugins,
  LRU cache) was never built or was later removed.
- **PR #4** (Oct 2026): BBS transport, terminal probes and key input moved
  to GoDoors; the hand-written Windows socket code was removed.

## November 2026 audit and follow-up

**PR #5** (merged): calendar opens past seasons all year instead of only in
December (sponsor request); future years hidden; logon mode drains
type-ahead and waits 30 s, silent off-season; implausible terminal sizes fall
back to 80x25; session limits follow door32.sys time left and expire
cleanly (also zero time left); one-line `-path` setup documented (closed #3).

**Branch `chore/display-perf-and-cleanup`**: buffered output (one write per
screen); CP437 kept internally with UTF-8 conversion on output (fixed broken
local-mode characters); SAUCE via GoDoors; trailing-CRLF art no longer
scrolls; in-place scrolling; correct CTerm blink sequence; ~1,200 lines of
dead code removed; logrus → slog with `-log`; Go 1.26; CI gofmt/vet.

**Branch `feat/2026-placeholders`** (stacked on the above): 2026 placeholder
art and `tools/mkplaceholders`; placeholder guard in validator, tests and
build/release scripts; colours preserved when scrolling (matters on 80x24
terminals, where 25-row days scroll); `[` `]` / PgUp PgDn year stepping
beyond key 9.

**Branch `feat/navigation-extras`**: Enter opens today's door in season;
Home/End and PgUp/PgDn on doors; type a door number to jump; `?` help;
`A` SAUCE credits; `S` slideshow (`-slide`); off-season countdown; main loop
moved into a `door` type (cmd/advent/door.go). Also fixes a regression from
PR #6: lines were cut to 79 cells, destroying art drawn for autowrap
(2025 doors 1, 7, 9, 12, 14, 17, 25). Art is now split into the screen rows
it fills and drawn row by row, so nothing is cut, column 80 shows, and
25-row art scrolls on 80x24 instead of overflowing (Codex review on #8).

## Open items

- **Issue #2**: confirm on a live Windows board (e.g. Synchronet) that telnet
  carrier holds after the door exits. Linux/Vision/3 tested fine.
- **Replace 2026 placeholders** with the real pack before 1 December 2026;
  update the release-package files in `internal/embedded/`.
- **Year menus in art**: number keys only reach nine years; existing art
  shows the numbers, `[` `]` cover the rest. Artists should mention `[` `]`
  from about 2031.
- **iCE colour restore**: the door leaves blink mode on at exit (the terminal
  default). Restoring a board that runs iCE would need a DECRQM query
  (`CSI ?33$p`) before changing it.
- **`dos/`**: legacy version, not maintained or built in CI.
