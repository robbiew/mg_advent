unit Door32;

{
  Minimal DOOR32.SYS parser for mg_advent.

  We deliberately implement just what mg_advent needs:
    - Node number
    - Baud rate
    - User name
}

interface

uses
  SysUtils;

type
  TDoorInfo = record
    Node       : Integer;
    Baud       : LongInt;
    UserName   : string;
    DropFile   : string;
  end;

function LoadDoor32(const FileName: string; out Info: TDoorInfo): Boolean;

implementation

function LoadDoor32(const FileName: string; out Info: TDoorInfo): Boolean;
var
  F        : Text;
  Line     : string;
  LineNum  : Integer;
begin
  Result := False;
  FillChar(Info, SizeOf(Info), 0);
  Info.DropFile := FileName;

  if not FileExists(FileName) then
    Exit;

  AssignFile(F, FileName);
  {$I-}
  Reset(F);
  {$I+}
  if IOResult <> 0 then
    Exit;

  LineNum := 0;
  try
    while not Eof(F) do
    begin
      ReadLn(F, Line);
      Inc(LineNum);

      case LineNum of
        1: {Com port} ;
        2: Info.Baud   := StrToIntDef(Trim(Line), 0);
        3: Info.Node   := StrToIntDef(Trim(Line), 0);
        4: Info.UserName := Trim(Line);
      end;
    end;
  finally
    CloseFile(F);
  end;

  Result := True;
end;

end.
