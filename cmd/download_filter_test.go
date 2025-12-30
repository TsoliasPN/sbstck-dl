package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
	"github.com/stretchr/testify/require"
)

func TestFilterExistingPostsManifestSkips(t *testing.T) {
	outputDir := t.TempDir()
	url := "https://example.substack.com/p/test-post"
	filePath := filepath.Join(outputDir, "20240101_000000_test-post.html")

	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

	manifest := lib.NewManifest()
	require.NoError(t, manifest.UpdateEntry(url, filePath, outputDir, "html", time.Now(), ""))
	require.NoError(t, manifest.Save(filepath.Join(outputDir, lib.ManifestFilename)))

	filtered, err := filterExistingPosts([]string{url}, outputDir, "html")
	require.NoError(t, err)
	require.Len(t, filtered, 0)
}

func TestFilterExistingPostsManifestFormatMismatch(t *testing.T) {
	outputDir := t.TempDir()
	url := "https://example.substack.com/p/test-post"
	filePath := filepath.Join(outputDir, "20240101_000000_test-post.md")

	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

	manifest := lib.NewManifest()
	require.NoError(t, manifest.UpdateEntry(url, filePath, outputDir, "md", time.Now(), ""))
	require.NoError(t, manifest.Save(filepath.Join(outputDir, lib.ManifestFilename)))

	filtered, err := filterExistingPosts([]string{url}, outputDir, "html")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, url, filtered[0])
}

func TestFilterExistingPostsManifestMissingFile(t *testing.T) {
	outputDir := t.TempDir()
	url := "https://example.substack.com/p/test-post"
	filePath := filepath.Join(outputDir, "20240101_000000_test-post.html")

	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

	manifest := lib.NewManifest()
	require.NoError(t, manifest.UpdateEntry(url, filePath, outputDir, "html", time.Now(), ""))
	require.NoError(t, manifest.Save(filepath.Join(outputDir, lib.ManifestFilename)))

	require.NoError(t, os.Remove(filePath))

	filtered, err := filterExistingPosts([]string{url}, outputDir, "html")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, url, filtered[0])
}

func TestFilterExistingPostsSlugFallback(t *testing.T) {
	outputDir := t.TempDir()
	url := "https://example.substack.com/p/test-post"
	filePath := filepath.Join(outputDir, "20240101_000000_test-post.html")

	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

	filtered, err := filterExistingPosts([]string{url}, outputDir, "html")
	require.NoError(t, err)
	require.Len(t, filtered, 0)
}
