# mg_advent

This is an advent calendar BBS Door. The BBS door is Linux only and has been tested on Ubuntu 23.04 x64. You can probably cross-compile to ARM based systems (e.g. Pi).
It also contains a DOS executable to run in dosbox.

## Contents
- BBS Door Executable & Source (advent + main.go, godoor.go)
- Example shell script for launching w/Door32.sys
- DOS Source (BP7) executable - not a BBS door - for running in dosbox (DOS/madvent.exe + madvent.pas)

## Usage
Build the Go Program as usual. Then, run ./advent with the --path flag and point to the directory containing door32.sys (include trailing slash and not "door32.sys"):
~~~~
./advent --path /path/to/dropfile/
~~~~
Or, you can run locally with the --local flag. This will attempt to utilize UTF-8 encoding instead of CP437:
~~~~
./advent --local
~~~~

## Notes
- BBS version ANSI files are in /art
- ANSI art files should be created @ 80x25 -- but 80th column should be EMPTY
- Placeholder WELCOME.ANS & GOODBYE.ANS art
- Requires door32.sys drop file if running via BBS
- CP437 via BBS, UTF-8 when running --local
- User will timeout after 1 minute no keyboard activity
- DOS verion uses a single .DAT file for calendar art

## ToDo
- [ ] Detect art more than 25 rows and allow user scrolling up/down
- [x] Add UTF8 support for local terminal usage
