# Changelog

All notable changes to Proxmox Backup Guardian GUI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Native GUI interface with Fyne
- Multi-folder selection for backups
- Disk detection for machine backups
- Exclusion patterns with presets (dev, temp, caches, media)
- Automatic scheduling (cron on Linux, Task Scheduler on Windows)
- Job management (create, edit, delete, export)
- Configuration import/export (JSON and INI formats)
- Retention policy configuration
- Advanced options (compression, chunk size, bandwidth limit)
- PBS connection test
- Real-time backup monitoring with progress bar

### Documentation
- GUI_README.md - Complete GUI documentation
- CONFIG_FORMAT_SPEC.md - JSON/INI format specifications
- INTEGRATION_MEMBERS.md - Laravel integration guide
- SUMMARY.md - Technical overview

### CI/CD
- GitLab CI/CD pipeline
- Multi-platform builds (Linux, Windows, macOS)
- Automated testing and linting
- Release artifacts packaging
- Docker image support

## [1.0.0] - 2026-03-17

### Added
- Initial GUI release based on proxmoxbackupclient_go CLI
- Native cross-platform interface (Windows, Linux, macOS)
- Visual PBS configuration
- Job scheduling and management
- Integration with members.rdem-systems.com

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
