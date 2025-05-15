# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.2-builds] - 2023-10-16

### Added
- Standalone VERSION file for improved version management
- Flexible development versioning with branch-based versions
- Version guardrails to prevent merging to main without version increment
- GitHub Actions workflow to validate version updates on PRs
- Local script (scripts/check-version.sh) to verify version changes
- Script to generate development versions (scripts/set-dev-version.sh)
- Git hooks for pre-push version validation
- Script to install Git hooks (scripts/install-hooks.sh)

## [1.0.1] - 2023-10-15

### Added
- Git installation check at startup
- Improved GITHUB_TOKEN validation with helpful error messages
- Detection and clear guidance for authentication issues
- Comprehensive troubleshooting information in error messages
- Authentication-specific error reporting in summary output
- Prerequisites section in README
- Detailed authentication setup guides in README
- Troubleshooting section in README

### Changed
- Enhanced GitHub Actions workflow with proper permissions for releases
- Improved error messages with actionable next steps
- Updated README with more detailed installation and usage instructions

### Fixed
- GitHub Actions release process permission issues
- Clearer error handling for authentication failures
- Better guidance when SSH or token authentication fails

## [1.0.0] - 2023-10-10

### Added
- Initial release
- Support for cloning non-archived repositories from a GitHub organization
- Fetch-only updates for existing repositories (no local branch modifications)
- Parallel processing with configurable number of workers
- SSH key support for Git operations
- Skip-update option for existing repositories
- Verbose mode for detailed status information
- GitHub Actions workflow for cross-platform builds
- Comprehensive README with usage examples
