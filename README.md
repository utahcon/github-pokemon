# GitHub Repository Manager

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

### Update Notifications

The tool automatically checks for newer releases on GitHub when it runs. If a newer version is available, a notice is printed to stderr at the end of the run. This check runs in the background with a 5-second timeout and never blocks the main operation.

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Commit using [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, etc.)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

### Releasing

This project uses [release-please](https://github.com/googleapis/release-please) for automated versioning:

1. Commit messages must follow Conventional Commits
2. On merge to `main`, release-please opens/updates a release PR with version bump and CHANGELOG
3. When the release PR is merged, a Git tag is created
4. The tag triggers GoReleaser to build cross-platform binaries and create a GitHub release
