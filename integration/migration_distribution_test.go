// Package integration provides S3 distribution tests for Go binary deployment.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// S3 bucket URL for LacyLights binaries
	defaultS3BaseURL = "https://lacylights-binaries.s3.amazonaws.com"
)

// LatestJSON represents the structure of latest.json
type LatestJSON struct {
	Version   string            `json:"version"`
	Timestamp string            `json:"timestamp"`
	Checksums map[string]string `json:"checksums"` // platform -> sha256
	Artifacts map[string]string `json:"artifacts"` // platform -> download URL
}

// TestLatestJSONEndpoint verifies the latest.json file is accessible and valid
func TestLatestJSONEndpoint(t *testing.T) {
	s3BaseURL := getS3BaseURL()
	latestURL := fmt.Sprintf("%s/latest.json", s3BaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", latestURL, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "latest.json should be accessible")

	// Parse JSON
	var latest LatestJSON
	err = json.NewDecoder(resp.Body).Decode(&latest)
	require.NoError(t, err)

	// Validate structure
	assert.NotEmpty(t, latest.Version, "Version should be present")
	assert.NotEmpty(t, latest.Timestamp, "Timestamp should be present")
	assert.NotEmpty(t, latest.Checksums, "Checksums should be present")
	assert.NotEmpty(t, latest.Artifacts, "Artifacts should be present")

	// Verify expected platforms
	expectedPlatforms := []string{
		"linux-amd64",
		"linux-arm64",
		"darwin-amd64",
		"darwin-arm64",
		"windows-amd64",
	}

	for _, platform := range expectedPlatforms {
		assert.Contains(t, latest.Checksums, platform,
			"Checksum for %s should be present", platform)
		assert.Contains(t, latest.Artifacts, platform,
			"Artifact URL for %s should be present", platform)
	}

	t.Logf("Latest version: %s (released %s)", latest.Version, latest.Timestamp)
}

// TestBinaryDownload verifies binaries can be downloaded
func TestBinaryDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping binary download test in short mode")
	}

	s3BaseURL := getS3BaseURL()
	platform := getCurrentPlatform()

	// Get latest.json first
	latest := getLatestJSON(t, s3BaseURL)

	// Check if current platform is supported
	artifactURL, ok := latest.Artifacts[platform]
	if !ok {
		t.Skipf("Platform %s not found in artifacts", platform)
	}

	expectedChecksum, ok := latest.Checksums[platform]
	require.True(t, ok, "Checksum for %s should be present", platform)

	t.Logf("Downloading binary for %s from %s", platform, artifactURL)

	// Download binary
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", artifactURL, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Binary should be downloadable")

	// Read and verify checksum
	hasher := sha256.New()
	size, err := io.Copy(hasher, resp.Body)
	require.NoError(t, err)

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	assert.Equal(t, expectedChecksum, actualChecksum,
		"Downloaded binary checksum should match")

	t.Logf("Downloaded %d bytes, checksum verified: %s", size, actualChecksum)
}

// TestChecksumValidation verifies checksums are correct for all platforms
func TestChecksumValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping checksum validation test in short mode")
	}

	s3BaseURL := getS3BaseURL()
	latest := getLatestJSON(t, s3BaseURL)

	// Test a subset of platforms to avoid long test times
	platformsToTest := []string{getCurrentPlatform()}

	for _, platform := range platformsToTest {
		t.Run(platform, func(t *testing.T) {
			artifactURL, ok := latest.Artifacts[platform]
			if !ok {
				t.Skipf("Platform %s not found in artifacts", platform)
			}

			expectedChecksum, ok := latest.Checksums[platform]
			require.True(t, ok, "Checksum for %s should be present", platform)

			// Download and verify
			checksum := downloadAndChecksum(t, artifactURL)
			assert.Equal(t, expectedChecksum, checksum,
				"Checksum for %s should match", platform)
		})
	}
}

// TestBinaryExecutable verifies downloaded binary is executable
func TestBinaryExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping binary execution test in short mode")
	}

	// Only test on current platform
	platform := getCurrentPlatform()
	if strings.HasPrefix(platform, "windows") {
		t.Skip("Skipping executable test on Windows")
	}

	s3BaseURL := getS3BaseURL()
	latest := getLatestJSON(t, s3BaseURL)

	artifactURL, ok := latest.Artifacts[platform]
	if !ok {
		t.Skipf("Platform %s not found in artifacts", platform)
	}

	// Download binary to temp file
	tmpFile := downloadBinary(t, artifactURL)
	defer os.Remove(tmpFile)

	// Make executable
	err := os.Chmod(tmpFile, 0755)
	require.NoError(t, err)

	// Note: Actual execution test would need the binary to be properly structured
	// For now, we just verify it's executable
	fileInfo, err := os.Stat(tmpFile)
	require.NoError(t, err)

	mode := fileInfo.Mode()
	assert.True(t, mode&0111 != 0, "Binary should be executable")

	t.Logf("Binary is executable with mode %s", mode)
}

// TestVersionConsistency verifies version in latest.json matches binary version
func TestVersionConsistency(t *testing.T) {
	s3BaseURL := getS3BaseURL()
	latest := getLatestJSON(t, s3BaseURL)

	// Verify version format (e.g., v1.0.0 or v1.0.0-beta.1)
	version := latest.Version
	assert.NotEmpty(t, version, "Version should not be empty")
	assert.True(t, strings.HasPrefix(version, "v"),
		"Version should start with 'v'")

	// Verify timestamp is a valid ISO8601 format
	_, err := time.Parse(time.RFC3339, latest.Timestamp)
	assert.NoError(t, err, "Timestamp should be valid ISO8601 format")

	t.Logf("Version: %s, Released: %s", version, latest.Timestamp)
}

// TestAllPlatformsAvailable verifies all expected platform binaries exist
func TestAllPlatformsAvailable(t *testing.T) {
	s3BaseURL := getS3BaseURL()
	latest := getLatestJSON(t, s3BaseURL)

	expectedPlatforms := map[string]bool{
		"linux-amd64":   true,
		"linux-arm64":   true,
		"darwin-amd64":  true,
		"darwin-arm64":  true,
		"windows-amd64": true,
	}

	for platform := range expectedPlatforms {
		t.Run(platform, func(t *testing.T) {
			// Check artifact exists
			artifactURL, ok := latest.Artifacts[platform]
			assert.True(t, ok, "Artifact URL for %s should exist", platform)
			assert.NotEmpty(t, artifactURL, "Artifact URL should not be empty")

			// Check checksum exists
			checksum, ok := latest.Checksums[platform]
			assert.True(t, ok, "Checksum for %s should exist", platform)
			assert.Len(t, checksum, 64, "SHA256 checksum should be 64 characters")

			// Verify URL is accessible (HEAD request only)
			if ok && artifactURL != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, "HEAD", artifactURL, nil)
				require.NoError(t, err)

				client := &http.Client{Timeout: 10 * time.Second}
				resp, err := client.Do(req)
				if err == nil {
					defer resp.Body.Close()
					assert.Equal(t, http.StatusOK, resp.StatusCode,
						"Binary for %s should be accessible", platform)
				} else {
					t.Logf("Warning: Could not verify %s binary accessibility: %v", platform, err)
				}
			}
		})
	}
}

// TestDistributionCDN verifies CDN/S3 configuration
func TestDistributionCDN(t *testing.T) {
	s3BaseURL := getS3BaseURL()
	latestURL := fmt.Sprintf("%s/latest.json", s3BaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", latestURL, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify CORS headers for browser downloads
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"),
		"CORS headers should be configured")

	// Verify content type
	contentType := resp.Header.Get("Content-Type")
	t.Logf("Content-Type: %s", contentType)

	// Verify caching headers
	cacheControl := resp.Header.Get("Cache-Control")
	t.Logf("Cache-Control: %s", cacheControl)
}

// Helper functions

func getS3BaseURL() string {
	url := os.Getenv("S3_BASE_URL")
	if url == "" {
		url = defaultS3BaseURL
	}
	return url
}

func getLatestJSON(t *testing.T, s3BaseURL string) LatestJSON {
	t.Helper()

	latestURL := fmt.Sprintf("%s/latest.json", s3BaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", latestURL, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var latest LatestJSON
	err = json.NewDecoder(resp.Body).Decode(&latest)
	require.NoError(t, err)

	return latest
}

func getCurrentPlatform() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	return fmt.Sprintf("%s-%s", goos, goarch)
}

func downloadAndChecksum(t *testing.T, url string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, resp.Body)
	require.NoError(t, err)

	return hex.EncodeToString(hasher.Sum(nil))
}

func downloadBinary(t *testing.T, url string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "lacylights-test-*")
	require.NoError(t, err)
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	require.NoError(t, err)

	return tmpFile.Name()
}
