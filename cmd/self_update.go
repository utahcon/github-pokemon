package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/spf13/cobra"
)

const (
	// maxDownloadSize bounds every downloaded or extracted file.
	maxDownloadSize = 100 << 20
	checksumsAsset  = "checksums.txt"
)

var selfUpdateCheckOnly bool

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update github-pokemon to the latest release",
	Long: `Checks GitHub for the latest release and, if it is newer than the running
version, downloads the archive for this OS/architecture, verifies it against the
release's checksums.txt, and replaces the running binary in place.

Use --check to only report whether an update is available.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSelfUpdate(cmd.Context())
	},
	SilenceUsage: true,
}

func init() {
	selfUpdateCmd.Flags().BoolVar(&selfUpdateCheckOnly, "check", false, "Only check whether a newer version is available")
	rootCmd.AddCommand(selfUpdateCmd)
}

func runSelfUpdate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	client := github.NewClient(nil)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = client.WithAuthToken(token)
	}

	release, _, err := client.Repositories.GetLatestRelease(ctx, repoOwner, repoName)
	if err != nil {
		return fmt.Errorf("fetching latest release: %w", err)
	}

	latest := strings.TrimPrefix(release.GetTagName(), "v")
	if !isNewer(latest, version) {
		fmt.Printf("github-pokemon v%s is up to date\n", version)
		return nil
	}

	fmt.Printf("New version available: v%s (current: v%s)\n", latest, version)
	if selfUpdateCheckOnly {
		fmt.Println("Run 'github-pokemon self-update' to install it.")
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current executable: %w", err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return fmt.Errorf("resolving current executable: %w", err)
	}

	archiveName := releaseArchiveName(latest, runtime.GOOS, runtime.GOARCH)
	archiveAsset := findAsset(release, archiveName)
	if archiveAsset == nil {
		return fmt.Errorf("release v%s has no asset %q for %s/%s", latest, archiveName, runtime.GOOS, runtime.GOARCH)
	}
	sumsAsset := findAsset(release, checksumsAsset)
	if sumsAsset == nil {
		return fmt.Errorf("release v%s has no %s; refusing to install an unverified binary", latest, checksumsAsset)
	}

	fmt.Printf("Downloading %s...\n", archiveName)
	archive, err := downloadAsset(ctx, client, archiveAsset)
	if err != nil {
		return err
	}
	sums, err := downloadAsset(ctx, client, sumsAsset)
	if err != nil {
		return err
	}

	if err := verifyChecksum(sums, archiveName, archive); err != nil {
		return err
	}
	fmt.Println("Checksum verified.")

	binary, err := extractBinary(archive, archiveName, binaryName(runtime.GOOS))
	if err != nil {
		return err
	}

	if err := replaceExecutable(exe, binary, runtime.GOOS); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("%w\nYou may not have write access to %s; re-run with sufficient privileges (e.g. sudo) or download manually: https://github.com/%s/%s/releases/latest", err, filepath.Dir(exe), repoOwner, repoName)
		}
		return err
	}

	fmt.Printf("Updated %s to v%s\n", exe, latest)
	return nil
}

// releaseArchiveName mirrors the GoReleaser archive name_template.
func releaseArchiveName(ver, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s_%s.%s", repoName, ver, goos, goarch, ext)
}

func binaryName(goos string) string {
	if goos == "windows" {
		return repoName + ".exe"
	}
	return repoName
}

func findAsset(release *github.RepositoryRelease, name string) *github.ReleaseAsset {
	for _, a := range release.Assets {
		if a.GetName() == name {
			return a
		}
	}
	return nil
}

func downloadAsset(ctx context.Context, client *github.Client, asset *github.ReleaseAsset) ([]byte, error) {
	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(ctx, repoOwner, repoName, asset.GetID(), http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", asset.GetName(), err)
	}
	if rc == nil {
		return nil, fmt.Errorf("downloading %s: unexpected redirect to %s", asset.GetName(), redirectURL)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(io.LimitReader(rc, maxDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", asset.GetName(), err)
	}
	if len(data) > maxDownloadSize {
		return nil, fmt.Errorf("%s exceeds maximum size of %d bytes", asset.GetName(), maxDownloadSize)
	}
	return data, nil
}

// verifyChecksum checks data against the sha256 entry for name in a
// checksums.txt body ("<hex>  <filename>" per line).
func verifyChecksum(checksums []byte, name string, data []byte) error {
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		sum := sha256.Sum256(data)
		if !strings.EqualFold(fields[0], hex.EncodeToString(sum[:])) {
			return fmt.Errorf("checksum mismatch for %s", name)
		}
		return nil
	}
	return fmt.Errorf("no checksum listed for %s", name)
}

// extractBinary returns the contents of the file named binName from a
// .tar.gz or .zip archive.
func extractBinary(archive []byte, archiveName, binName string) ([]byte, error) {
	if strings.HasSuffix(archiveName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, fmt.Errorf("opening zip archive: %w", err)
		}
		for _, f := range zr.File {
			if f.FileInfo().IsDir() || path.Base(f.Name) != binName {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening %s in archive: %w", f.Name, err)
			}
			defer func() { _ = rc.Close() }()
			return readLimited(rc, binName)
		}
		return nil, fmt.Errorf("%s not found in archive", binName)
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("opening gzip archive: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s not found in archive", binName)
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar archive: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && path.Base(hdr.Name) == binName {
			return readLimited(tr, binName)
		}
	}
}

func readLimited(r io.Reader, name string) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("extracting %s: %w", name, err)
	}
	if len(data) > maxDownloadSize {
		return nil, fmt.Errorf("%s exceeds maximum size of %d bytes", name, maxDownloadSize)
	}
	return data, nil
}

// replaceExecutable atomically swaps the file at exe with newBin. The new
// file is written next to exe so the final rename stays on one filesystem.
// On Windows a running executable can't be overwritten, but it can be
// renamed, so the old binary is moved aside first.
func replaceExecutable(exe string, newBin []byte, goos string) error {
	info, err := os.Stat(exe)
	if err != nil {
		return fmt.Errorf("inspecting %s: %w", exe, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(exe), "."+filepath.Base(exe)+".new-*")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(newBin); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("writing new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("writing new binary: %w", err)
	}
	if err := os.Chmod(tmpName, info.Mode().Perm()|0o111); err != nil {
		cleanup()
		return fmt.Errorf("setting permissions: %w", err)
	}

	if goos == "windows" {
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			cleanup()
			return fmt.Errorf("moving current binary aside: %w", err)
		}
		if err := os.Rename(tmpName, exe); err != nil {
			_ = os.Rename(old, exe)
			cleanup()
			return fmt.Errorf("installing new binary: %w", err)
		}
		return nil
	}

	if err := os.Rename(tmpName, exe); err != nil {
		cleanup()
		return fmt.Errorf("installing new binary: %w", err)
	}
	return nil
}
