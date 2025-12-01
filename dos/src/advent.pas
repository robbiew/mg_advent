program Advent;

{$APPTYPE CONSOLE}

uses
  SysUtils, Crt,
  Config, Door32, Screens, UI;

var
  DoorInfo : TDoorInfo;
  DoorPath : string;
  Ok       : Boolean;

begin
  { Determine DOOR32.SYS path }
  if ParamCount >= 1 then
    DoorPath := ParamStr(1)
  else
    DoorPath := DefaultDoor32Path;

  if not LoadDoor32(DoorPath, DoorInfo) then
  begin
    { Fallback: local mode / no dropfile }
    FillChar(DoorInfo, SizeOf(DoorInfo), 0);
    DoorInfo.UserName := 'Local User';
    DoorInfo.DropFile := '';
  end;

  { You can later add a /LOGON or -L flag here for logon-mode behavior }

  ShowWelcome(DoorInfo);
  RunAdventBrowser(DoorInfo);

  ClearScreen;
  CenterWrite(12, 'Thank you for visiting the MiSTiGRiS Advent Calendar!');
  Delay(800);
end.
