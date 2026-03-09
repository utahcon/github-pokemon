package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/google/go-github/v84/github"
	"github.com/spf13/cobra"
)

// version is set via ldflags at build time (GoReleaser) or falls back to
// the module version embedded by go install.
var version = func() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "0.0.0-dev"
}()

var (
	organization    string
	targetPath      string
	skipUpdate      bool
	verbose         bool
	parallelLimit   int
	includeArchived bool
	noColor       bool
)

const maxParallelLimit = 50

// authError wraps an error to indicate it was caused by an authentication problem.
type authError struct {
	err error
}

func (e *authError) Error() string { return e.err.Error() }
func (e *authError) Unwrap() error { return e.err }

// repoResult holds the outcome of processing a single repository.
type repoResult struct {
	repoName      string
	action        repoAction
	err           error
	duration      time.Duration
	verboseDetail string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "github-pokemon",
	Short: "Clone non-archived repositories for a GitHub organization",
	Long: `This tool fetches all non-archived repositories for a specified GitHub organization
and either clones them (if they don't exist locally) or fetches remote updates (if they do exist)
to a specified local path. Use --include-archived to also process archived repositories.

The tool will never modify local branches - it only fetches remote tracking information
for existing repositories.

GitHub API credentials are expected to be in environment variables:
- GITHUB_TOKEN: Personal access token for GitHub API`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRootCommand(cmd.Context())
	},
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("github-pokemon version {{.Version}}\n")

	rootCmd.Flags().StringVarP(&organization, "org", "o", "", "GitHub organization to fetch repositories from (required)")
	rootCmd.Flags().StringVarP(&targetPath, "path", "p", "", "Local path to clone/update repositories to (required)")
	rootCmd.Flags().BoolVarP(&skipUpdate, "skip-update", "s", false, "Skip updating existing repositories")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().IntVarP(&parallelLimit, "parallel", "j", 5, "Number of repositories to process in parallel")
	rootCmd.Flags().BoolVar(&includeArchived, "include-archived", false, "Include archived repositories")

	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	_ = rootCmd.MarkFlagRequired("org")
	_ = rootCmd.MarkFlagRequired("path")
}

// isAuthRelated returns true if the output contains authentication-related keywords.
func isAuthRelated(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "authenticity") ||
		strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "could not read username") ||
		strings.Contains(lower, "auth")
}

// processRepository handles cloning or fetching for a single repository.
// It returns the action taken, an optional verbose detail string, and an error.
func processRepository(ctx context.Context, repo *github.Repository, repoPath string, skipUpdate bool, verboseMode bool) (repoAction, string, error) {
	repoName := repo.GetName()

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		cloneURL := repo.GetSSHURL()
		if cloneURL == "" {
			cloneURL = repo.GetCloneURL()
		}

		cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, repoPath)
		output, err := cmd.CombinedOutput()

		if err != nil {
			cloneErr := fmt.Errorf("cloning repository %s: %w\n%s", repoName, err, output)
			if isAuthRelated(string(output)) {
				return actionErrored, "", &authError{err: cloneErr}
			}
			return actionErrored, "", cloneErr
		}

		return actionCloned, "", nil
	}

	if skipUpdate {
		return actionSkipped, "", nil
	}

	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--all")
	fetchCmd.Dir = repoPath
	fetchOutput, err := fetchCmd.CombinedOutput()

	if err != nil {
		fetchErr := fmt.Errorf("fetching repository %s: %w\n%s", repoName, err, fetchOutput)
		if isAuthRelated(string(fetchOutput)) {
			return actionErrored, "", &authError{err: fetchErr}
		}
		return actionErrored, "", fetchErr
	}

	var detail string
	if verboseMode {
		statusCmd := exec.CommandContext(ctx, "git", "status", "-sb")
		statusCmd.Dir = repoPath
		statusOutput, err := statusCmd.CombinedOutput()
		if err == nil {
			detail = strings.TrimSpace(string(statusOutput))
		}
	}

	return actionFetched, detail, nil
}

// worker processes repositories from jobs and sends results to the results channel.
// absTargetPath must be the absolute path of the target directory (computed once by the caller).
func worker(ctx context.Context, jobs <-chan *github.Repository, results chan<- repoResult, absTargetPath string, skipUpdate bool, verboseMode bool) {
	for repo := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		repoName := repo.GetName()

		// Validate repoName to prevent path traversal
		if repoName == ".." || strings.ContainsAny(repoName, `/\`) {
			results <- repoResult{
				repoName: repoName,
				action:   actionErrored,
				err:      fmt.Errorf("skipping repository %q: name contains invalid path characters", repoName),
			}
			continue
		}

		absRepo := filepath.Join(absTargetPath, repoName)

		// Verify the resolved path stays within targetPath
		if !strings.HasPrefix(absRepo+string(os.PathSeparator), absTargetPath+string(os.PathSeparator)) {
			results <- repoResult{
				repoName: repoName,
				action:   actionErrored,
				err:      fmt.Errorf("skipping repository %q: resolved path escapes target directory", repoName),
			}
			continue
		}

		start := time.Now()
		action, detail, err := processRepository(ctx, repo, absRepo, skipUpdate, verboseMode)
		elapsed := time.Since(start)

		results <- repoResult{
			repoName:      repoName,
			action:        action,
			err:           err,
			duration:      elapsed,
			verboseDetail: detail,
		}
	}
}

// fetchOrgRepos retrieves all repositories for the given organization using pagination.
func fetchOrgRepos(ctx context.Context, client *github.Client, org string) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Type:        "all",
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			return nil, fmt.Errorf("listing repositories: %w", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, nil
}

// filterNonArchived returns only repos that are not archived.
func filterNonArchived(repos []*github.Repository) []*github.Repository {
	var result []*github.Repository
	for _, repo := range repos {
		if !repo.GetArchived() {
			result = append(result, repo)
		}
	}
	return result
}

func runRootCommand(ctx context.Context) error {
	// Start update check in background (non-blocking, 5s timeout).
	token := os.Getenv("GITHUB_TOKEN")
	updateCh := checkForUpdate(ctx, token)
	defer printUpdateNotice(updateCh)
	if noColor {
		color.NoColor = true
	}

	if parallelLimit <= 0 {
		parallelLimit = 5
	}
	if parallelLimit > maxParallelLimit {
		return fmt.Errorf("parallel limit %d exceeds maximum of %d", parallelLimit, maxParallelLimit)
	}

	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}

	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving target path: %w", err)
	}

	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN not set: set it with: export GITHUB_TOKEN=\"your-personal-access-token\"")
	}

	if verbose {
		fmt.Println("GitHub token found in environment")
	}

	client := github.NewClient(nil).WithAuthToken(token)

	startTime := time.Now()

	allRepos, err := fetchOrgRepos(ctx, client, organization)
	if err != nil {
		return err
	}

	var repos []*github.Repository
	if includeArchived {
		repos = allRepos
	} else {
		repos = filterNonArchived(allRepos)
	}
	repoCount := len(repos)

	nonArchivedCount := len(filterNonArchived(allRepos))
	fmt.Printf("Found %d repositories in organization %s (%d non-archived)\n",
		len(allRepos), organization, nonArchivedCount)

	if includeArchived {
		fmt.Printf("Including archived repositories (--include-archived)\n")
	}

	if repoCount == 0 {
		fmt.Println("No repositories found. Nothing to process.")
		return nil
	}

	fmt.Printf("Processing repositories with %d parallel workers\n\n", parallelLimit)

	jobs := make(chan *github.Repository, repoCount)
	results := make(chan repoResult, repoCount)

	var wg sync.WaitGroup
	for w := 0; w < parallelLimit; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, jobs, results, absTargetPath, skipUpdate, verbose)
		}()
	}

	for _, repo := range repos {
		jobs <- repo
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	errorCount, _ := collectAndDisplay(results, repoCount, verbose, startTime)

	if errorCount > 0 {
		return fmt.Errorf("failed to process %d repositories", errorCount)
	}

	return nil
}
