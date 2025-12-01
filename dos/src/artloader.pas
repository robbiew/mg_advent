unit ArtLoader;

interface

uses
  SysUtils;

function ArtPathForDay(Year, Day: Integer): string;
function LoadFileToString(const FileName: string): string;

implementation

function ArtPathForDay(Year, Day: Integer): string;
var
  DayStr: string;
begin
  DayStr := Format('%.2d', [Day]);  { 01, 02, ..., 24 }
  // Adjust layout if your filenames differ
  Result := Format('art/%d/%s.ANS', [Year, DayStr]);
end;

function LoadFileToString(const FileName: string): string;
var
  F    : File;
  Buf  : AnsiString;
  Size : LongInt;
begin
  Result := '';
  if not FileExists(FileName) then
    Exit;

  AssignFile(F, FileName);
  {$I-}
  Reset(F, 1);
  {$I+}
  if IOResult <> 0 then
    Exit;

  Size := FileSize(F);
  SetLength(Buf, Size);

  BlockRead(F, Buf[1], Size);
  CloseFile(F);

  Result := string(Buf);
end;

end.
