# Display System Enhancements

## Overview
This document details the specific enhancements needed for the ANSI art display system to address scrolling, 80-column handling, and UTF-8/CP437 mode separation.

## Current Issues Analysis

### 1. Vertical Scrolling Limitation
**Problem**: Art files taller than the terminal height are truncated
**Current Behavior**: Only renders up to `terminalHeight` lines
**Impact**: Users cannot see complete tall artwork

**Code Location**: `ansiart.go` lines 120-146 (printUtf8) and 148-168 (printAnsi)

### 2. 80-Column Line Break Issue
**Problem**: Exactly 80-column wide art gets unwanted line breaks
**Root Cause**: Terminal automatically adds line breaks when character is in 80th column
**Impact**: Art formatting breaks on standard 80-column displays

**Code Location**: `ansiart.go` rendering functions

### 3. UTF-8/CP437 Mode Separation
**Problem**: "-local" switch logic not clearly separated
**Current Behavior**: Global `localDisplay` variable controls mode
**Impact**: Tight coupling, difficult to test and maintain

## Enhancement Specifications

### Vertical Scrolling Implementation

#### Requirements
- Detect when art height exceeds terminal height
- Implement smooth scrolling with keyboard controls
- Show scroll indicators (position, total lines)
- Maintain ANSI escape sequence integrity during scrolling

#### Technical Approach
```go
type ScrollState struct {
    currentLine int
    totalLines int
    visibleLines int
    canScrollUp bool
    canScrollDown bool
}

func (ds *DisplaySystem) renderWithScrolling(content string, terminalHeight int) error {
    lines := strings.Split(content, "\r\n")
    totalLines := len(lines)

    if totalLines <= terminalHeight {
        // Render normally
        return ds.renderNormal(lines, terminalHeight)
    }

    // Implement scrolling logic
    scrollState := &ScrollState{
        totalLines: totalLines,
        visibleLines: terminalHeight,
        currentLine: 0,
    }

    return ds.renderScrollable(lines, scrollState)
}
```

#### User Interface
- **Page Up/Down**: Scroll by full screen
- **Arrow Up/Down**: Scroll by single line
- **Home/End**: Jump to top/bottom
- **Status Line**: Show "Line X of Y" indicator

### 80-Column Handling

#### Problem Analysis
When terminal width is exactly 80 columns, placing a character in the 80th column causes automatic line wrapping. This breaks ANSI art that is designed to be exactly 80 columns wide.

#### Solution Approach
```go
func handle80ColumnArt(line string, terminalWidth int) string {
    if terminalWidth == 80 && len(line) == 80 {
        // Remove the last character or handle specially
        // Option 1: Truncate to 79 columns
        // Option 2: Use terminal control to prevent wrapping
        // Option 3: Detect and adjust rendering
    }
    return line
}
```

#### Implementation Options
1. **Detection-Based**: Detect exactly 80-column art and adjust
2. **Terminal Control**: Use ANSI codes to control wrapping
3. **Width Adjustment**: Render at 79 columns for 80-column terminals

### UTF-8/CP437 Mode Enhancement

#### Current Architecture Issues
- Global `localDisplay` variable
- Mixed logic in display functions
- Difficult to test different modes

#### Proposed Architecture
```go
type DisplayMode int
const (
    ModeCP437 DisplayMode = iota
    ModeUTF8
)

type DisplayConfig struct {
    Mode DisplayMode
    Width int
    Height int
    EnableScrolling bool
    Handle80Columns bool
}

type DisplayEngine struct {
    config DisplayConfig
    cache map[string][]string // Cached processed lines
}

func (de *DisplayEngine) Render(filePath string) error {
    content, err := de.loadAndProcess(filePath)
    if err != nil {
        return err
    }

    switch de.config.Mode {
    case ModeCP437:
        return de.renderCP437(content)
    case ModeUTF8:
        return de.renderUTF8(content)
    default:
        return fmt.Errorf("unsupported display mode")
    }
}
```

## Implementation Plan

### Phase 1: Core Display Refactoring
1. Create `DisplayEngine` struct
2. Separate CP437 and UTF-8 rendering logic
3. Implement configuration-based initialization
4. Add basic scrolling framework

### Phase 2: Scrolling Implementation
1. Add scroll state management
2. Implement keyboard scroll controls
3. Add scroll indicators
4. Handle ANSI sequences in scrolled content

### Phase 3: 80-Column Fix
1. Add width detection logic
2. Implement 80-column handling
3. Test with various art files
4. Ensure backward compatibility

### Phase 4: Testing and Optimization
1. Unit tests for all display modes
2. Integration tests with real art files
3. Performance testing
4. Memory usage optimization

## Testing Strategy

### Unit Tests
```go
func Test80ColumnHandling(t *testing.T) {
    tests := []struct {
        name string
        input string
        terminalWidth int
        expectedOutput string
    }{
        {"Exact 80 columns", strings.Repeat("X", 80), 80, strings.Repeat("X", 79)},
        {"Under 80 columns", strings.Repeat("X", 70), 80, strings.Repeat("X", 70)},
        {"Over 80 columns", strings.Repeat("X", 90), 80, strings.Repeat("X", 80)},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := handle80ColumnArt(tt.input, tt.terminalWidth)
            assert.Equal(t, tt.expectedOutput, result)
        })
    }
}
```

### Integration Tests
- Test scrolling with real ANSI art files
- Verify 80-column handling across different terminal widths
- Test mode switching (CP437 ↔ UTF-8)
- Performance benchmarks

### Compatibility Tests
- Test with existing art files
- Verify BBS compatibility
- Check terminal emulator compatibility
- Validate with different screen sizes

## Performance Considerations

### Memory Usage
- Cache processed art lines to avoid re-processing
- Implement LRU cache with size limits
- Stream processing for very large files

### Rendering Speed
- Minimize ANSI sequence processing overhead
- Optimize scroll calculations
- Use buffered output where possible

### CPU Usage
- Background processing for art preparation
- Lazy loading of scrollable content
- Efficient string operations

## Error Handling

### Display Errors
- File not found: Show user-friendly message
- Corrupted art: Fallback to text display
- Terminal issues: Graceful degradation
- Scroll errors: Reset to safe state

### Recovery Mechanisms
- Automatic retry for transient failures
- Fallback display modes
- Error state indicators
- User recovery options

## Configuration

### Display Configuration Schema
```yaml
display:
  mode: "cp437"  # cp437 or utf8
  scrolling:
    enabled: true
    indicators: true
    keyboard_shortcuts: true
  columns:
    handle_80_column_issue: true
    auto_detect_width: true
  performance:
    cache_enabled: true
    cache_size_mb: 50
    preload_lines: 100
```

### Runtime Configuration
- Allow mode switching without restart
- Dynamic width/height adjustment
- Configuration reloading
- User preference persistence

## Backward Compatibility

### Existing Features
- Maintain all current display functionality
- Preserve existing command-line options
- Keep BBS integration intact
- Support legacy art files

### Migration Path
- Automatic detection of display capabilities
- Graceful fallback for unsupported features
- Configuration migration tools
- Documentation updates

## Success Criteria

### Functional
- Vertical scrolling works for tall art
- 80-column art displays correctly
- UTF-8 and CP437 modes work properly
- All existing functionality preserved

### Performance
- No significant performance regression
- Memory usage stays within limits
- Smooth scrolling experience
- Fast mode switching

### Quality
- Comprehensive test coverage
- Clean, maintainable code
- Good error handling
- Clear documentation