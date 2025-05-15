# GitHub Repository Manager

[![Build and Release](https://github.com/utahcon/github-pokemon/actions/workflows/release.yml/badge.svg)](https://github.com/utahcon/github-pokemon/actions/workflows/release.yml)

A Go tool for efficiently managing multiple GitHub repositories from an organization.

## Features

- Quickly clone all non-archived repositories from a GitHub organization
- Update existing repositories by fetching remote tracking branches
- Process repositories in parallel for better performance
- SSH key support for authentication
- Safe operations - never modifies local working directories

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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Releasing

This project uses GitHub Actions to automatically build and release binaries:

1. Create and push a tag with version number:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. The GitHub Action will automatically build binaries for all supported platforms and create a release.

3. You can also manually trigger a build using the "workflow_dispatch" event in GitHub Actions.
