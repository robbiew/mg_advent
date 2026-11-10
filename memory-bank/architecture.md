# Architecture

## Layout

```
cmd/advent/              main.go: flags, startup, logon mode, key reading
                         door.go: the interactive loop (door type), help,
                         credits, door-number entry, slideshow, scrollView
internal/
  art/                   screen -> art path resolution; placeholder marker
  display/               ANSI rendering engine (CP437), scrolling, SGR tracking
  embedded/              go:embed of art/ (all art is compiled in)
    art/<year>/          one season: WELCOME, COMEBACK, GOODBYE, INFOFILE,
                         MEMBERS, 25 daily doors
    art/common/          fallbacks: MISSING, NOTYET, GOODBYE, FOOTER
  input/                 maps GoDoors keys to door keys; Drain for type-ahead
  navigation/            calendar rules (which years/days are open) and
                         screen-to-screen movement
  session/               idle and total time limits (no timers, no goroutines)
  validation/            art, terminal and emulation checks; placeholder scan
tools/mkplaceholders/    generates placeholder art for a new year
tools/packans/           DOS-build packing tool (binary is not tracked)
scripts/                 build.sh, package.sh, release.sh, check-placeholders.sh
dos/                     legacy Turbo Pascal version, not maintained
```

## Startup (cmd/advent `run`)

1. Parse flags; set up `slog` (errors to stderr, or everything to `-log`).
2. Configure the calendar clock (`-debug-date`, `-debug-disable-date`).
3. Logon mode outside a live season returns here, before any output.
4. `gd.OpenSession(dir)` for `-path` (file or directory), then `OpenInput`.
5. Probe the terminal size unless `-nodetect`; implausible answers (narrower
   than 80, shorter than 24, or 999) fall back to 80x25.
6. Start the session limits: 5 min idle, total = caller's door32.sys time
   (capped at 120 min). Zero time left shows a notice and exits at once.
7. Logon mode or the main loop. Deferred cleanup restores the cursor, clears
   the screen, resets colours and closes the GoDoors session.

## Invariants

**Calendar** (`navigation.Navigator.MaxDay`): a past year is open in full all
year; the current year opens one day at a time in December (capped at 25);
a future year is hidden (MaxDay 0, absent from `VisibleYears`). The door opens
on the newest visible year. `-debug-date` replaces the calendar clock only;
session limits use real time. Never gate the whole door on the month.

**Art stays CP437.** The engine keeps art as CP437 bytes so one byte is one
cell, which the 80-column logic depends on. Local mode converts to UTF-8 only
in the output writer (`utf8Writer`). Converting earlier splits multi-byte
characters.

**Output is buffered.** `DisplayEngine` writes into a 64 KB `bufio.Writer`;
every public render method flushes once at the end, so a screen is one write
to the BBS. A new public method must flush.

**Art loading** (`loadAndProcess`, cached per path):
`gd.TrimStringFromSauce` (anchored SAUCE + exact COMNT length + EOF), split on
CRLF, drop trailing empty lines (25-row art with a final CRLF must not become
26 rows), then `toRows` splits the lines into the **screen rows** they fill at
80 columns. Much art is drawn for the terminal's own wrap (one line of 160
cells fills two rows; five bundled files are a single 2,000-cell stream), so
everything downstream (fit, scroll state, drawing) works in rows, never in
file lines. `ESC[nC` counts as n cells and is split across rows. Lines are
never cut: `TestBundledArtKeepsItsContent` checks no cell is lost (cutting to
79 once wiped most of seven 2025 doors).

**Drawing** positions every row explicitly (`ESC[r;1H`), so a full 80-cell
row never depends on the terminal's wrap or last-column flag; only a row in
the bottom-right corner stops at column 79, so the screen can't scroll. Art
that positions the cursor itself (`hasPositioning`; none bundled today) is
drawn as a stream with the door's original fixes (`legacyFit`) and doesn't
scroll.

**Scrolling** repaints rows in place (`drawRows(..., repaint=true)`); art
taller than the terminal in rows (e.g. 25-row art on 80x24) scrolls. Each repainted row first restores the SGR state the lines
above it left in effect (`sgrState`), because art often sets a colour once
for several lines.

**Blink/iCE**: `EnableBlinkMode` sends `CSI ?33l` (CTerm DECRST 33: blink, the
default) once at start, unless `-noice`. Nothing is sent at exit. Do not use
`CSI =0h`/`=0l`: undocumented in CTerm and an ANSI.SYS screen-mode switch.

**Session limits** are deadlines, not timers: `session.Manager.Next` says how
long the next `ReadKeyTimeout` may wait. Expiry is handled on the main
goroutine so deferred cleanup always runs. Every key (including scrolling)
counts as activity.

**Art paths** (`art.Manager.GetPath`): year screens prefer the year's file,
then `art/common`, else the year path (shown as MISSING.ANS labelled with that
name). Day files: 2025 onward `01_DEC25.ANS`, earlier `1_DEC24.ANS`; both names
are accepted for any year (`art.DayFileNames`).

**Placeholders** carry `art.PlaceholderMarker` (`ADVENT-PLACEHOLDER`). The
validator logs them for the opened year; tests keep them out of released
seasons; the build/release scripts refuse to ship them.

## Transport (GoDoors)

- Linux/macOS: the BBS supplies the caller on stdin/stdout for every comm
  type. Under a PTY (e.g. Vision/3 "Raw Terminal") the size comes from the PTY.
- Windows, comm type 2: GoDoors duplicates the inherited socket with
  `WSADuplicateSocket` and does overlapped I/O on its private copy, never
  changing the shared socket's mode. The pre-GoDoors code changed it to
  non-blocking and closed it, which dropped carrier (issue #2).
- Keys typed during a terminal probe are kept for the next read.
