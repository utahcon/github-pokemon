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
		return runPruneRoot(cmd)
	},
	SilenceUsage: true,
}

func init() {
	pruneArchivedCmd.Flags().StringVarP(&pruneOrg, "org", "o", "", "GitHub organization to check repositories against (required)")
	pruneArchivedCmd.Flags().StringVarP(&prunePath, "path", "p", "", "Local path containing cloned repositories (required)")
	pruneArchivedCmd.Flags().BoolVar(&pruneConfirm, "confirm", false, "Actually remove directories (without this flag, runs in dry-run mode)")

	rootCmd.AddCommand(pruneArchivedCmd)
}

// runPruneRoot decides whether to use CLI flags (single org) or the config file (multi-org).
func runPruneRoot(cmd *cobra.Command) error {
	ctx := cmd.Context()

	if pruneOrg != "" && prunePath != "" {
		return runPruneArchived(ctx, pruneOrg, prunePath)
	}

	// If --org is set without --path, look it up in the config.
	if pruneOrg != "" && prunePath == "" {
		cfgFile := configPath
		if cfgFile == "" {
			var err error
			cfgFile, err = defaultConfigPath()
			if err != nil {
				return fmt.Errorf("--path is required when no config file is available: %w", err)
			}
		}
		cfg, err := loadOrCreateConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("--path is required: could not load config: %w", err)
		}
		entry, found := configLookupOrg(cfg, pruneOrg)
		if !found {
			return fmt.Errorf("org %q not found in config file %s; provide --path explicitly", pruneOrg, cfgFile)
		}
		return runPruneArchived(ctx, entry.Org, entry.Path)
	}

	// --path without --org doesn't make sense.
	if prunePath != "" {
		return fmt.Errorf("--org is required when --path is provided")
	}

	// No flags — load config file and run all orgs.
	cfgFile := configPath
	if cfgFile == "" {
		var err error
		cfgFile, err = defaultConfigPath()
		if err != nil {
			return fmt.Errorf("no --org/--path flags provided and %w", err)
		}
	}

	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("no --org/--path flags provided and config file unavailable: %w", err)
	}

	fmt.Printf("Loaded config with %d org(s) from %s\n\n", len(cfg.Orgs), cfgFile)

	var firstErr error
	for i, entry := range cfg.Orgs {
		if i > 0 {
			fmt.Println("---")
			fmt.Println()
		}

		fmt.Printf("Pruning archived repos for org: %s in %s\n\n", entry.Org, entry.Path)

		if err := runPruneArchived(ctx, entry.Org, entry.Path); err != nil {
			fmt.Printf("Error pruning org %s: %v\n", entry.Org, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func runPruneArchived(ctx context.Context, org string, path string) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN not set: set it with: export GITHUB_TOKEN=\"your-personal-access-token\"")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving target path: %w", err)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Errorf("reading target directory: %w", err)
	}

	client := github.NewClient(nil).WithAuthToken(token)

	allRepos, err := fetchOrgRepos(ctx, client, org)
	if err != nil {
		return fmt.Errorf("fetching repos for org %s: %w", org, err)
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

	if skippedCount > 0 {
		return fmt.Errorf("failed to remove %d archived repositories", skippedCount)
	}

	return nil
}