                    MISTIGRIS ADVENT CALENDAR
                    ========================

COMMAND LINE SWITCHES
--------------------

The Mistigris Advent Calendar supports the following command line switches. 
Only "-path" is required:
      
  -path <filepath>
      Path to door32.sys file
      
  -logon
      Logon mode: show current day's door, then COMEBACK.ANS and exit
      
  -debug-date <YYYY-MM-DD>
      Override date for testing
      
  -debug-disable-date
      Disable date validation (allows running outside of December)
      

EXAMPLES
--------

Run as a BBS door with door32.sys:
  advent -path /path/to/door32.sys

Run in logon mode (shows current day and exits):
  advent -logon -path /path/to/door32.sys


IMPORTANT NOTE FOR WINDOWS USERS
-------------------------------_
DO NOT RENAME THE EXECUTABLE FILE. The program must be run with the exact 
filename provided in the package to ensure proper operation.