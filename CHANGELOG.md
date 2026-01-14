# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2025-01-14

### Added
- Unit tests for CLI commands (`internal/cli/cli_test.go`)
- Unit tests for Supervisor mode (`internal/cli/supervisor_mode_test.go`)
- Unit tests for pretty JSON formatting (`internal/prettyjson/`)

### Changed
- **Supervisor mode activation**: Changed from environment variable to slash command (`/supervisor`)
  - Use `/supervisor` to enable supervisor mode
  - Use `/supervisor-off` to disable supervisor mode
- Renamed command file `supervisor-off.md` → `supervisoroff.md`

### Removed
- Obsolete and incomplete tests
- Tests for non-existent `Enabled` field and deprecated environment variables
- Dead code in integration tests

### Fixed
- E2E tests for supervisor-hook command

## [0.2.0] - 2025-01-13

### Added
- Supervisor Mode with automatic task review
- Support for custom supervisor prompt via `~/.claude/SUPERVISOR.md`
- Structured logging and unified error handling
- MIT License

### Changed
- Repositioned project as "Claude Code Supervisor"
- Repository renamed from `claude-code-config-switcher` to `claude-code-supervisor`

[Unreleased]: https://github.com/guyskk/claude-code-supervisor/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/guyskk/claude-code-supervisor/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/guyskk/claude-code-supervisor/releases/tag/v0.2.0
