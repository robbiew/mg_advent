unit Screens;

interface

uses
  Door32, ArtLoader, UI, Config;

procedure ShowWelcome(const Info: TDoorInfo);
procedure RunAdventBrowser(const Info: TDoorInfo);

implementation

uses
  SysUtils, Crt;

procedure ShowWelcome(const Info: TDoorInfo);
begin
  ClearScreen;
  CenterWrite(2, ProgramName + ' ' + ProgramVersion);
  CenterWrite(4, 'Welcome, ' + Info.UserName);
  CenterWrite(6, 'Loading MiSTiGRiS Advent...');

  CenterWrite(10, 'Use arrow keys to browse,');
  CenterWrite(11, '1/2/3 to change year, I=Info, M=Members, Q=Quit.');

  Delay(1200);
end;

procedure DrawDay(Year, Day: Integer);
var
  Path: string;
  Art : string;
begin
  ClearScreen;
  Path := ArtPathForDay(Year, Day);

  Art := LoadFileToString(Path);
  if Art = '' then
  begin
    CenterWrite(2, Format('No art found for %d-%.2d', [Year, Day]));
  end
  else
  begin
    { Very naive: just dump the ANSI to stdout }
    Write(Art);
  end;

  GotoXY(1, 25);
  Write(Format('[Year: %d] [Day: %d]  Arrows=Nav  1-3=Year  I=Info  M=Members  Q=Quit',
    [Year, Day]));
end;

procedure ShowInfoScreen;
begin
  ClearScreen;
  { Later: load art/common/INFO.ANS instead of text }
  CenterWrite(2, 'MiSTiGRiS Advent - Info');
  CenterWrite(4, '(INFO.ANS placeholder)');
  CenterWrite(6, 'Press any key to return.');
  ReadKey;
end;

procedure ShowMembersScreen;
begin
  ClearScreen;
  { Later: load art/common/MEMBERS.ANS instead of text }
  CenterWrite(2, 'MiSTiGRiS Advent - Members');
  CenterWrite(4, '(MEMBERS.ANS placeholder)');
  CenterWrite(6, 'Press any key to return.');
  ReadKey;
end;

procedure RunAdventBrowser(const Info: TDoorInfo);
var
  Year, Day : Integer;
  Ch, Ch2   : Char;
begin
  Year := YearMax;
  Day  := 1;

  DrawDay(Year, Day);

  repeat
    Ch := ReadKeyNonBlocking;
    if Ch = #0 then
    begin
      Delay(10);
      Continue;
    end;

    case Ch of
      #0: begin
            { extended key, need to read second byte }
            Ch2 := ReadKey;
            case Ch2 of
              #72: if Day > 1 then Dec(Day);          { Up }
              #80: if Day < 24 then Inc(Day);         { Down }
              #75: if Day > 1 then Dec(Day);          { Left }
              #77: if Day < 24 then Inc(Day);         { Right }
            end;
            DrawDay(Year, Day);
          end;

      '1': begin Year := 2023; Day := 1; DrawDay(Year, Day); end;
      '2': begin Year := 2024; Day := 1; DrawDay(Year, Day); end;
      '3': begin Year := 2025; Day := 1; DrawDay(Year, Day); end;

      'i', 'I': begin
                  ShowInfoScreen;
                  DrawDay(Year, Day);
                end;
      'm', 'M': begin
                  ShowMembersScreen;
                  DrawDay(Year, Day);
                end;

      'q', 'Q', #27: Exit;
    end;

  until False;
end;

end.
