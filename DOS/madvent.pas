program madvent;

uses crt, dos;

const
  YEAR = 2024;
  SCREEN_SIZE = 4000;

procedure wait_retrace;
begin
  while (Port[$3DA] and 8) <> 0 do;
  while (Port[$3DA] and 8) = 0 do;
end;

function advent_day: Integer;
var
  Year, Month, Day, DayOfWeek: Word;
begin
  GetDate(Year, Month, Day, DayOfWeek);
  if ((Year = YEAR) and (Month < 12)) or (Year < YEAR) then
    advent_day := 25
  else if (Year > YEAR) or (Day > 25) then
    advent_day := 25
  else
    advent_day := Day;
end;

procedure display_day(day: Integer);
var
  screen: array [1..SCREEN_SIZE] of Byte;
  f: file;
  numRead: Word;
  offset: LongInt;  { Use LongInt to handle larger values }
begin
  Assign(f, 'ADVENT.DAT');
  {$I-}  { Disable automatic runtime error handling }
  Reset(f, 1);
  if IOResult <> 0 then
  begin
    WriteLn('Error opening file.');
    Exit;
  end;

  offset := LongInt(day - 1) * LongInt(SCREEN_SIZE);
  Seek(f, offset);
  if IOResult <> 0 then
  begin
    WriteLn('Error seeking in file. Offset: ', offset);
    Close(f);
    Exit;
  end;

  BlockRead(f, screen, SCREEN_SIZE, numRead);
  if (IOResult <> 0) or (numRead <> SCREEN_SIZE) then
  begin
    WriteLn('Error reading from file. NumRead: ', numRead);
    Close(f);
    Exit;
  end;

  Close(f);
  {$I+}  { Re-enable automatic runtime error handling }

  ClrScr;  { Clear the screen before displaying a new image }
  Move(screen, Mem[$B800:0000], SCREEN_SIZE);
end;


procedure CursorOff;
var
  regs: registers;
begin
  regs.ah := $01;
  regs.cx := $2000;
  Intr($10, regs);
end;

procedure CursorOn;
var
  regs: registers;
begin
  regs.ah := $01;
  regs.cx := $0607;
  Intr($10, regs);
end;

procedure display_advent(current_day: Integer);
var
  max_day, new_day: Integer;
  key: Char;
begin
  max_day := current_day;
  CursorOff;
  ClrScr;
  display_day(current_day);  { Display the current day's art immediately }

  while True do
  begin
    new_day := advent_day;  { Check the new day }
    if new_day > max_day then
    begin
      current_day := new_day;
      display_day(current_day);
      max_day := new_day;
    end;

    if KeyPressed then
    begin
      key := ReadKey;
      case key of
        #27:  { ESC key }
          begin
          CursorOn;
            Exit;
          end;
        #77:  { Right arrow key }
          begin
            if current_day < max_day then
            begin
              Inc(current_day);
              display_day(current_day);
            end;
          end;
        #75:  { Left arrow key }
          begin
            if current_day > 1 then
            begin
              Dec(current_day);
              display_day(current_day);
            end;
          end;
      end;
    end;
    wait_retrace;
  end;
end;

var
  current_day: Integer;

begin
  current_day := advent_day;
  if current_day = 0 then
  begin
    ClrScr;
    WriteLn('Come back on December 1st, ', YEAR, '!');
    ReadKey;
  end
  else
  begin
    display_advent(current_day);
    WriteLn;
    WriteLn('Back to the void (MS-DOS)...');
  end;
end.
