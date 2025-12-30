package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFailedURLs(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "nested")
	urls := []string{
		"https://example.substack.com/p/first",
		"https://example.substack.com/p/second",
	}

	path, err := writeFailedURLs(outputDir, urls)
	if err != nil {
		t.Fatalf("writeFailedURLs returned error: %v", err)
	}

	if path == "" {
		t.Fatalf("expected non-empty file path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed URLs file: %v", err)
	}

	content := string(data)
	expected := strings.Join(urls, "\n") + "\n"
	if content != expected {
		t.Fatalf("expected content %q, got %q", expected, content)
	}
}
