package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-github/v63/github"
	"github.com/spf13/cobra"
)

// Version information - read from VERSION file at init time
var (
	version string
)

func init() {
	// Read version from file
	versionBytes, err := os.ReadFile("VERSION")
	if err != nil {
		// If VERSION file not found, use development version
		version = "0.0.0-dev"
	} else {
		// Trim whitespace and newlines
		version = strings.TrimSpace(string(versionBytes))
	}
}

var (
	organization   string
	targetPath     string
	skipUpdate     bool
	verbose        bool
	parallelLimit  int
	showVersion    bool
)

// WriteLogger is a helper to safely write to a buffer from multiple goroutines
type WriteLogger struct {
	mutex  *sync.Mutex
	buffer *[]byte
}

func (w *WriteLogger) Write(p []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	*w.buffer = append(*w.buffer, p...)
	return len(p), nil
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
		if showVersion {
			fmt.Printf("github-pokemon version %s\n", version)
			return nil
		}
		return runRootCommand()
	},
	// Check before validation if we're just showing version
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// If showing version, we don't need to check for required flags
		if cmd.Flag("version").Changed {
			showVersion = true
			// Prevent cobra from validating the org and path flags
			cmd.Flags().Set("org", "dummy-value")
			cmd.Flags().Set("path", "dummy-value")
		}
		return nil
	},
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Define command line flags
	rootCmd.Flags().StringVarP(&organization, "org", "o", "", "GitHub organization to fetch repositories from (required)")
	rootCmd.Flags().StringVarP(&targetPath, "path", "p", "", "Local path to clone/update repositories to (required)")
	rootCmd.Flags().BoolVarP(&skipUpdate, "skip-update", "s", false, "Skip updating existing repositories")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().IntVarP(&parallelLimit, "parallel", "j", 5, "Number of repositories to process in parallel")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "V", false, "Show version information and exit")
	
		// Mark flags as required
		rootCmd.MarkFlagRequired("org")
		rootCmd.MarkFlagRequired("path")
}

// worker function for parallel processing
func worker(id int, jobs <-chan *github.Repository, results chan<- struct {
	repoName string
	success  bool
	message  string
}, targetPath string, skipUpdate bool, verbose bool) {

	for repo := range jobs {
		repoName := repo.GetName()
		repoPath := filepath.Join(targetPath, repoName)
	
		// Process repository
		success, message := processRepository(repo, repoPath, skipUpdate, verbose)
	
		// Send result back
		results <- struct {
			repoName string
			success  bool
			message  string
		}{
			repoName: repoName,
			success:  success,
			message:  message,
		}
	}
}

// processRepository handles cloning or fetching for a single repository
func processRepository(repo *github.Repository, repoPath string, skipUpdate bool, verbose bool) (bool, string) {
	repoName := repo.GetName()

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		// Repository doesn't exist locally, clone it
		// Use git command line for cloning
		cloneURL := repo.GetSSHURL() // Use SSH URL instead of HTTPS
		if cloneURL == "" {
			cloneURL = repo.GetCloneURL() // Fallback to HTTPS if SSH URL isn't available
		}
	
		// Create separate stdout/stderr buffers for this goroutine
		var outputBuffer []byte
		var outputMutex sync.Mutex
	
		cmd := exec.Command("git", "clone", cloneURL, repoPath)
		cmd.Stdout = &WriteLogger{&outputMutex, &outputBuffer}
		cmd.Stderr = &WriteLogger{&outputMutex, &outputBuffer}
	
		err := cmd.Run()
		if err != nil {
			errMsg := fmt.Sprintf("Error cloning repository %s: %v\n%s", repoName, err, outputBuffer)
			
			// Provide helpful authentication guidance
			if strings.Contains(errMsg, "authenticity") || strings.Contains(errMsg, "permission denied") || 
			   strings.Contains(errMsg, "could not read Username") || strings.Contains(errMsg, "auth") {
				errMsg += "\n\nAuthentication error detected. Please ensure:\n" +
					"1. Your SSH key is set up correctly with GitHub: https://docs.github.com/en/authentication/connecting-to-github-with-ssh\n" +
					"2. Your GITHUB_TOKEN has sufficient permissions\n" +
					"3. For SSH: Your SSH agent is running ('eval $(ssh-agent -s)')\n" +
					"4. For HTTPS: You may need to configure credential helper ('git config --global credential.helper cache')"
			}
			
			return false, errMsg
		}
	
		return true, fmt.Sprintf("Successfully cloned repository: %s", repoName)
	} else {
		// Repository exists locally
		if skipUpdate {
			if verbose {
				return true, fmt.Sprintf("Skipping update for existing repository: %s", repoName)
			}
			return true, fmt.Sprintf("Repository exists (skipping): %s", repoName)
		}
	
		// Fetch updates to remote tracking branches (without modifying local branches)
		fetchCmd := exec.Command("git", "fetch", "--all")
		fetchCmd.Dir = repoPath
		fetchOutput, err := fetchCmd.CombinedOutput()
	
		if err != nil {
			errMsg := fmt.Sprintf("Warning: Failed to fetch for repository %s: %v\n%s", 
				repoName, err, fetchOutput)
				
			// Provide helpful authentication guidance for fetch errors
			if strings.Contains(string(fetchOutput), "authenticity") || strings.Contains(string(fetchOutput), "permission denied") || 
			   strings.Contains(string(fetchOutput), "could not read Username") || strings.Contains(string(fetchOutput), "auth") {
				errMsg += "\n\nAuthentication error detected. Please ensure:\n" +
					"1. Your SSH key is set up correctly with GitHub: https://docs.github.com/en/authentication/connecting-to-github-with-ssh\n" +
					"2. Your GITHUB_TOKEN has sufficient permissions\n" +
					"3. For SSH: Your SSH agent is running ('eval $(ssh-agent -s)')\n" +
					"4. For HTTPS: You may need to configure credential helper ('git config --global credential.helper cache')"
			}
			
			return false, errMsg
		}
	
		successMsg := fmt.Sprintf("Successfully fetched updates for %s", repoName)
	
		// Optionally show info about how many commits ahead/behind
		if verbose {
			// Get status compared to remote
			statusCmd := exec.Command("git", "status", "-sb")
			statusCmd.Dir = repoPath
			statusOutput, err := statusCmd.CombinedOutput()
			if err == nil {
				successMsg += fmt.Sprintf("\nStatus for %s:\n%s", repoName, statusOutput)
			}
		}
	
		return true, successMsg
	}
}

func runRootCommand() error {
	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}

	// Ensure target path exists
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable not set. Please set it with: export GITHUB_TOKEN=\"your-personal-access-token\"")
	}

	if verbose {
		fmt.Println("GitHub token found in environment")
	}

	// Create GitHub client
	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(token)

	// Get all repositories for the organization
	var allRepos []*github.Repository
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Type:        "all",
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, organization, opt)
		if err != nil {
			return fmt.Errorf("failed to list repositories: %w", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// Count non-archived repositories
	var nonArchivedRepos []*github.Repository
	for _, repo := range allRepos {
		if !repo.GetArchived() {
			nonArchivedRepos = append(nonArchivedRepos, repo)
		}
	}

	nonArchivedCount := len(nonArchivedRepos)
	fmt.Printf("Found %d repositories in organization %s (%d non-archived)\n", 
		len(allRepos), organization, nonArchivedCount)
	
	if nonArchivedCount == 0 {
		fmt.Println("No non-archived repositories found. Nothing to process.")
		return nil
	}

	// Create a channel for parallel processing
	if parallelLimit <= 0 {
		parallelLimit = 5 // Default to 5 if invalid value provided
	}
	
	fmt.Printf("Processing repositories with %d parallel workers\n", parallelLimit)
	
	// Setup channels for worker pool
	jobs := make(chan *github.Repository, nonArchivedCount)
	results := make(chan struct {
		repoName string
		success  bool
		message  string
	}, nonArchivedCount)
	
	// Create workers
	for w := 1; w <= parallelLimit; w++ {
		go worker(w, jobs, results, targetPath, skipUpdate, verbose)
	}
	
	// Send jobs to workers
	for _, repo := range nonArchivedRepos {
		jobs <- repo
	}
	close(jobs)
	
	// Collect results
	processedCount := 0
	errorCount := 0
	var authErrors bool
	
	for i := 0; i < len(nonArchivedRepos); i++ {
		result := <-results
		processedCount++
		
		fmt.Printf("[%d/%d] %s\n", processedCount, nonArchivedCount, result.message)
		
		if !result.success {
			errorCount++
			// Check if this was an authentication error
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
	
	return nil
}