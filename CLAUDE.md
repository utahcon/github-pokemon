# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go CLI tool (`github-pokemon`) that clones and fetches all non-archived repositories from a GitHub organization in parallel. Uses SSH for git operations and a `GITHUB_TOKEN` for API access. Built with Cobra for CLI and `go-github` for the GitHub API.

## Build & Run

```bash
# Build
go build -o github-pokemon .

# Build static binary (as CI does)
CGO_ENABLED=0 go build -a -ldflags="-w -s -extldflags '-static'" -o github-pokemon .

# Run
export GITHUB_TOKEN="your-token"
./github-pokemon --org "org-name" --path "/path/to/repos"
```

There are no tests currently in this project.

## Architecture

Single-command Cobra CLI app. All logic lives in two files:

- `main.go` — entrypoint, calls `cmd.Execute()`
- `cmd/root.go` — CLI flags, GitHub API pagination, worker pool for parallel clone/fetch

The worker pool pattern: `runRootCommand()` creates a buffered channel of repos, spawns `--parallel` (default 5) goroutines via `worker()`, each calling `processRepository()` which either `git clone` or `git fetch --all`.

Version is set via ldflags at build time by GoReleaser, or falls back to the Go module version embedded by `go install`.

## Version Management

This project uses [release-please](https://github.com/googleapis/release-please) for automated versioning and releases:

1. Commit messages **must** follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `chore:`, etc.)
2. On merge to `main`, release-please analyzes commits and opens/updates a release PR with version bump and CHANGELOG updates
3. When the release PR is merged, release-please creates a Git tag (`v*`)
4. The `v*` tag triggers `release.yml` which builds cross-platform binaries via GoReleaser

## CI Workflows

- **ci.yml** — Runs build, test, lint (golangci-lint v2), and GoReleaser config validation on PRs to main
- **release-please.yml** — Manages release PRs and tagging on pushes to main
- **release.yml** — Builds linux/darwin/windows (amd64/arm64) static binaries on `v*` tags via GoReleaser
