Program Advent;

Uses
  D32, Dos;

Const
  MAX_YEARS = 3;
  MAX_DAYS = 25;
  CURRENT_YEAR = 2025;
  TEMP_FILE = '~TEMP.ANS';  { Temp file for extracting from DAT }
  
  { Screen indices in year DAT files }
  SCREEN_WELCOME = 0;
  SCREEN_INFOFILE = 1;
  SCREEN_MEMBERS = 2;
  SCREEN_DAY1 = 3;      { Days 1-25 are screens 3-27 }
  
  { Screen indices in COMMON.DAT }
  COMMON_FOOTER = 0;
  COMMON_GOODBYE = 1;
  COMMON_COMEBACK = 2;
  COMMON_NOTYET = 3;
  COMMON_MISSING = 4;

Type
  ScreenType = (scWelcome, scDay, scInfo, scMembers, scComeback, scNotYet);
  YearArray = Array[1..MAX_YEARS] of Integer;
  
Var
  CurrentYear: Integer;
  CurrentDay: Integer;
  CurrentScreen: ScreenType;
  AvailableYears: YearArray;
  DebugDisableDate: Boolean;  { Debug flag to disable date validation }
  DebugDay: Integer;          { Simulated day for testing (0 = use real date) }

{ Cursor control procedures }
Procedure CursorOff;
Var
  regs: registers;
Begin
  regs.ah := $01;
  regs.cx := $2000;
  Intr($10, regs);
End;

Procedure CursorOn;
Var
  regs: registers;
Begin
  regs.ah := $01;
  regs.cx := $0607;
  Intr($10, regs);
End;

{ String utility functions }
Function IntToStr(Value: Integer): String;
Var
  Temp: String;
Begin
  Str(Value, Temp);
  IntToStr := Temp;
End;

{ Simple delay function (milliseconds) }
Procedure SimpleDelay(Ms: Word);
Var
  i, j: LongInt;
Begin
  { Rough delay approximation using loops }
  { Adjust multiplier based on CPU speed if needed }
  For i := 1 To Ms Do
    For j := 1 To 1000 Do
      { Empty loop for delay }
      ;
End;

Function PadLeft(S: String; Width: Integer): String;
Begin
  While Length(S) < Width Do
    S := '0' + S;
  PadLeft := S;
End;

{ Get current advent day (1-25) or 0 if not December }
Function GetAdventDay: Integer;
Var
  Year, Month, Day, Dow: Word;
Begin
  { If debug day specified, use it }
  If DebugDay > 0 Then
  Begin
    GetAdventDay := DebugDay;
    Exit;
  End;
  
  { If debug mode, return 25 so all days are viewable }
  If DebugDisableDate Then
  Begin
    GetAdventDay := MAX_DAYS;
    Exit;
  End;
  
  GetDate(Year, Month, Day, Dow);
  If (Month = 12) And (Day >= 1) And (Day <= 25) Then
    GetAdventDay := Day
  Else
    GetAdventDay := 0;
End;

{ Check if we're in December }
Function IsDecember: Boolean;
Var
  Year, Month, Day, Dow: Word;
Begin
  { If debug mode or debug day specified, always return true }
  If DebugDisableDate Or (DebugDay > 0) Then
  Begin
    IsDecember := True;
    Exit;
  End;
  
  GetDate(Year, Month, Day, Dow);
  IsDecember := (Month = 12);
End;

{ Get DAT filename for a year }
Function GetDatFileName(Year: Integer): String;
Begin
  GetDatFileName := 'ADVENT' + IntToStr(Year Mod 100) + '.DAT';
End;


{ Extract and display screen from indexed ANS-packed DAT file }
Procedure DisplayDatScreen(DatFileName: String; ScreenNum: Integer);
Var
  DatFile, TempFile: File;
  NumScreens: LongInt;
  ScreenOffset, ScreenSize: LongInt;
  Buffer: Array[1..8192] of Byte;
  BytesRead, BytesToWrite: Word;
  TotalRead: LongInt;
Begin
  { Open DAT file }
  Assign(DatFile, DatFileName);
  {$I-}
  Reset(DatFile, 1);
  If IOResult <> 0 Then
  Begin
    SendCLS;
    SendLn('|12ERROR: Cannot open ' + DatFileName + '|CR');
    GetKey;
    Exit;
  End;
  {$I+}
  
  { Read number of screens }
  BlockRead(DatFile, NumScreens, 4, BytesRead);
  If (BytesRead <> 4) Or (ScreenNum >= NumScreens) Then
  Begin
    SendCLS;
    SendLn('|12ERROR: Invalid screen number|CR');
    Close(DatFile);
    GetKey;
    Exit;
  End;
  
  { Read index entry for this screen }
  Seek(DatFile, 4 + (ScreenNum * 8));
  BlockRead(DatFile, ScreenOffset, 4, BytesRead);
  BlockRead(DatFile, ScreenSize, 4, BytesRead);
  
  { Extract ANS data to temp file }
  { Note: Empty slots now have MISSING.ANS packed in, so no size check needed }
  Assign(TempFile, TEMP_FILE);
  {$I-}
  Rewrite(TempFile, 1);
  If IOResult <> 0 Then
  Begin
    SendCLS;
    SendLn('|12ERROR: Cannot create temp file|CR');
    Close(DatFile);
    GetKey;
    Exit;
  End;
  {$I+}
  
  { Copy screen data from DAT to temp file }
  Seek(DatFile, ScreenOffset);
  TotalRead := 0;
  While TotalRead < ScreenSize Do
  Begin
    BytesToWrite := 8192;
    If ScreenSize - TotalRead < 8192 Then
      BytesToWrite := ScreenSize - TotalRead;
    
    BlockRead(DatFile, Buffer, BytesToWrite, BytesRead);
    BlockWrite(TempFile, Buffer, BytesRead);
    TotalRead := TotalRead + BytesRead;
    
    If BytesRead = 0 Then Break;
  End;
  
  Close(DatFile);
  Close(TempFile);
  
  { Display temp file using SendFile }
  SendCLS;
  SendFile(TEMP_FILE, False);
  
  { Clean up temp file }
  Erase(TempFile);
End;

{ Display footer - extract from COMMON.DAT and show at row 25 }
Procedure DisplayFooter;
Begin
  ansi_GotoXY(1, 25);
  { For simplicity, footer could be shown via temp file too }
  { For now, skip footer to avoid complexity }
End;

{ Display welcome screen from indexed DAT }
Procedure DisplayWelcomeScreen;
Begin
  DisplayDatScreen(GetDatFileName(CurrentYear), SCREEN_WELCOME);
End;

{ Display day art from indexed DAT }
Procedure DisplayDayScreen(Year, Day: Integer);
Begin
  DisplayDatScreen(GetDatFileName(Year), SCREEN_DAY1 + Day - 1);
End;

{ Display info screen from indexed DAT }
Procedure DisplayInfoScreen;
Begin
  DisplayDatScreen(GetDatFileName(CurrentYear), SCREEN_INFOFILE);
  DisplayFooter;
End;

{ Display members screen from indexed DAT }
Procedure DisplayMembersScreen;
Begin
  DisplayDatScreen(GetDatFileName(CurrentYear), SCREEN_MEMBERS);
  DisplayFooter;
End;

{ Display comeback screen from COMMON.DAT }
Procedure DisplayComebackScreen;
Var
  Year, Month, Day, Dow: Word;
Begin
  DisplayDatScreen('COMMON.DAT', COMMON_COMEBACK);
  
  { Display message about when to come back }
  GetDate(Year, Month, Day, Dow);
  ansi_GotoXY(25, 22);
  If (Month = 12) And (Day < 25) Then
    SendLn('|15Come back tomorrow for Day ' + IntToStr(Day + 1) + '!|CR')
  Else If (Month = 12) And (Day >= 25) Then
    SendLn('|15See you next year!|CR')
  Else
    SendLn('|15Come back in December!|CR');
End;

{ Display not yet available screen from COMMON.DAT }
Procedure DisplayNotYetScreen;
Begin
  DisplayDatScreen('COMMON.DAT', COMMON_NOTYET);
  
  ansi_GotoXY(30, 24);
  SendLn('|07[Press any key]|CR');
End;

{ Get the latest available year }
Function GetLatestYear: Integer;
Begin
  GetLatestYear := AvailableYears[MAX_YEARS];
End;

{ Check if a year index is valid (1-3) }
Function IsValidYearIndex(Index: Integer): Boolean;
Begin
  IsValidYearIndex := (Index >= 1) And (Index <= MAX_YEARS);
End;

{ Select year by index (1=2023, 2=2024, 3=2025) }
Procedure SelectYearByIndex(Index: Integer);
Begin
  If IsValidYearIndex(Index) Then
  Begin
    CurrentYear := AvailableYears[Index];
    CurrentDay := GetAdventDay;
    If CurrentDay = 0 Then CurrentDay := 1;
    CurrentScreen := scWelcome;  { Show that year's welcome screen first }
  End;
End;

{ Handle navigation and render current screen }
Procedure RenderCurrentScreen;
Begin
  Case CurrentScreen of
    scWelcome:
      DisplayWelcomeScreen;
    
    scDay:
      If (CurrentDay >= 1) And (CurrentDay <= GetAdventDay) Then
        DisplayDayScreen(CurrentYear, CurrentDay)
      Else If CurrentDay = GetAdventDay + 1 Then
      Begin
        CurrentScreen := scComeback;
        DisplayComebackScreen;
      End
      Else
        DisplayDayScreen(CurrentYear, CurrentDay);
    
    scInfo:
      DisplayInfoScreen;
    
    scMembers:
      DisplayMembersScreen;
    
    scComeback:
      DisplayComebackScreen;
    
    scNotYet:
      DisplayNotYetScreen;
  End;
End;

{ Main navigation loop }
Procedure NavigationLoop;
Var
  Ch: Char;
  LastScreen: ScreenType;
  LastDay, LastYear: Integer;
  AdventDay: Integer;
Begin
  LastScreen := scWelcome;
  LastDay := -1;
  LastYear := -1;
  
  { Initial display }
  RenderCurrentScreen;
  LastScreen := CurrentScreen;
  LastDay := CurrentDay;
  LastYear := CurrentYear;
  
  Repeat
    { Only redraw if something changed }
    If (CurrentScreen <> LastScreen) Or 
       (CurrentDay <> LastDay) Or
       (CurrentYear <> LastYear) Then
    Begin
      RenderCurrentScreen;
      LastScreen := CurrentScreen;
      LastDay := CurrentDay;
      LastYear := CurrentYear;
    End;
    
    { Wait for user input }
    Ch := GetKey;
    AdventDay := GetAdventDay;
    
    { Handle arrow key navigation }
    If IsArrowKey Then
    Begin
      Case Ch of
        #75: { Left Arrow }
          If CurrentScreen = scDay Then
          Begin
            If CurrentDay > 1 Then
              Dec(CurrentDay)
            Else
              CurrentScreen := scWelcome;
          End
          Else If (CurrentScreen = scInfo) Or (CurrentScreen = scMembers) Then
            CurrentScreen := scWelcome;
        
        #77: { Right Arrow }
          If CurrentScreen = scWelcome Then
          Begin
            CurrentDay := AdventDay;
            If CurrentDay = 0 Then CurrentDay := 1;
            CurrentScreen := scDay;
          End
          Else If CurrentScreen = scDay Then
          Begin
            If CurrentDay < AdventDay Then
              Inc(CurrentDay)
            Else If CurrentDay = AdventDay Then
            Begin
              CurrentDay := AdventDay + 1;
              CurrentScreen := scComeback;
            End;
          End;
      End;
    End
    { Handle character input }
    Else
    Begin
      { Convert to uppercase for comparison }
      If (Ch >= 'a') And (Ch <= 'z') Then
        Ch := Chr(Ord(Ch) - 32);
      
      Case Ch of
        'Q', #27: { Q or ESC }
        Begin
          If CurrentScreen = scWelcome Then
          Begin
            If CurrentYear = GetLatestYear Then
            Begin
              { Exit program from latest year's welcome }
              DisplayDatScreen('COMMON.DAT', COMMON_GOODBYE);
              SimpleDelay(2000);
              Exit;
            End
            Else
            Begin
              { Return to latest year }
              CurrentYear := GetLatestYear;
            End;
          End
          Else
          Begin
            { Return to current year's welcome screen }
            CurrentScreen := scWelcome;
          End;
        End;
        
        'I': { Info screen }
          If CurrentScreen = scWelcome Then
            CurrentScreen := scInfo;
        
        'M': { Members screen }
          If CurrentScreen = scWelcome Then
            CurrentScreen := scMembers;
        
        '1': { Select 2023 }
          If CurrentScreen = scWelcome Then
            SelectYearByIndex(1);
        
        '2': { Select 2024 }
          If CurrentScreen = scWelcome Then
            SelectYearByIndex(2);
        
        '3': { Select 2025 }
          If CurrentScreen = scWelcome Then
            SelectYearByIndex(3);
        
        #13: { Enter key on welcome = go to current day }
          If CurrentScreen = scWelcome Then
          Begin
            CurrentDay := AdventDay;
            If CurrentDay = 0 Then CurrentDay := 1;
            CurrentScreen := scDay;
          End;
      End;
    End;
    
  Until False;
End;

{ Check for debug flags in command line }
Function HasDebugFlag(Flag: String): Boolean;
Var
  I: Integer;
  Param: String;
Begin
  HasDebugFlag := False;
  For I := 1 To ParamCount Do
  Begin
    Param := ParamStr(I);
    { Convert to lowercase for comparison }
    If (Param = Flag) Or (Param = '-' + Flag) Then
    Begin
      HasDebugFlag := True;
      Exit;
    End;
  End;
End;

{ Get debug day parameter value (e.g., -debug-day=15) }
Function GetDebugDayParam: Integer;
Var
  I: Integer;
  Param: String;
  EqualsPos: Integer;
  DayStr: String;
  DayVal, Code: Integer;
Begin
  GetDebugDayParam := 0;
  For I := 1 To ParamCount Do
  Begin
    Param := ParamStr(I);
    If (Pos('-debug-day=', Param) = 1) Or (Pos('debug-day=', Param) = 1) Then
    Begin
      EqualsPos := Pos('=', Param);
      If EqualsPos > 0 Then
      Begin
        DayStr := Copy(Param, EqualsPos + 1, Length(Param) - EqualsPos);
        Val(DayStr, DayVal, Code);
        If (Code = 0) And (DayVal >= 1) And (DayVal <= 25) Then
        Begin
          GetDebugDayParam := DayVal;
          Exit;
        End;
      End;
    End;
  End;
End;

{ Main program }
Var
  Res: Byte;
  I: Integer;
Begin
  { Initialize debug flags }
  DebugDisableDate := False;
  DebugDay := 0;
  
  { Check command line parameters }
  If ParamCount < 1 Then
  Begin
    WriteLn;
    WriteLn('MiSTiGRiS Advent Calendar Door v2.0');
    WriteLn('Usage: ADVENT <path to dropfile> [options]');
    WriteLn('       Supports: DORINFOX.DEF, DOOR.SYS, CHAIN.TXT');
    WriteLn;
    WriteLn('Options:');
    WriteLn('  -debug-disable-date     Disable December restriction (test anytime)');
    WriteLn('  -debug-day=N            Simulate specific day (1-25)');
    WriteLn;
    WriteLn('Features:');
    WriteLn('  - Multi-year support (2023, 2024, 2025)');
    WriteLn('  - Arrow keys: Navigate between days');
    WriteLn('  - Number keys 1-3: Jump to different years');
    WriteLn('  - I: View info file');
    WriteLn('  - M: View members list');
    WriteLn('  - Q/ESC: Return to welcome or exit');
    WriteLn;
    Halt;
  End;
  
  { Parse debug flags }
  DebugDisableDate := HasDebugFlag('debug-disable-date');
  DebugDay := GetDebugDayParam;
  
  If DebugDay > 0 Then
    SendLn('|07DEBUG: Simulating day ' + IntToStr(DebugDay) + '|CR');
  
  { Initialize available years }
  AvailableYears[1] := 2023;
  AvailableYears[2] := 2024;
  AvailableYears[3] := 2025;
  
  { Open BBS connection }
  Res := OpenDOOR(ParamStr(1));
  
  Case Res of
    1: WriteLn('ERROR: Drop file not found.');
    2: WriteLn('ERROR: Error reading drop file.');
    3: WriteLn('ERROR: Unknown drop file type.');
    4: WriteLn('ERROR: Unable to open communications port.');
  End;
  
  If Res <> 0 Then Halt;
  
  { Initialize state }
  CurrentYear := CURRENT_YEAR;
  CurrentDay := GetAdventDay;
  CurrentScreen := scWelcome;
  
  { Check if we're in December }
  If Not IsDecember Then
  Begin
    CursorOff;
    DisplayNotYetScreen;
    GetKey;
    CursorOn;
    CloseDOOR;
    Halt;
  End;
  
  { Show version on screen so we know we're running latest code }
  SendCLS;
  SendLn('|15** ADVENT v2.0 (Enhanced) - Build 2025-11-25 **|CR');
  SendLn('|CR');
  SimpleDelay(1000);
  
  { Start main loop }
  CursorOff;
  NavigationLoop;
  CursorOn;
  
  CloseDOOR;
End.
