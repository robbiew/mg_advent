unit UI;

interface

uses
  Crt;

procedure ClearScreen;
procedure CenterWrite(Y: Integer; const S: string);
function ReadKeyNonBlocking: Char;

implementation

procedure ClearScreen;
begin
  ClrScr;
end;

procedure CenterWrite(Y: Integer; const S: string);
var
  X: Integer;
begin
  X := (80 - Length(S)) div 2;
  if X < 1 then X := 1;
  GotoXY(X, Y);
  Write(S);
end;

function ReadKeyNonBlocking: Char;
begin
  if KeyPressed then
    Result := ReadKey
  else
    Result := #0;
end;

end.
