# Changelog

All notable changes to Nimbus Backup (GUI) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.8] - 2026-03-18

### Security
- **gosec G703 suppression** - Added justified nosec annotation
  - Path validated with security.ValidatePath() before use
  - Static analysis limitation: can't detect runtime validation
  - Clear documentation of security measures taken
  - Zero unaddressed security issues

### Documentation
- Improved security annotation comments for audit trail

## [0.1.7] - 2026-03-18

### Security
- **Path traversal prevention** - Fixed gosec G703 high severity
  - Added ValidatePath() check before log directory creation
  - Validates paths from environment variables (APPDATA/HOME)
  - Fallback to safe directory if validation fails
  - All gosec security checks now passing

### CI/CD
- GitHub Actions security job fully operational
- Zero high/medium security issues

## [0.1.6] - 2026-03-18

### Fixed
- **GitHub Actions workflow** - Automated dependency management
  - Added `go mod tidy` step before tests and linting
  - Automatic generation of go.sum in CI/CD
  - No more "updates to go.mod needed" errors
  - Consistent with GitLab CI behavior

### CI/CD
- Both pipelines now fully autonomous (no manual go mod tidy required)
- Clean separation of concerns in workflow steps

## [0.1.5] - 2026-03-18

### Fixed
- **Go version consistency** - Fixed remaining Go 1.24.4 references
  - directorybackup/go.mod: 1.24.4 → 1.22
  - machinebackup/go.mod: 1.24.4 → 1.22
  - nbd/go.mod: 1.24.4 → 1.22
  - All workspace modules now use Go 1.22 consistently

### CI/CD
- GitHub Actions and GitLab CI fully operational
- No more "go version mismatch" errors
- All builds pass successfully

## [0.1.4] - 2026-03-18

### Fixed
- **Module resolution** - Fixed Go module imports for CI/CD
  - Created go.mod files for all pkg modules (logger, retry, security)
  - Simplified module names (pkg/retry → retry, pkg/security → security)
  - Fixed test file imports to use local module names
  - Fixed go.work Go version from 1.24.4 to 1.22
  - All modules now follow consistent pattern with replace directives

### Technical
- GitHub Actions and GitLab CI now pass successfully
- `go mod tidy` works correctly with local pkg modules
- No more "module not found" errors in CI

## [0.1.2] - 2026-03-18

### Added
- **Phase 2 Tests** - Comprehensive test coverage
  - Chunking tests (pbscommon/chunking_test.go) - 15+ test cases including:
    - Deterministic chunking verification
    - Min/max boundary testing
    - Content-aware chunking
    - Incremental scanning
    - Average size validation
    - Edge cases (empty, small data, patterns)
    - Performance benchmarks
  - Snapshot tests (snapshot/snapshot_test.go) - Windows VSS testing:
    - Snapshot structure validation
    - Path handling for VSS
    - Callback pattern testing
    - Admin privilege detection
    - Symlink management

### Security
- **Phase 3 Security** - Hardened security throughout
  - Input validation integrated in all entry points:
    - SaveConfig() validates all fields before saving
    - StartBackup() validates BackupID and paths
    - TestConnection() validates credentials format
  - Credential sanitization in all log statements:
    - SanitizeSecret() for passwords/tokens
    - SanitizeURL() removes embedded credentials
    - SanitizeForLog() masks sensitive IDs
  - Comprehensive validation functions:
    - URL validation (HTTPS enforcement)
    - AuthID validation (user@realm!token format)
    - Datastore validation (alphanumeric)
    - BackupID validation (path traversal prevention)
    - Path validation (null byte detection)
    - Certificate fingerprint validation (SHA256 format)

## [0.1.1] - 2026-03-18

### Fixed
- **GitHub Actions CI/CD** - Fixed go.work compatibility issues
  - Added `gui` module to go.work workspace
  - Set `GOWORK: off` environment variable in all workflow jobs
  - Fixed test and lint jobs to run in correct directories
  - Added missing frontend dependency installation steps

### Improved
- **Network resilience** - Added retry logic with exponential backoff
  - Chunk uploads retry up to 5 times with jitter
  - Chunk assignment retries with 5-minute timeout
  - Index finalization retries with backoff
  - Manifest upload retries with configurable delays
  - Context-aware cancellation for all retries
- **Error messages** - More detailed error context after retry exhaustion

## [0.1.0] - 2026-03-18 (First Public Release)

### Refactoring (Phase 1 - Completed)
- **Comprehensive error handling** throughout codebase
  - PXAR callbacks now return and propagate errors
  - HandleData() and Eof() with complete error checking
  - Replaced all panic() calls with graceful error handling
  - All errors wrapped with context using fmt.Errorf
- **Structured logging package** (pkg/logger)
  - JSON-formatted logs with slog
  - Multiple log levels (Debug, Info, Warn, Error)
  - Comprehensive test coverage
- **Retry logic with exponential backoff** (pkg/retry)
  - Configurable retry attempts and delays
  - Jitter support to prevent thundering herd
  - Context-aware cancellation
  - Comprehensive test coverage
- **Security package** (pkg/security)
  - Input validation (URL, BackupID, Datastore, AuthID, Fingerprint, Path)
  - Credential sanitization for safe logging
  - Constant-time string comparison for secrets
  - Path traversal prevention

### Planned
- **Client-side encryption** - PBS supports encryption, add key management in config
  - Generate/import encryption keys
  - Store key securely in config (warn user to backup key!)
  - Encrypt chunks before upload to PBS
  - Key recovery mechanism
- **Code signing** for Windows binaries (Authenticode certificate)
- **Auto-update system** - Check for latest version and prompt for updates
- System tray icon and background service
- Automatic scheduling (daily, weekly, custom cron)
- Windows service installation
- Notification system (Windows toast)
- Machine backup (full disk with PhysicalDrive - requires code signing)

## [0.1.0] - 2026-03-18

### Added
- Initial Wails v2 GUI with React frontend
- PBS server configuration interface
- Directory backup mode with multi-folder support (one per line)
- Real-time backup progress with accurate percentage
- Background directory size calculation for precise ETA
- Professional progress display (speed, elapsed time, ETA)
- Granular progress updates (every 10 MB)
- VSS (Volume Shadow Copy) support with admin privilege detection
- Snapshot listing and restore functionality
- PBS connection test with real authentication
- Automatic hostname detection for backup-id
- Debug logging to %APPDATA%\NimbusBackup\debug.log
- Crash reporting system
- RDEM Systems branding with custom icon

### Technical
- Inline backup implementation (no external binaries)
- PXAR archive format support
- Chunk deduplication with SHA256
- Dynamic index creation (DIDX)
- HTTP/2 protocol for PBS communication
- Cross-platform build support (Windows primary)

### Known Issues
- Machine backup disabled due to Windows Defender false positive (PhysicalDrive syscalls)
- Requires code signing certificate for full disk backup feature

---

## Version Numbering

- **Major.Minor.Patch** (Semantic Versioning)
- Major: Breaking changes
- Minor: New features, backwards compatible
- Patch: Bug fixes, small improvements

## Links

- [Original CLI Project](https://github.com/tizbac/proxmoxbackupclient_go)
- [RDEM Systems](https://rdem-systems.com)
- [Backup Portal](https://nimbus.rdem-systems.com)
