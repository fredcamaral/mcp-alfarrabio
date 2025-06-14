package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
	PublishedAt time.Time `json:"published_at"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
}

const (
	githubAPIURL = "https://api.github.com/repos/lerianstudio/lerian-mcp-memory/releases"
	repoURL      = "lerianstudio/lerian-mcp-memory"
)

// createUpdateCommand creates the 'update' command
func (c *CLI) createUpdateCommand() *cobra.Command {
	var (
		force      bool
		prerelease bool
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update lmmc to the latest version",
		Long: `Check for and install the latest version of lmmc.

The update command will:
1. Check GitHub for the latest release
2. Compare with current version
3. Download and install the new version if available

Examples:
  lmmc update                    # Update to latest stable release
  lmmc update --force            # Force update even if same version
  lmmc update --prerelease       # Include pre-release versions
  lmmc update --dry-run          # Check for updates without installing`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runUpdate(force, prerelease, dryRun)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force update even if current version is latest")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include pre-release versions")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Check for updates without installing")

	return cmd
}

// runUpdate performs the update operation
func (c *CLI) runUpdate(force, prerelease, dryRun bool) error {
	fmt.Println("🔍 Checking for updates...")

	// Get current version
	currentVersion := Version
	if currentVersion == "dev" {
		if !force {
			fmt.Println("⚠️  Development build detected. Use --force to update anyway.")
			return nil
		}
		currentVersion = "v0.0.0" // Treat dev as very old version
	}

	// Get latest release
	release, err := c.getLatestRelease(prerelease)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	fmt.Printf("📦 Latest release: %s\n", release.TagName)
	fmt.Printf("📅 Published: %s\n", release.PublishedAt.Format("2006-01-02 15:04:05"))

	// Compare versions
	if !force && !c.needsUpdate(currentVersion, release.TagName) {
		fmt.Println("✅ You're already running the latest version!")
		return nil
	}

	if dryRun {
		fmt.Printf("🔄 Update available: %s -> %s\n", currentVersion, release.TagName)
		fmt.Println("📝 Run without --dry-run to install the update")
		return nil
	}

	// Find appropriate asset for current platform
	asset := c.findAssetForPlatform(release)
	if asset == nil {
		return fmt.Errorf("no compatible binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("⬇️  Downloading %s (%d bytes)...\n", asset.Name, asset.Size)

	// Download and install
	if err := c.downloadAndInstall(asset.BrowserDownloadURL, asset.Name); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	fmt.Printf("✅ Successfully updated to %s!\n", release.TagName)
	fmt.Println("🎉 Restart your terminal to use the new version")

	return nil
}

// getLatestRelease fetches the latest release from GitHub
func (c *CLI) getLatestRelease(includePrerelease bool) (*GitHubRelease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url := githubAPIURL + "/latest"
	if includePrerelease {
		url = githubAPIURL + "?per_page=1"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("lmmc/%s", Version))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if includePrerelease {
		var releases []GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, err
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("no releases found")
		}
		release = releases[0]
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, err
		}
	}

	return &release, nil
}

// needsUpdate compares version strings to determine if update is needed
func (c *CLI) needsUpdate(current, latest string) bool {
	// Simple version comparison - in production you'd want semantic versioning
	// For now, just compare strings (works for tags like v1.0.0, v1.0.1, etc.)
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	return current != latest
}

// findAssetForPlatform finds the appropriate binary asset for the current platform
func (c *CLI) findAssetForPlatform(release *GitHubRelease) *struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
} {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, platform) {
			return &asset
		}
	}

	return nil
}

// downloadAndInstall downloads and installs the new binary
func (c *CLI) downloadAndInstall(downloadURL, filename string) error {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "lmmc-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Download file
	tempFile := filepath.Join(tempDir, filename)
	if err := c.downloadFile(downloadURL, tempFile); err != nil {
		return err
	}

	// Extract binary
	binaryPath, err := c.extractBinary(tempFile, tempDir)
	if err != nil {
		return err
	}

	// Get current executable path
	currentExec, err := os.Executable()
	if err != nil {
		return err
	}

	// Create backup of current binary
	backupPath := currentExec + ".backup"
	if err := c.copyFile(currentExec, backupPath); err != nil {
		return err
	}

	// Replace current binary
	if err := c.copyFile(binaryPath, currentExec); err != nil {
		// Restore backup on failure
		_ = os.Rename(backupPath, currentExec)
		return err
	}

	// Make executable with secure permissions
	// #nosec G302 -- Owner-only executable permissions are secure for binary updates
	if err := os.Chmod(currentExec, 0o700); err != nil {
		return err
	}

	// Remove backup on success
	_ = os.Remove(backupPath)

	return nil
}

// downloadFile downloads a file from URL to local path
func (c *CLI) downloadFile(url, filepath string) error {
	// Clean and validate the file path
	filepath = filepath.Clean(filepath)
	if strings.Contains(filepath, "..") {
		return fmt.Errorf("path traversal detected: %s", filepath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractBinary extracts the binary from archive
func (c *CLI) extractBinary(archivePath, extractDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return c.extractFromZip(archivePath, extractDir)
	} else if strings.HasSuffix(archivePath, ".tar.gz") {
		return c.extractFromTarGz(archivePath, extractDir)
	}

	return "", fmt.Errorf("unsupported archive format")
}

// extractFromZip extracts binary from ZIP archive
func (c *CLI) extractFromZip(zipPath, extractDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.Contains(f.Name, "lmmc") && !strings.Contains(f.Name, "/") {
			return c.extractZipFile(f, extractDir)
		}
	}

	return "", fmt.Errorf("binary not found in archive")
}

// extractZipFile extracts a single file from ZIP
func (c *CLI) extractZipFile(f *zip.File, extractDir string) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	// Validate file path to prevent directory traversal
	cleanName := filepath.Clean(f.Name)
	if strings.Contains(cleanName, "..") || filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("unsafe file path: %s", f.Name)
	}

	path := filepath.Join(extractDir, cleanName)
	outFile, err := os.Create(path) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	// Limit extraction size to prevent decompression bombs (100MB limit)
	limitedReader := io.LimitReader(rc, 100*1024*1024)
	_, err = io.Copy(outFile, limitedReader)
	if err != nil {
		return "", err
	}

	return path, nil
}

// extractFromTarGz extracts binary from tar.gz archive
func (c *CLI) extractFromTarGz(tarPath, extractDir string) (string, error) {
	// Clean and validate the tar path
	tarPath = filepath.Clean(tarPath)
	if strings.Contains(tarPath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", tarPath)
	}

	f, err := os.Open(tarPath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return "", err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if strings.Contains(header.Name, "lmmc") && !strings.Contains(header.Name, "/") {
			// Validate file path to prevent directory traversal
			cleanName := filepath.Clean(header.Name)
			if strings.Contains(cleanName, "..") || filepath.IsAbs(cleanName) {
				return "", fmt.Errorf("unsafe file path: %s", header.Name)
			}

			path := filepath.Join(extractDir, cleanName)
			outFile, err := os.Create(path) // #nosec G304 -- Path is cleaned and validated above
			if err != nil {
				return "", err
			}
			defer outFile.Close()

			// Limit extraction size to prevent decompression bombs (100MB limit)
			limitedReader := io.LimitReader(tr, 100*1024*1024)
			_, err = io.Copy(outFile, limitedReader)
			if err != nil {
				return "", err
			}

			return path, nil
		}
	}

	return "", fmt.Errorf("binary not found in archive")
}

// copyFile copies a file from src to dst
func (c *CLI) copyFile(src, dst string) error {
	// Clean and validate the file paths
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	if strings.Contains(src, "..") || strings.Contains(dst, "..") {
		return fmt.Errorf("path traversal detected: src=%s, dst=%s", src, dst)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
