# GitHub Repository Manager (v1.0.2-builds)

[![Build and Release](https://github.com/utahcon/github-pokemon/actions/workflows/release.yml/badge.svg)](https://github.com/utahcon/github-pokemon/actions/workflows/release.yml)

A Go tool for efficiently managing multiple GitHub repositories from an organization.

## Features

- Quickly clone all non-archived repositories from a GitHub organization
- Update existing repositories by fetching remote tracking branches
- Process repositories in parallel for better performance
- SSH key support for authentication
- Safe operations - never modifies local working directories

## Prerequisites

- Git must be installed and available in your PATH
- A GitHub personal access token with `repo` scope
- For SSH authentication: properly configured SSH keys for GitHub

## Installation

### Download Pre-built Binary

You can download the latest pre-built binary for your platform from the [Releases](https://github.com/utahcon/github-pokemon/releases) page.

We provide builds for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

After downloading, make the binary executable:

```bash
chmod +x github-pokemon-*
```

### Build from Source

```bash
# Clone this repository
git clone https://github.com/utahcon/github-pokemon.git
cd github-pokemon

# Build the binary
go build -o github-repo-manager

# Optionally move to your path
mv github-repo-manager /usr/local/bin/
```

## Usage

```bash
# Set GitHub token in environment variable
export GITHUB_TOKEN="your-github-personal-access-token"

# Basic usage with required parameters
github-repo-manager --org "organization-name" --path "/path/to/store/repos"

# Check version
github-repo-manager --version

# The tool will automatically check if git is installed and if GITHUB_TOKEN is set
```

### Command Line Options

```
Flags:
  -h, --help           Display help information
  -o, --org string     GitHub organization to fetch repositories from (required)
  -j, --parallel int   Number of repositories to process in parallel (default 5)
  -p, --path string    Local path to clone/update repositories to (required)
  -s, --skip-update    Skip updating existing repositories
  -v, --verbose        Enable verbose output
  -V, --version        Show version information and exit
```

### Examples

```bash
# Clone/fetch with 10 parallel workers
github-repo-manager --org "my-organization" --path "./repos" --parallel 10

# Skip updating existing repositories
github-repo-manager --org "my-organization" --path "./repos" --skip-update

# Verbose output with status information
github-repo-manager --org "my-organization" --path "./repos" --verbose
```

## How It Works

1. The tool queries the GitHub API to list all repositories in the specified organization
2. For each non-archived repository:
   - If it doesn't exist locally, it clones the repository
   - If it exists locally, it only fetches updates to remote tracking branches
3. Local working directories are never modified - the tool only updates remote tracking information

## Authentication

- **GitHub API**: Uses a personal access token via the `GITHUB_TOKEN` environment variable
- **Git operations**: Uses SSH keys configured in your system

### Version Management

This project uses a flexible version management approach:

- During development: Uses branch-based versions (e.g., `1.0.0-feature_branch`)
- For releases: Uses Semantic Versioning (e.g., `1.0.0`)

Version information is stored in a standalone `VERSION` file in the project root.

#### Setting Development Versions

To set a development version based on your branch name:

```bash
# Automatically create a version using current branch name
./scripts/set-dev-version.sh

# Specify a base version with branch suffix
./scripts/set-dev-version.sh 1.2.0
```

#### Setting Release Versions

When preparing a PR to main, update to a proper semantic version:

```bash
# Just edit the VERSION file directly
echo "1.2.0" > VERSION

# And update the CHANGELOG.md accordingly
```

### Setting Up Authentication

1. **GitHub Personal Access Token**:
   - Go to GitHub → Settings → Developer settings → Personal access tokens
   - Create a token with `repo` scope
   - Set it in your environment: `export GITHUB_TOKEN="your-token-here"`

2. **SSH Authentication**:
   - Generate SSH keys: `ssh-keygen -t ed25519 -C "your_email@example.com"`
   - Add to SSH agent: `eval "$(ssh-agent -s)" && ssh-add ~/.ssh/id_ed25519`
   - Add to GitHub: Copy your public key (`cat ~/.ssh/id_ed25519.pub`) and add it in GitHub → Settings → SSH keys

## Benefits

- **Safe Operations**: Never modifies local branches or working directories
- **High Performance**: Parallel processing significantly speeds up operations
- **Organization Backup**: Easily create and maintain a local backup of all repositories
- **Developer Onboarding**: Quickly clone all repositories for new team members

## License

MIT

## Example Workflow

Here's a common workflow for using this tool:

```bash
# Initial setup
export GITHUB_TOKEN="your-github-personal-access-token"
mkdir -p ~/github-repos

# Clone all repositories from your organization
github-repo-manager --org "your-organization" --path ~/github-repos

# Update repositories daily to stay current (could be in a cron job)
github-repo-manager --org "your-organization" --path ~/github-repos

# After updating, if you want to update local branches in a specific repository:
cd ~/github-repos/specific-repo
git status      # Check how far behind/ahead you are
git merge origin/main   # Merge remote changes into your local branch

# Or if you prefer rebasing:
git rebase origin/main  # Rebase your local changes on top of remote changes
```

### Automating with Cron

You can set up a cron job to automatically update your repositories:

```bash
# Edit your crontab
crontab -e

# Add a line to run the tool daily at 9 AM
0 9 * * * export GITHUB_TOKEN="your-token"; /path/to/github-repo-manager --org "your-organization" --path ~/github-repos
```

### Troubleshooting

#### Authentication Issues

If you encounter errors like "permission denied" or "authentication failed":

1. Verify your GitHub token has the correct permissions
2. Check that your SSH key is properly set up with GitHub
3. Ensure your SSH agent is running: `eval "$(ssh-agent -s)"`
4. Try the verbose flag for more detailed output: `github-repo-manager --org "your-org" --path "./repos" --verbose`

The tool will automatically detect authentication issues and provide helpful guidance.

#### Git Hook Issues

If you're having problems with Git hooks:

1. Make sure the hooks are installed: `./scripts/install-hooks.sh`
2. Check hook permissions: `ls -la .githooks/`
3. You can bypass hooks temporarily with: `git push --no-verify`
4. If hooks aren't triggering, verify Git configuration: `git config core.hooksPath`

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Install Git hooks (`./scripts/install-hooks.sh`)
4. Set a development version (`./scripts/set-dev-version.sh`)
5. Make your changes
6. Update CHANGELOG.md with your changes
7. Update to a proper release version when ready for a PR to main
8. Run the version check (`./scripts/check-version.sh`)
9. Commit your changes (`git commit -m 'Add some amazing feature'`)
10. Push to the branch (`git push origin feature/amazing-feature`)
11. Open a Pull Request

The CI workflow will verify that your changes include a version increment and CHANGELOG update when targeting the main branch.

### Releasing and Version Management

This project uses GitHub Actions to automatically build and release binaries. See [CHANGELOG.md](CHANGELOG.md) for version history details.

We follow [Semantic Versioning](https://semver.org/) for version numbers (MAJOR.MINOR.PATCH).

#### Version Guardrails

The main branch is protected and requires that PRs increment the version number:

1. Any PR to main must contain a proper semantic version in the `VERSION` file
2. The CHANGELOG.md must be updated with the new version
3. A GitHub Actions workflow verifies these requirements on every PR to main
4. During development, branch-based versions are allowed (e.g., `1.0.0-feature_branch`)

#### Automatic Version Synchronization

When changes are merged to main, a GitHub Actions workflow automatically:

1. Extracts the current version from `cmd/root.go`
2. Updates all version references in the documentation (README, CHANGELOG, etc.)
3. Commits and pushes these synchronized changes back to the main branch

You can also run version synchronization locally:

```bash
# Using the version from VERSION file
./scripts/sync-versions.sh

# Or specifying a version
./scripts/sync-versions.sh 1.2.3
```

#### Git Hooks for Version Management

This repository includes Git hooks to ensure version compliance before pushing:

1. Install the hooks:
   ```bash
   ./scripts/install-hooks.sh
   ```

2. The pre-push hook will automatically:
   - Detect when you're pushing to the main branch
   - Check if version-related files were modified
   - Run strict version checks only when pushing to main
   - Allow development branch-based versions in other branches
   - Block the push to main if semantic version rules aren't followed

This helps catch version issues early, before even creating a PR.

#### Creating a Release

1. Create a feature branch:
   ```bash
   git checkout -b feature/my-feature
   ```

2. Update the VERSION file with a proper semantic version and update CHANGELOG.md

3. Run the version check script to validate:
   ```bash
   ./scripts/check-version.sh
   ```

4. Open a PR to the main branch

5. After merging to main, create and push a tag with the version number:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

6. The GitHub Action will automatically build binaries for all supported platforms and create a release.

7. You can also manually trigger a build using the "workflow_dispatch" event in GitHub Actions.
