MiSTiGRiS ADVENT CALENDAR BBS DOOR
Programmed by J0hnny A1pha
https://github.com/robbiew/mg_advent
====================================

COMMAND LINE SWITCHES
---------------------
The Mistigris Advent Calendar BBS Door supports the following command line switches. 
Only "-path" is required:
      
  -path <path>
      door32.sys, or the directory containing it

  -logon
      Logon mode: show today's door (30 seconds or until a key is
      pressed), then exit. Outside the December season it exits at once
      without output, so it can stay in your logon event all year.

  -nodetect
      Skip terminal size detection and use 80x25

  -noice
      Don't send the blink/iCE colour mode sequence

  -slide <duration>
      Slideshow speed, e.g. 5s (default 8s)

  -log <file>
      Append a detailed log to this file. Without it, only errors are
      written, to stderr (which some BBSes show to the caller).

  -debug-date <YYYY-MM-DD>
      Pretend today is this date, for previewing a season

  -debug-disable-date
      Unlock every installed year and day

  -version
      Show version and author information


KEYS
----
  Welcome   Right arrow / Enter: open the calendar
            1-9: pick a year (1 = oldest)   [ ] or PgUp/PgDn: prev/next year
            I: info file   M: member list   Q / Esc: quit
  Day       Left / Right: previous / next door   Up / Down: scroll tall art
            Home / End: first / last open door   PgUp / PgDn: five doors
            Type 1-25: jump to that door   Q / Esc: back to welcome
  Anywhere  A: art credits   S: slideshow (any key stops)   ?: key help


HOW THE CALENDAR OPENS
----------------------
Past years can be browsed in full at any time of year. The current year
opens one door per day from December 1st to 25th. Years whose season has
not started yet stay hidden.

Sessions end after 5 minutes without a key or when the caller's BBS time
runs out (at most 2 hours).


EXAMPLES
--------
Run as a BBS door (no batch file needed):
  advent -path /path/to/node/dir
  advent -path /path/to/door32.sys

Run in logon mode:
  advent -logon -path /path/to/node/dir


WINDOWS SUPPORT
---------------
Current releases use GoDoors and require Windows 10 / Server 2016 or later.
Windows 7 and the former Go 1.20 build path are no longer supported.

IMPORTANT NOTE FOR LINUX USERS
------------------------------
If possible, build your executable from the Go source to avoid any version 
incompatibilities (like GLIB-C):

https://github.com/robbiew/mg_advent