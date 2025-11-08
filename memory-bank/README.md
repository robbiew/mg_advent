# Memory Bank - Mistigris Advent Calendar Modernization (2025)

## Overview
This Memory Bank documents the analysis, architecture, and modernization plan for updating the Mistigris Advent Calendar BBS door program for 2025. The project is a Go-based ANSI art viewer that displays daily advent calendar art with navigation features.

## Current Architecture Analysis

### Code Structure
- **main.go**: Main application logic, navigation, initialization
- **ansiart.go**: ANSI art rendering and display functions
- **utility.go**: BBS integration, file validation, terminal utilities
- **timers.go**: Session timeout management

### Key Components
- **User Management**: BBS user data parsing from door32.sys
- **Art Display**: ANSI/CP437 art rendering with UTF-8 support
- **Navigation**: Arrow key navigation through calendar days
- **Validation**: Date and art file validation
- **Timers**: Idle and session timeout handling

### Current Limitations
- Hardcoded year logic (2023/2024 specific)
- Monolithic main.go file
- Limited error handling
- No configuration management
- Basic navigation only
- No theming or customization
- Memory inefficient for large art files

## Modernization Plan

### Phase 1: Foundation Updates
- Update Go version to 1.21+
- Update dependencies
- Implement proper error handling
- Add configuration management

### Phase 2: Code Restructuring
- Modularize code into packages
- Separate concerns (display, navigation, validation)
- Implement interfaces for extensibility
- Add comprehensive logging

### Phase 3: Feature Enhancements
- Multi-year browsing capability
- Theme system for different visual styles
- Animation support for transitions
- Enhanced navigation (jump to date, search)
- Performance optimizations

### Phase 4: 2025 Specific Updates
- Add 2025 art directory structure
- Update year detection logic
- Implement 2025-specific features
- Update documentation

## Technical Debt Identified

### High Priority
- Global variables usage
- Hardcoded paths and constants
- Lack of unit tests
- No configuration file support

### Medium Priority
- Memory usage optimization needed
- Better error recovery
- Input validation improvements

### Low Priority
- Code documentation improvements
- Performance profiling
- Accessibility features

## Dependencies Analysis

Current dependencies:
- github.com/eiannone/keyboard v0.0.0-20220611211555-0d226195f203
- golang.org/x/text v0.20.0

Potential new dependencies for modernization:
- Configuration: github.com/spf13/viper
- Logging: github.com/sirupsen/logrus
- Testing: github.com/stretchr/testify
- CLI: github.com/spf13/cobra

## Performance Considerations

### Current Issues
- Art files loaded entirely into memory
- No caching mechanism
- Synchronous file operations
- No background processing

### Optimization Opportunities
- Implement art file caching
- Lazy loading of art assets
- Asynchronous operations where possible
- Memory pooling for common operations

## Security Considerations

### Current State
- Basic input validation
- File path traversal protection needed
- No authentication beyond BBS integration

### Improvements Needed
- Path traversal protection
- Input sanitization
- Resource limits
- Safe file operations

## Testing Strategy

### Unit Tests
- Core functions (parsing, validation, rendering)
- Error conditions
- Edge cases

### Integration Tests
- Full navigation flows
- BBS integration
- Art file handling

### Performance Tests
- Memory usage under load
- Response times
- Resource utilization

## Deployment Considerations

### BBS Integration
- Maintain compatibility with existing BBS systems
- Support multiple BBS software
- Graceful degradation for missing features

### Distribution
- Single binary deployment
- Configuration file handling
- Art asset management

## Future Roadmap

### Short Term (2025)
- Complete modernization
- Multi-year support
- Theme system

### Medium Term (2026+)
- Web interface option
- Mobile app
- Advanced animations
- Social features

### Long Term
- Plugin architecture
- Multi-platform support
- Cloud integration

## Risk Assessment

### High Risk
- Breaking changes to BBS integration
- Performance degradation
- Art file compatibility issues

### Medium Risk
- Dependency update conflicts
- Code restructuring complexity
- Testing coverage gaps

### Mitigation Strategies
- Incremental changes with testing
- Backward compatibility checks
- Comprehensive testing suite
- Rollback procedures

## Success Metrics

### Technical
- Reduced memory usage by 30%
- Improved response times
- Zero critical bugs in production

### User Experience
- Faster loading times
- Enhanced navigation
- Better visual appeal

### Maintainability
- Modular code structure
- Comprehensive documentation
- Automated testing coverage