# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0](https://github.com/utahcon/github-pokemon/compare/v1.1.0...v1.2.0) (2026-03-06)


### Features

* improve CLI output with colored grouped results and progress bar ([1384885](https://github.com/utahcon/github-pokemon/commit/1384885cbcb82667019959bce19579edda1842cd))
* improve CLI output with colored grouped results and progress bar ([9ff3564](https://github.com/utahcon/github-pokemon/commit/9ff35640cd271b96eb73904cf55c2a7c2b6de8b2))


### Bug Fixes

* resolve lint errors for errcheck and unused constant ([3b047b2](https://github.com/utahcon/github-pokemon/commit/3b047b20f869cc75953a562493b64688a7149856))

## [1.1.0](https://github.com/utahcon/github-pokemon/compare/v1.0.0...v1.1.0) (2026-03-06)


### Features

* add GoReleaser, CI pipeline, and security hardening ([cbb4e6f](https://github.com/utahcon/github-pokemon/commit/cbb4e6f067be877a81fcca385287c6558cf9886c))
* GoReleaser, CI pipeline, and automated releases ([fa416f6](https://github.com/utahcon/github-pokemon/commit/fa416f6f0f2c68958bf162fafa406f1266033172))
* GoReleaser, CI pipeline, and security hardening ([f75fc9e](https://github.com/utahcon/github-pokemon/commit/f75fc9eff9691a230d5de063bc1a81ba0cfe1879))
* replace manual versioning with release-please ([6bfa62f](https://github.com/utahcon/github-pokemon/commit/6bfa62f59a8c12fefd3a981f6049597e12fb7b32))
* support version detection via go install ([8af98e9](https://github.com/utahcon/github-pokemon/commit/8af98e97930c3fa1405f8a6426ee8bf367599098))


### Bug Fixes

* address code review feedback ([d88d54a](https://github.com/utahcon/github-pokemon/commit/d88d54a64faa5373ecacb4c9a7880cbb85d6a0f5))
* check MarkFlagRequired return values to satisfy errcheck lint ([659bec9](https://github.com/utahcon/github-pokemon/commit/659bec9bb27bb76782feed96a0ad06a75949452e))
* pin golangci-lint to v2.11.1 for Go 1.26 compatibility ([b392161](https://github.com/utahcon/github-pokemon/commit/b392161b78d7fa04bc2f40c673796732ef8c474a))
* upgrade golangci-lint-action to v7 for golangci-lint v2 support ([0d46c42](https://github.com/utahcon/github-pokemon/commit/0d46c42e4abeaa75c286a85628c53ff80b84e729))
* use non-deprecated goreleaser v2 archive format syntax ([6c114ba](https://github.com/utahcon/github-pokemon/commit/6c114ba6b10cf7f281cd339eca8f79efd89f2a99))

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
