# Features

## Calendar

| Year | When it can be opened |
| --- | --- |
| Past seasons | Every day, all year round |
| Current year | One door per day, 1–25 December; paging past today's door shows COMEBACK.ANS |
| Future years | Hidden until 1 December of that year |

The door opens on the live season in December, otherwise the most recent past
season. NOTYET.ANS appears only when no installed year can be opened yet.
If December arrives before that year's art is installed, the door simply
opens the archive.

## Keys

| Screen | Keys |
| --- | --- |
| Welcome / Come back | `→` or Enter: open the calendar, at today's door during the live season (door 1 otherwise; resumes the last door viewed) · `1`–`9`: year by number (1 = oldest) · `[` `]` or PgUp/PgDn: previous/next year · `I` info · `M` members · `Q`/Esc: back, or quit from the newest year |
| Day | `←` `→`: previous/next door (past day 1 returns to Welcome) · Home/End: first/last open door · PgUp/PgDn: five doors · digits: jump to a door (3–9 or two digits jump at once; 1, 2, 0 wait for a second digit or Enter; Esc cancels; unopened doors say so) · `↑` `↓`: scroll art taller than the screen · `Q`/Esc: back to Welcome |
| Info / Members | `↑` `↓` line, PgUp/PgDn page, Home/End · `Q`/Esc: back |
| Any | `A`: SAUCE credits (title, artist, group, date, file) of the art on screen · `S`: slideshow of the year's open doors, `-slide` apart, looping (any key stops; not on Info/Members) · `?`: key help |

Outside December the welcome screen shows "The YYYY calendar opens in N days"
in the bottom-right corner. A slideshow still counts toward the idle limit on
a BBS (so it can't hold a node), but not in `-local` mode.

Year numbers follow the installed years, oldest first, so the art's menu must
match (2023 = 1, 2024 = 2, ...). `[` `]` have no limit, for when there are
more than nine years.

## Logon mode (`-logon`)

During a live season: shows today's door, discards type-ahead (so the Enter
that started the logon event can't dismiss it), then waits 30 s or for a key.
Outside the season: exits immediately with no output and no terminal probe,
so it can stay in a logon event all year. COMEBACK.ANS is deliberately not
shown (b1066e8): the 2025 version is a menu whose keys don't work there.

## Flags

| Flag | Meaning |
| --- | --- |
| `-path` | door32.sys, or the directory containing it (any case) |
| `-logon` | logon mode, above |
| `-local` | run in a local terminal (UTF-8 output) |
| `-nodetect` | skip the size probe; use 80x25 |
| `-noice` | don't send the blink-mode sequence |
| `-log file` | append a detailed log; default is errors only, to stderr |
| `-slide duration` | slideshow interval (default 8s) |
| `-debug-date YYYY-MM-DD` | pretend today is this date (calendar only) |
| `-debug-disable-date` | unlock every installed year and day |
| `-version` | version and build date |

## Limits and fallbacks

- 5 minutes idle, or the caller's BBS time left (max 2 h), then a notice and
  back to the BBS. A caller with no time left is sent back before any art.
- Terminal sizes below 80x24 (or an unclamped 999) fall back to 80x25.
- A missing day shows MISSING.ANS labelled with the expected file name; a
  missing year screen uses art/common.
