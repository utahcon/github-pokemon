package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-github/v84/github"
	"github.com/spf13/cobra"
)

var (
	pruneOrg     string
	prunePath    string
	pruneConfirm bool
)

var pruneArchivedCmd = &cobra.Command{
	Use:   "prune-archived",
	Short: "Remove local directories for repositories that are archived on GitHub",
	Long: `Scans directories in the target path and checks each one against the GitHub API.
Any directory that corresponds to an archived repository in the specified organization
will be removed.

By default runs in dry-run mode so you can preview what would be deleted.
Pass --confirm to actually remove directories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPruneArchived(cmd.Context())
	},
	SilenceUsage: true,
}

func init() {
	pruneArchivedCmd.Flags().StringVarP(&pruneOrg, "org", "o", "", "GitHub organization to check repositories against (required)")
	pruneArchivedCmd.Flags().StringVarP(&prunePath, "path", "p", "", "Local path containing cloned repositories (required)")
	pruneArchivedCmd.Flags().BoolVar(&pruneConfirm, "confirm", false, "Actually remove directories (without this flag, runs in dry-run mode)")

	_ = pruneArchivedCmd.MarkFlagRequired("org")
	_ = pruneArchivedCmd.MarkFlagRequired("path")

	rootCmd.AddCommand(pruneArchivedCmd)
}

func runPruneArchived(ctx context.Context) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN not set: set it with: export GITHUB_TOKEN=\"your-personal-access-token\"")
	}

	absPath, err := filepath.Abs(prunePath)
	if err != nil {
		return fmt.Errorf("resolving target path: %w", err)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Errorf("reading target directory: %w", err)
	}

	client := github.NewClient(nil).WithAuthToken(token)

	allRepos, err := fetchOrgRepos(ctx, client, pruneOrg)
	if err != nil {
		return err
	}

	archivedSet := make(map[string]bool)
	for _, repo := range allRepos {
		if repo.GetArchived() {
			archivedSet[repo.GetName()] = true
		}
	}

	removedCount := 0
	skippedCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !archivedSet[name] {
			continue
		}

		dirPath := filepath.Join(absPath, name)

		if !pruneConfirm {
			fmt.Printf("[dry-run] would remove: %s\n", dirPath)
			removedCount++
			continue
		}

		fmt.Printf("Removing archived repository: %s\n", dirPath)
		if err := os.RemoveAll(dirPath); err != nil {
			fmt.Printf("Error removing %s: %v\n", dirPath, err)
			skippedCount++
			continue
		}
		removedCount++
	}

	if !pruneConfirm {
		fmt.Printf("\nDry-run complete: %d archived repositories would be removed\n", removedCount)
		if removedCount > 0 {
			fmt.Println("To actually remove them, re-run with the --confirm flag")
		}
	} else {
		fmt.Printf("\nSummary: Removed %d archived repositories", removedCount)
		if skippedCount > 0 {
			fmt.Printf(" (%d failed)", skippedCount)
		}
		fmt.Println()
	}

	return nil
}