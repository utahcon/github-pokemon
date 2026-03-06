package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/google/go-github/v84/github"
	"github.com/spf13/cobra"
)

// Version information - set via ldflags at build time:
//
//	go build -ldflags="-X github.com/utahcon/github-pokemon/cmd.version=1.0.0"
var version = "0.0.0-dev"

var (
	organization  string
	targetPath    string
	skipUpdate    bool
	verbose       bool
	parallelLimit int
)

const maxParallelLimit = 50

// authErrorGuidance is the help text shown when an authentication error is detected.
const authErrorGuidance = "\n\nAuthentication error detected. Please ensure:\n" +
	"1. Your SSH key is set up correctly with GitHub: https://docs.github.com/en/authentication/connecting-to-github-with-ssh\n" +
	"2. Your GITHUB_TOKEN has sufficient permissions\n" +
	"3. For SSH: Your SSH agent is running ('eval $(ssh-agent -s)')\n" +
	"4. For HTTPS: You may need to configure credential helper ('git config --global credential.helper cache')"

// authError wraps an error to indicate it was caused by an authentication problem.
type authError struct {
	err error
}

func (e *authError) Error() string { return e.err.Error() }
func (e *authError) Unwrap() error { return e.err }

// repoResult holds the outcome of processing a single repository.
type repoResult struct {
	repoName string
	success  bool
	message  string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "github-pokemon",
	Short: "Clone non-archived repositories for a GitHub organization",
	Long: `This tool fetches all non-archived repositories for a specified GitHub organization
and either clones them (if they don't exist locally) or fetches remote updates (if they do exist)
to a specified local path.

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

	rootCmd.MarkFlagRequired("org")
	rootCmd.MarkFlagRequired("path")
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
// On success it returns a descriptive message and nil error.
// On failure it returns an empty string and an error (possibly an *authError).
func processRepository(ctx context.Context, repo *github.Repository, repoPath string, skipUpdate bool, verbose bool) (string, error) {
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
				return "", &authError{err: fmt.Errorf("%w%s", cloneErr, authErrorGuidance)}
			}
			return "", cloneErr
		}

		return fmt.Sprintf("successfully cloned repository: %s", repoName), nil
	}

	if skipUpdate {
		if verbose {
			return fmt.Sprintf("skipping update for existing repository: %s", repoName), nil
		}
		return fmt.Sprintf("repository exists (skipping): %s", repoName), nil
	}

	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--all")
	fetchCmd.Dir = repoPath
	fetchOutput, err := fetchCmd.CombinedOutput()

	if err != nil {
		fetchErr := fmt.Errorf("fetching repository %s: %w\n%s", repoName, err, fetchOutput)
		if isAuthRelated(string(fetchOutput)) {
			return "", &authError{err: fmt.Errorf("%w%s", fetchErr, authErrorGuidance)}
		}
		return "", fetchErr
	}

	successMsg := fmt.Sprintf("successfully fetched updates for %s", repoName)

	if verbose {
		statusCmd := exec.CommandContext(ctx, "git", "status", "-sb")
		statusCmd.Dir = repoPath
		statusOutput, err := statusCmd.CombinedOutput()
		if err == nil {
			successMsg += fmt.Sprintf("\nStatus for %s:\n%s", repoName, statusOutput)
		}
	}

	return successMsg, nil
}

// worker processes repositories from jobs and sends results to the results channel.
// absTargetPath must be the absolute path of the target directory (computed once by the caller).
func worker(ctx context.Context, jobs <-chan *github.Repository, results chan<- repoResult, absTargetPath string, skipUpdate bool, verbose bool) {
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
				success:  false,
				message:  fmt.Sprintf("skipping repository %q: name contains invalid path characters", repoName),
			}
			continue
		}

		absRepo := filepath.Join(absTargetPath, repoName)

		// Verify the resolved path stays within targetPath
		if !strings.HasPrefix(absRepo+string(os.PathSeparator), absTargetPath+string(os.PathSeparator)) {
			results <- repoResult{
				repoName: repoName,
				success:  false,
				message:  fmt.Sprintf("skipping repository %q: resolved path escapes target directory", repoName),
			}
			continue
		}

		msg, err := processRepository(ctx, repo, absRepo, skipUpdate, verbose)
		if err != nil {
			results <- repoResult{
				repoName: repoName,
				success:  false,
				message:  err.Error(),
			}
		} else {
			results <- repoResult{
				repoName: repoName,
				success:  true,
				message:  msg,
			}
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

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN not set: set it with: export GITHUB_TOKEN=\"your-personal-access-token\"")
	}

	if verbose {
		fmt.Println("GitHub token found in environment")
	}

	client := github.NewClient(nil).WithAuthToken(token)

	allRepos, err := fetchOrgRepos(ctx, client, organization)
	if err != nil {
		return err
	}

	nonArchivedRepos := filterNonArchived(allRepos)
	nonArchivedCount := len(nonArchivedRepos)

	fmt.Printf("Found %d repositories in organization %s (%d non-archived)\n",
		len(allRepos), organization, nonArchivedCount)

	if nonArchivedCount == 0 {
		fmt.Println("No non-archived repositories found. Nothing to process.")
		return nil
	}

	fmt.Printf("Processing repositories with %d parallel workers\n", parallelLimit)

	jobs := make(chan *github.Repository, nonArchivedCount)
	results := make(chan repoResult, nonArchivedCount)

	var wg sync.WaitGroup
	for w := 0; w < parallelLimit; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, jobs, results, absTargetPath, skipUpdate, verbose)
		}()
	}

	for _, repo := range nonArchivedRepos {
		jobs <- repo
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	processedCount := 0
	errorCount := 0
	var authErrors bool

	for result := range results {
		processedCount++

		fmt.Printf("[%d/%d] %s\n", processedCount, nonArchivedCount, result.message)

		if !result.success {
			errorCount++
			if strings.Contains(result.message, "Authentication error detected") {
				authErrors = true
			}
		}
	}

	fmt.Printf("\nSummary: Processed %d/%d non-archived repositories from organization %s\n",
		processedCount, nonArchivedCount, organization)

	if errorCount > 0 {
		fmt.Printf("Encountered %d errors during processing\n", errorCount)

		if authErrors {
			fmt.Printf("\nSome authentication errors were detected. Please verify your setup:\n")
			fmt.Printf("1. SSH setup guide: https://docs.github.com/en/authentication/connecting-to-github-with-ssh\n")
			fmt.Printf("2. Personal access token guide: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token\n")
		}
	} else {
		fmt.Printf("All repositories processed successfully\n")
	}

	fmt.Printf("\nNote: For existing repositories, only 'git fetch --all' was performed.\n")
	fmt.Printf("Local branches were not modified. Use 'git merge' or 'git rebase' manually to update local branches.\n")

	if errorCount > 0 {
		return fmt.Errorf("failed to process %d repositories", errorCount)
	}

	return nil
}
