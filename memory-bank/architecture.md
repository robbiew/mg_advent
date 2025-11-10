# Architecture Documentation

## Current Architecture

### File Structure
````markdown
# Architecture Documentation

## ✅ IMPLEMENTED ARCHITECTURE (2025)

### Production File Structure
```
mg_advent/
├── cmd/advent/           # Application entry point
│   └── main.go          # CLI setup, component wiring, main loop
├── internal/            # Private packages (not importable externally) 
│   ├── art/             # Art asset management & path resolution
│   │   └── manager.go
│   ├── bbs/             # BBS integration & cross-platform I/O
│   │   ├── connector.go        # Main BBS connection logic
│   │   ├── windows_socket.go   # Windows socket inheritance
│   │   ├── linux_socket.go    # Linux STDIN/STDOUT handling
│   │   └── terminal_detect.go # Terminal size detection
│   ├── display/         # Display engine with dual-mode output
│   │   ├── engine.go    # Main display logic & caching
│   │   ├── types.go     # User struct and configuration
│   │   └── theme.go     # Theme system foundation
│   ├── embedded/        # Embedded filesystem for art assets
│   │   └── embedded.go  # embed.FS for all art files
│   ├── input/           # Input handling & keyboard interface  
│   │   └── input.go     # Cross-platform key reading
│   ├── navigation/      # Navigation state & logic
│   │   └── navigation.go # Screen transitions & year browsing
│   ├── session/         # Session & timeout management
│   │   └── manager.go   # Idle/max timers with callbacks
│   └── validation/      # Validation logic
│       └── validator.go # Date, art, terminal validation
├── scripts/             # Build & deployment automation
│   ├── build.sh        # Cross-platform build script
│   └── release.sh      # Git tagging and release automation
└── embedded/art/        # Art assets (compiled into binary)
    ├── common/         # Year-independent screens
    ├── 2023/, 2024/, 2025/  # Year-specific daily art
```

### ✅ Component Relationships (IMPLEMENTED)

```
cmd/advent/main.go (Entry Point & Orchestration)
├── CLI flag parsing & debug mode setup
├── Component initialization with dependency injection
├── BBS connection establishment (Windows socket vs Linux STDIO)
├── Display engine configuration (CP437 vs UTF-8 mode)
├── Session manager with timeout callbacks
├── Main application loop with input handling
└── Graceful cleanup and exit

internal/bbs/ (Cross-Platform BBS Integration)
├── connector.go: BBSConnection with ConnectionType detection
├── windows_socket.go: Socket inheritance from parent BBS process
├── linux_socket.go: STDIN/STDOUT handling for Unix systems
├── terminal_detect.go: ANSI query-based size detection
└── ParseDoor32(): door32.sys parsing with validation

internal/display/ (Dual-Mode Display Engine)  
├── engine.go: DisplayEngine with caching & scroll state
├── DualModeWriter: CP437→UTF-8 conversion for local terminals
├── types.go: User struct & DisplayConfig with performance settings
├── theme.go: Theme system foundation for future customization
└── Embedded FS integration for art asset loading

internal/navigation/ (State Management & Year Browsing)
├── State struct: CurrentYear, CurrentDay, Screen, MaxDay
├── Navigator: Multi-year support with numeric key selection
├── Direction enum: Left/Right/PageUp/PageDown/Home/End
└── Screen transitions: Welcome→Day→Comeback with validation

internal/session/ (Timeout Management)
├── Manager: Idle timer (5min) + Max timer (120min)
├── ResetIdleTimer(): Called on user input
├── Timeout callbacks: Graceful exit with cleanup
└── Start()/Stop(): Goroutine lifecycle management

internal/art/ (Asset Management)
├── Manager: Path resolution for year/day/screen combinations  
├── Embedded FS integration: All art compiled into binary
├── GetPath(): Smart fallback to MISSING.ANS for missing files
└── Year validation: Ensures art directories exist

internal/validation/ (Comprehensive Validation)
├── ValidateDate(): December-only restriction (debug override)
├── ValidateArtFiles(): Required screens existence check
├── ValidateTerminalSize(): Minimum size requirements
├── ValidateEmulation(): ANSI support verification
└── Structured error reporting with context
```

### ✅ Data Flow (IMPLEMENTED)

1. **✅ Initialization Pipeline**
   - Parse CLI flags (--local, --debug-date, --path, etc.)
   - Initialize components with embedded.ArtFS
   - Detect display mode (CP437Raw for BBS, CP437 for local)
   - Create BBS connection (socket inheritance or STDIN/STDOUT)
   - Configure DisplayEngine with performance settings
   - Parse user info from door32.sys or use defaults
   - Detect terminal size via BBS query or term.GetSize()

2. **✅ Validation Pipeline**
   - Date validation (December check with debug override)
   - Art file validation (required screens existence)
   - Terminal size validation (minimum requirements)
   - ANSI emulation validation (compatibility check)
   - BBS connection validation (socket vs stdio)

3. **✅ Main Application Loop**
   - Get initial navigation state from Navigator
   - Start session manager with timeout callbacks
   - Open input handler (BBS connection or keyboard)
   - Display art based on current state (Welcome/Day/Comeback)
   - Read user input with proper BBS handling
   - Reset idle timer on each input
   - Navigate via Direction enum (arrow keys, page keys)
   - Handle year selection via numeric keys (1-9)
   - Handle quit/back navigation (Q/ESC)
   - Scroll handling for long art (if CanScrollUp/Down)
   - Graceful cleanup on exit or timeout

## ✅ PRODUCTION ARCHITECTURE (IMPLEMENTED)

### ✅ Design Decisions & Rationale

**✅ internal/ vs pkg/ Choice:**
- Used `internal/` exclusively - prevents external imports, appropriate for BBS door
- No `pkg/` needed - no reusable libraries for external projects
- Embedded assets - eliminates need for external `art/` directory

**✅ Configuration Strategy:**  
- CLI flags with hardcoded defaults - simpler than config files for BBS environment
- Debug flags for development - `--debug-date`, `--debug-disable-date`, etc.
- BBS-first design - optimized for door32.sys dropfile integration

**✅ Cross-Platform Approach:**
- Platform-specific files: `windows_socket.go`, `linux_socket.go` 
- Runtime detection of BBS connection type (socket vs stdio)
- Single binary deployment with embedded assets

### ✅ Implemented Component Interfaces

#### ✅ DisplayEngine (Concrete Implementation)
```go
type DisplayEngine struct {
    config      DisplayConfig
    artFS       fs.FS
    bbsConn     *bbs.BBSConnection
    cache       map[string][]string
    scrollState ScrollState
}

// Key methods implemented:
func (de *DisplayEngine) Display(filePath string, user User) error
func (de *DisplayEngine) ClearScreen() 
func (de *DisplayEngine) SetBBSConnection(conn *bbs.BBSConnection)
func (de *DisplayEngine) GetScrollState() ScrollState
func (de *DisplayEngine) ScrollUp() / ScrollDown()
```

#### ✅ Navigator (Concrete Implementation)  
```go
type Navigator struct {
    artFS   fs.FS
    artDir  string
}

// Key methods implemented:
func (n *Navigator) Navigate(direction Direction, state State) (State, string, error)
func (n *Navigator) GetInitialState() (State, error) 
func (n *Navigator) SelectYearByIndex(index int, state State) (State, string, error)
func (n *Navigator) GetAvailableYears() []int
```

#### ✅ Art Manager (Concrete Implementation)
```go
type Manager struct {
    artFS  fs.FS
    artDir string
}

// Key methods implemented:
func (m *Manager) GetPath(year int, day int, screen string) string
func (m *Manager) ValidateYear(year int) error  
// Caching handled by DisplayEngine, not ArtManager
```

**Design Note**: Chose concrete implementations over interfaces for simplicity - appropriate for single-purpose BBS door application. Interfaces can be extracted later if needed for testing or extensibility.

### ✅ Implemented State Management

#### ✅ Navigation State (Production Implementation)
```go
type State struct {
    CurrentYear int        // Active year (2023, 2024, 2025)
    CurrentDay  int        // Day of month (1-25) 
    Screen      ScreenType // Current screen being displayed
    MaxDay      int        // Maximum accessible day (based on current date)
}
```

#### ✅ Screen Types (Production Implementation)
```go
type ScreenType int
const (
    ScreenWelcome  ScreenType = iota // Welcome screen with year selection
    ScreenDay                       // Daily art display  
    ScreenComeback                  // "Come back later" for future dates
    ScreenExit                      // Goodbye screen (brief display)
)
```

#### ✅ User State (Display Package)
```go  
type User struct {
    Alias     string        // BBS username
    TimeLeft  time.Duration // Session time remaining
    Emulation int           // Terminal emulation type
    NodeNum   int           // BBS node number
    W, H      int           // Terminal width/height
    ModalW, ModalH int      // Modal dialog dimensions
}
```

**Design Choice**: State management is distributed across components rather than centralized - each component manages its own state while main.go coordinates transitions. This reduces complexity and coupling in a single-user BBS door application.

### ✅ Configuration Strategy (CLI-Based Implementation)

#### ✅ Hard-Coded Sensible Defaults (BBS Door Philosophy)
```go
// Session timeouts
idleTimeout := 5 * time.Minute   // 5 minute idle timeout
maxTimeout := 120 * time.Minute  // 2 hour maximum session

// Display configuration
config := DisplayConfig{
    Mode:   displayMode,    // CP437Raw for BBS, CP437 for local
    Width:  80,             // Default BBS width
    Height: 25,             // Default BBS height  
    Theme:  "classic",      // Single theme for now
    Performance: PerformanceConfig{
        CacheEnabled: true,
        CacheSizeMB:  50,   // 50MB cache limit
        PreloadLines: 100,  // Lines to preload for scrolling
    },
    Columns: ColumnConfig{
        Handle80ColumnIssue: true,    // Handle 80-column wrapping
        AutoDetectWidth:     true,    // Detect actual width
    },
}
```

#### ✅ CLI Flags for Overrides (Development & Debug)
```bash
--local                    # UTF-8 mode for local terminal testing
--debug-disable-date      # Skip December restriction  
--debug-date=YYYY-MM-DD   # Override current date
--debug-disable-art       # Skip art file validation
--path=/path/to/door32.sys # BBS dropfile location
--socket-host=127.0.0.1   # BBS server IP (Windows)
--debug                   # Enable debug logging
```

**Design Rationale**: BBS doors traditionally use simple configurations. CLI flags provide flexibility for development while maintaining production simplicity. No config files needed - reduces complexity and failure points.

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