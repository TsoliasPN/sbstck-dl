package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
)

func TestIsEntryUpdated(t *testing.T) {
	entry := lib.ManifestEntry{
		LastModified: "2023-01-01",
		DownloadedAt: "2023-01-01T00:00:00Z",
	}

	if !isEntryUpdated(entry, "2023-01-02") {
		t.Fatalf("expected entry to be updated")
	}
	if isEntryUpdated(entry, "2023-01-01") {
		t.Fatalf("expected entry to not be updated")
	}
}

func TestFilterEntriesForDownloadRefreshUpdated(t *testing.T) {
	outputDir := t.TempDir()
	url := "https://example.substack.com/p/test-post"
	filePath := filepath.Join(outputDir, "20230101_000000_test-post.html")

	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	manifest := lib.NewManifest()
	downloadedAt := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := manifest.UpdateEntry(url, filePath, outputDir, "html", downloadedAt, "2023-01-01"); err != nil {
		t.Fatalf("UpdateEntry returned error: %v", err)
	}

	entries := []lib.SitemapEntry{
		{URL: url, LastMod: "2023-01-02"},
	}

	filtered, skipped, refreshed, err := filterEntriesForDownload(entries, outputDir, "html", manifest, true)
	if err != nil {
		t.Fatalf("filterEntriesForDownload returned error: %v", err)
	}
	if len(filtered) != 1 || skipped != 0 || refreshed != 1 {
		t.Fatalf("expected refresh to include entry (filtered=%d skipped=%d refreshed=%d)", len(filtered), skipped, refreshed)
	}

	filtered, skipped, refreshed, err = filterEntriesForDownload(entries, outputDir, "html", manifest, false)
	if err != nil {
		t.Fatalf("filterEntriesForDownload returned error: %v", err)
	}
	if len(filtered) != 0 || skipped != 1 || refreshed != 0 {
		t.Fatalf("expected skip when refresh is disabled (filtered=%d skipped=%d refreshed=%d)", len(filtered), skipped, refreshed)
	}
}
