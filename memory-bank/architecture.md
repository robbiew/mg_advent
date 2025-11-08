# Architecture Documentation

## Current Architecture

### File Structure
```
mg_advent/
├── main.go           # Main application logic
├── ansiart.go        # ANSI art rendering
├── utility.go        # BBS integration and utilities
├── timers.go         # Session management
├── go.mod            # Dependencies
├── launch.sh         # Launch script
├── README.md         # Documentation
└── art/              # Art assets
    ├── 2023/         # Year-specific art
    └── 2024/         # Year-specific art
```

### Component Relationships

```
main.go (Entry Point)
├── Initializes User from BBS/dropfile
├── Sets up timers (idle/max session)
├── Validates date and art files
├── Main navigation loop
│   ├── Keyboard input handling
│   ├── State management (welcome/day/comeback)
│   └── Art display calls
└── Cleanup and exit

utility.go (BBS Integration)
├── DropFileData() - Parse door32.sys
├── Initialize() - Terminal setup
├── validateArtFiles() - File validation
└── Terminal utilities (cursor, screen)

ansiart.go (Display Engine)
├── displayAnsiFile() - Main display function
├── printUtf8() - UTF-8 rendering
├── printAnsi() - ANSI rendering
└── Metadata trimming (SAUCE)

timers.go (Session Management)
├── TimerManager struct
├── Idle timeout handling
└── Max session timeout handling
```

### Data Flow

1. **Initialization**
   - Parse command line flags
   - Read BBS dropfile or use local mode
   - Get terminal dimensions
   - Initialize timers

2. **Validation**
   - Check current date (must be December)
   - Validate art files exist
   - Check ANSI emulation support

3. **Main Loop**
   - Display welcome screen
   - Handle keyboard input
   - Navigate between days
   - Display appropriate art
   - Handle timeouts

## Proposed New Architecture

### Modular Structure
```
mg_advent/
├── cmd/
│   └── advent/           # CLI application
│       └── main.go
├── internal/
│   ├── config/           # Configuration management
│   ├── display/          # Display engine
│   ├── navigation/       # Navigation logic
│   ├── bbs/              # BBS integration
│   ├── art/              # Art management
│   ├── session/          # Session/timers
│   └── validation/       # Validation logic
├── pkg/
│   ├── ansi/             # ANSI utilities
│   └── theme/            # Theme system
├── art/                  # Art assets
├── config/               # Configuration files
├── docs/                 # Documentation
└── scripts/              # Build/deployment scripts
```

### Component Interfaces

#### Display Interface
```go
type Displayer interface {
    Display(filePath string, user User) error
    ClearScreen() error
    MoveCursor(x, y int) error
    GetDimensions() (width, height int)
}
```

#### Navigator Interface
```go
type Navigator interface {
    Navigate(direction Direction, currentState State) (newState State, artPath string, err error)
    GetAvailableYears() []int
    SetYear(year int) error
}
```

#### Art Manager Interface
```go
type ArtManager interface {
    Validate(year int) error
    GetPath(year int, day int, screen ScreenType) string
    ListYears() []int
    CacheArt(filePath string) error
}
```

### State Management

#### Application State
```go
type AppState struct {
    CurrentYear    int
    CurrentDay     int
    Screen         ScreenType
    User           User
    Config         Config
    ArtManager     ArtManager
    Navigator      Navigator
    Displayer      Displayer
}
```

#### Screen Types
```go
type ScreenType int
const (
    ScreenWelcome ScreenType = iota
    ScreenDay
    ScreenComeback
    ScreenExit
    ScreenYearSelect
)
```

### Configuration Structure

#### Config File (YAML)
```yaml
app:
  name: "Mistigris Advent Calendar"
  version: "2.0.0"
  timeout_idle: "5m"
  timeout_max: "120m"

display:
  default_theme: "classic"
  animation_enabled: true
  cache_enabled: true

bbs:
  dropfile_path: "/path/to/door32.sys"
  emulation_required: 1

art:
  base_dir: "art"
  cache_size: "100MB"
  preload_days: 7
```

## Performance Optimizations

### Memory Management
- Implement art file caching with LRU eviction
- Use memory pools for common allocations
- Stream large files instead of loading entirely
- Background preloading of adjacent days

### Rendering Optimizations
- Double buffering for smooth transitions
- Incremental rendering for animations
- GPU acceleration where possible
- Optimized ANSI sequence processing

### I/O Optimizations
- Asynchronous file operations
- File descriptor pooling
- Compressed art storage option
- CDN integration for remote assets

## Error Handling Strategy

### Error Types
```go
type AppError struct {
    Code    ErrorCode
    Message string
    Context map[string]interface{}
}

type ErrorCode int
const (
    ErrArtNotFound ErrorCode = iota
    ErrInvalidDate
    ErrBBSDisconnect
    ErrTimeout
    ErrValidation
)
```

### Error Recovery
- Graceful degradation (fallback screens)
- Automatic retry for transient errors
- User-friendly error messages
- Logging with context

## Testing Architecture

### Test Structure
```
tests/
├── unit/              # Unit tests
├── integration/       # Integration tests
├── e2e/              # End-to-end tests
├── fixtures/         # Test data
└── mocks/            # Mock implementations
```

### Test Categories
- **Unit Tests**: Individual functions and methods
- **Integration Tests**: Component interactions
- **E2E Tests**: Full user workflows
- **Performance Tests**: Load and stress testing
- **Compatibility Tests**: BBS system integration

## Security Architecture

### Input Validation
- Path traversal protection
- File type validation
- Size limits on uploads
- Sanitization of user inputs

### Access Control
- BBS authentication integration
- File permission checks
- Resource usage limits
- Audit logging

## Deployment Architecture

### Build Process
- Multi-platform builds (Linux, Windows, macOS)
- Static linking for BBS compatibility
- Version embedding
- Automated testing in CI/CD

### Distribution
- Single binary deployment
- Configuration file management
- Art asset packaging
- Update mechanism

### Monitoring
- Performance metrics
- Error tracking
- Usage analytics
- Health checks