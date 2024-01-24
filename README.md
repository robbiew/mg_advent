# mg_advent

This is an advent calendar BBS Door from a scene group called Mistgris! 
The BBS door is Linux only and has been tested on Ubuntu 23.04 x64. You can probably cross-compile to ARM based systems (e.g. Pi).
It also contains a DOS executable to run in dosbox.

## Contents
- BBS Door Executable & Source (advent + main.go, godoor.go)
- Example shell script for launching w/Door32.sys
- DOS Source (BP7) executable - not a BBS door - for running in dosbox (DOS/madvent.exe + madvent.pas)

## Notes
- BBS version ANSI files are in /art
- Placeholder WELCOME.ANS & GOODBYE.ANS art
- Requires door32.sys drop file from BBS
- CP437 only, not unicode supported
- User timeout after 1 minute no keyboard activity
- 80x25 only
- DOS verion uses a single BIN file for calendar art

## ToDo
- Detect art more than 25 rows and allow user scrolling up/down
- Add UTF-8 support for local terminal usage
