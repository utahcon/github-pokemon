package cmd

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

type repoAction int

const (
	actionCloned repoAction = iota
	actionFetched
	actionSkipped
	actionErrored
)

func actionIcon(a repoAction) string {
	switch a {
	case actionCloned:
		return "\u2713 cloned"
	case actionFetched:
		return "\u21bb fetched"
	case actionSkipped:
		return "\u2298 skipped"
	case actionErrored:
		return "\u2717 error"
	default:
		return "? unknown"
	}
}

func actionColor(a repoAction) *color.Color {
	switch a {
	case actionCloned:
		return color.New(color.FgGreen)
	case actionFetched:
		return color.New(color.FgGreen)
	case actionSkipped:
		return color.New(color.FgYellow)
	case actionErrored:
		return color.New(color.FgRed)
	default:
		return color.New(color.Reset)
	}
}

func collectAndDisplay(resultsCh <-chan repoResult, total int, verboseMode bool, startTime time.Time) (int, bool) {
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetDescription("Processing repos"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionClearOnFinish(),
	)

	var results []repoResult
	for r := range resultsCh {
		results = append(results, r)
		_ = bar.Add(1)
	}
	_ = bar.Finish()
	fmt.Fprintln(os.Stderr) // blank line after progress bar clears

	errorCount := 0
	hadAuth := false
	for _, r := range results {
		if r.action == actionErrored {
			errorCount++
			if r.err != nil && isAuthRelated(r.err.Error()) {
				hadAuth = true
			}
		}
	}

	printGroupedResults(results, verboseMode)
	printSummary(results, total, time.Since(startTime), organization, hadAuth)

	return errorCount, hadAuth
}

func printGroupedResults(results []repoResult, verboseMode bool) {
	groups := map[repoAction][]repoResult{
		actionCloned:  {},
		actionFetched: {},
		actionSkipped: {},
		actionErrored: {},
	}

	for _, r := range results {
		groups[r.action] = append(groups[r.action], r)
	}

	// Sort each group alphabetically by repo name
	for a := range groups {
		sort.Slice(groups[a], func(i, j int) bool {
			return groups[a][i].repoName < groups[a][j].repoName
		})
	}

	order := []repoAction{actionCloned, actionFetched, actionSkipped, actionErrored}
	labels := map[repoAction]string{
		actionCloned:  "Cloned",
		actionFetched: "Fetched",
		actionSkipped: "Skipped",
		actionErrored: "Errors",
	}

	for _, a := range order {
		group := groups[a]
		if len(group) == 0 {
			continue
		}

		fmt.Printf("%s (%d):\n", labels[a], len(group))
		c := actionColor(a)
		icon := actionIcon(a)

		for _, r := range group {
			c.Printf("  %-10s %s", icon, r.repoName)
			if verboseMode && r.duration > 0 {
				fmt.Printf("  (%s)", r.duration.Round(time.Millisecond))
			}
			fmt.Println()

			if a == actionErrored && r.err != nil {
				c.Printf("             %s\n", r.err.Error())
			}

			if verboseMode && r.verboseDetail != "" {
				fmt.Printf("             %s\n", r.verboseDetail)
			}
		}
		fmt.Println()
	}
}

func printSummary(results []repoResult, total int, elapsed time.Duration, org string, hadAuth bool) {
	counts := map[repoAction]int{}
	for _, r := range results {
		counts[r.action]++
	}

	fmt.Println("--- Summary ---")
	fmt.Printf("Organization: %s\n", org)
	fmt.Printf("Total repos:  %d (non-archived)\n", total)

	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	green.Printf("  \u2713 Cloned:  %3d\n", counts[actionCloned])
	green.Printf("  \u21bb Fetched: %3d\n", counts[actionFetched])
	yellow.Printf("  \u2298 Skipped: %3d\n", counts[actionSkipped])
	red.Printf("  \u2717 Errors:  %3d\n", counts[actionErrored])
	fmt.Printf("Elapsed: %.1fs\n", elapsed.Seconds())

	if hadAuth {
		fmt.Println()
		red.Println("Some authentication errors were detected. Please verify your setup:")
		fmt.Println("1. SSH setup guide: https://docs.github.com/en/authentication/connecting-to-github-with-ssh")
		fmt.Println("2. Personal access token guide: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token")
	}

	fmt.Println()
	fmt.Println("Note: For existing repositories, only 'git fetch --all' was performed.")
	fmt.Println("Local branches were not modified. Use 'git merge' or 'git rebase' manually to update local branches.")
}
