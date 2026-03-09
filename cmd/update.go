package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
)

const (
	repoOwner = "utahcon"
	repoName  = "github-pokemon"
)

type updateResult struct {
	latest string
	err    error
}

// checkForUpdate queries the GitHub API for the latest release and returns
// the tag name if it is newer than the current version. The check respects
// the supplied context so it can be cancelled if the main work finishes first.
func checkForUpdate(ctx context.Context, token string) <-chan updateResult {
	ch := make(chan updateResult, 1)
	go func() {
		defer close(ch)

		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client := github.NewClient(nil)
		if token != "" {
			client = client.WithAuthToken(token)
		}

		release, _, err := client.Repositories.GetLatestRelease(checkCtx, repoOwner, repoName)
		if err != nil {
			ch <- updateResult{err: err}
			return
		}

		latest := strings.TrimPrefix(release.GetTagName(), "v")
		if latest != "" && latest != version && isNewer(latest, version) {
			ch <- updateResult{latest: latest}
		}
	}()
	return ch
}

// isNewer reports whether latest is a higher semver than current.
// Both are expected in "MAJOR.MINOR.PATCH" form (no "v" prefix).
// Returns false if either cannot be parsed.
func isNewer(latest, current string) bool {
	lParts := parseSemver(latest)
	cParts := parseSemver(current)
	if lParts == nil || cParts == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if lParts[i] > cParts[i] {
			return true
		}
		if lParts[i] < cParts[i] {
			return false
		}
	}
	return false
}

// parseSemver splits a "MAJOR.MINOR.PATCH" string (ignoring any pre-release
// suffix after a hyphen) into three integers. Returns nil on failure.
func parseSemver(v string) []int {
	// Strip pre-release suffix (e.g. "1.0.0-dev" -> "1.0.0")
	if idx := strings.IndexByte(v, '-'); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil
			}
			n = n*10 + int(c-'0')
		}
		nums[i] = n
	}
	return nums
}

// printUpdateNotice prints a message to stderr if a newer version is available.
func printUpdateNotice(ch <-chan updateResult) {
	result, ok := <-ch
	if !ok || result.err != nil || result.latest == "" {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "\nA newer version of github-pokemon is available: v%s (current: v%s)\n", result.latest, version)
	_, _ = fmt.Fprintf(os.Stderr, "Download: https://github.com/%s/%s/releases/latest\n", repoOwner, repoName)
}
