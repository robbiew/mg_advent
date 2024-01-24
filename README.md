# mg_advent

This is an advent calendar BBS Door from a scene group called Mistgris! 
The BBS door is Linux only and has been tested on Ubuntu 23.04 x64. You can probably cross-compile to ARM based systems (e.g. Pi).
It also contains a DOS executable to run in dosbox.

## Contents
- BBS Door Executable & Source (advent + main.go, godoor.go)
- Example shell script for launching w/Door32.sys
- DOS Source (BP7) executable - not a BBS door - for running in dosbox (DOS/madvent.exe + madvent.pas)

## Usage
Run ./advent with the --path flag and point to the directory containing door32.sys (include trailing slash and not "door32.sys"):
~~~~
./advent --path /path/to/dropfile/
~~~~
Or, you can run locally with the --local flag. This will attempt to utilize unicode/UTF-8 encoding instead of CP437:
~~~~
./advent --local
~~~~


## Notes
- BBS version ANSI files are in /art
- Placeholder WELCOME.ANS & GOODBYE.ANS art
- Requires door32.sys drop file from BBS
- CP437 only, not unicode supported
- User timeout after 1 minute no keyboard activity
- 80x25 only
- DOS verion uses a single BIN file for calendar art

## ToDo
- [-] Detect art more than 25 rows and allow user scrolling up/down
- [x] Add UTF-8 support for local terminal usage
