MiSTiGRiS ADVENT CALENDAR BBS DOOR
Programmed by J0hnny A1pha
https://github.com/robbiew/mg_advent
====================================

COMMAND LINE SWITCHES
---------------------
The Mistigris Advent Calendar BBS Door supports the following command line switches. 
Only "-path" is required:
      
  -path <filepath>
      Path to door32.sys file
      
  -logon
      Logon mode: show current day's door, then COMEBACK.ANS and exit
      
  -debug-date <YYYY-MM-DD>
      Override date for testing
      
  -debug-disable-date
      Disable date validation (allows running outside of December)

  -version
      Show version and author information


EXAMPLES
--------
Run as a BBS door with door32.sys:
  advent -path /path/to/door32.sys

Run in logon mode (shows current day and exits):
  advent -logon -path /path/to/door32.sys


IMPORTANT NOTE FOR WINDOWS 7 USERS
----------------------------------
DO NOT RENAME THE EXECUTABLE FILE. The program must be run with the exact 
filename provided in the package to ensure proper operation.


IMPORTANT NOTE FOR LINUX USERS
------------------------------
If possible, build your executable from the Go source to avoid any version 
incompatibilities (like GLIB-C):

https://github.com/robbiew/mg_advent