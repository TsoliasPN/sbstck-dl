package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadManifestMissingFile(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, ManifestFilename)

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if manifest.Version != ManifestVersion {
		t.Fatalf("expected version %d, got %d", ManifestVersion, manifest.Version)
	}
	if len(manifest.Entries) != 0 {
		t.Fatalf("expected empty entries, got %d", len(manifest.Entries))
	}
}

func TestManifestUpdateSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	postDir := filepath.Join(dir, "posts")
	if err := os.MkdirAll(postDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	filePath := filepath.Join(postDir, "post.html")
	content := []byte("hello manifest")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	manifestPath := filepath.Join(dir, ManifestFilename)
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	downloadedAt := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	canonical := "https://example.com/p/test"
	if err := manifest.UpdateEntry(canonical, filePath, dir, "html", downloadedAt); err != nil {
		t.Fatalf("UpdateEntry returned error: %v", err)
	}
	if err := manifest.Save(manifestPath); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	reloaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	entry, ok := reloaded.Entries[canonical]
	if !ok {
		t.Fatalf("expected entry for %s", canonical)
	}
	expectedRel := filepath.ToSlash(filepath.Join("posts", "post.html"))
	if entry.FilePath != expectedRel {
		t.Fatalf("expected file path %q, got %q", expectedRel, entry.FilePath)
	}
	expectedDownloadedAt := downloadedAt.Format(time.RFC3339)
	if entry.DownloadedAt != expectedDownloadedAt {
		t.Fatalf("expected downloaded_at %q, got %q", expectedDownloadedAt, entry.DownloadedAt)
	}
	expectedHash := sha256.Sum256(content)
	if entry.ContentHash != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("expected content hash %q, got %q", hex.EncodeToString(expectedHash[:]), entry.ContentHash)
	}
	if entry.Format != "html" {
		t.Fatalf("expected format %q, got %q", "html", entry.Format)
	}
	if reloaded.Version != ManifestVersion {
		t.Fatalf("expected version %d, got %d", ManifestVersion, reloaded.Version)
	}
}

func TestHashFileSHA256(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "hash.txt")
	content := []byte("hash me")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	got, err := HashFileSHA256(filePath)
	if err != nil {
		t.Fatalf("HashFileSHA256 returned error: %v", err)
	}

	expected := sha256.Sum256(content)
	if got != hex.EncodeToString(expected[:]) {
		t.Fatalf("expected hash %q, got %q", hex.EncodeToString(expected[:]), got)
	}
}
