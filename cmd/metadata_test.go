package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
)

func TestSidecarPath(t *testing.T) {
	path := sidecarPath(filepath.Join("out", "20230101_000000_post.html"))
	expected := filepath.Join("out", "20230101_000000_post.json")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}

func TestWriteMetadataSidecar(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "20230101_000000_post.html")
	post := lib.Post{
		Slug:         "post",
		Title:        "Post Title",
		CanonicalUrl: "https://example.substack.com/p/post",
		PostDate:     "2023-01-01T00:00:00Z",
	}
	downloadedAt := time.Date(2023, time.January, 2, 3, 4, 5, 0, time.UTC)

	writeMetadataSidecar(post, outputPath, tempDir, "html", downloadedAt, "2023-01-03")

	metadataPath := sidecarPath(outputPath)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("failed to read metadata: %v", err)
	}

	var meta postMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("failed to parse metadata: %v", err)
	}
	if meta.Slug != "post" || meta.Format != "html" {
		t.Fatalf("unexpected metadata values: %+v", meta)
	}
	if meta.LastModified != "2023-01-03" {
		t.Fatalf("expected last_modified to be set, got %q", meta.LastModified)
	}
}
