package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
)

func TestNormalizeNotionURL(t *testing.T) {
	normalized, ok := normalizeNotionURL("https://www.notion.so/Workspace/Page-123?utm_source=x#section")
	if !ok {
		t.Fatalf("expected notion URL to be valid")
	}
	if normalized != "https://notion.so/Workspace/Page-123" {
		t.Fatalf("unexpected normalization: %s", normalized)
	}

	normalized, ok = normalizeNotionURL("https://example.notion.site/Page-123/")
	if !ok {
		t.Fatalf("expected notion.site URL to be valid")
	}
	if normalized != "https://example.notion.site/Page-123" {
		t.Fatalf("unexpected normalization: %s", normalized)
	}

	if _, ok = normalizeNotionURL("https://example.com"); ok {
		t.Fatalf("expected non-notion URL to be rejected")
	}
}

func TestExtractNotionLinksMarkdown(t *testing.T) {
	content := "See https://www.notion.so/Workspace/Page-123?utm_source=x and https://www.notion.so/Workspace/Page-123."
	links := extractNotionLinks(content, "md")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0] != "https://notion.so/Workspace/Page-123" {
		t.Fatalf("unexpected link: %s", links[0])
	}
}

func TestBuildNotionIndex(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "20230101_000000_post.html")
	content := `<html><body><a href="https://www.notion.so/Workspace/Page-123?utm_source=x">Notion</a></body></html>`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	posts, err := buildNotionIndex(tempDir, "html")
	if err != nil {
		t.Fatalf("buildNotionIndex error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if len(posts[0].Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(posts[0].Links))
	}
}

func TestBuildNotionIndexUsesSidecar(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "20230101_000000_post.html")
	if err := os.WriteFile(path, []byte("<html><body><p>No links here.</p></body></html>"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	post := lib.Post{
		Slug:         "post",
		Title:        "Sidecar Post",
		CanonicalUrl: "https://example.substack.com/p/post",
		PostDate:     "2023-01-01T00:00:00Z",
		BodyHTML:     `<a href="https://www.notion.so/Workspace/Page-123?utm_source=x">Notion</a>`,
	}
	writeMetadataSidecar(post, path, tempDir, "html", time.Now(), "")

	posts, err := buildNotionIndex(tempDir, "html")
	if err != nil {
		t.Fatalf("buildNotionIndex error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if len(posts[0].Links) != 1 || posts[0].Links[0] != "https://notion.so/Workspace/Page-123" {
		t.Fatalf("unexpected links: %+v", posts[0].Links)
	}
}

func TestWriteNotionIndexMarkdownWithLabels(t *testing.T) {
	tempDir := t.TempDir()
	posts := []notionPostLinks{
		{
			Title:   "Post",
			RelPath: "post.html",
			Links:   []string{"https://notion.so/Workspace/Page-123"},
		},
	}
	labels := map[string]string{
		"https://notion.so/Workspace/Page-123": "Project Plan",
	}

	if err := writeNotionIndexMarkdown(tempDir, posts, labels); err != nil {
		t.Fatalf("writeNotionIndexMarkdown error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tempDir, notionLinksMD))
	if err != nil {
		t.Fatalf("read output error: %v", err)
	}
	if !strings.Contains(string(content), "- [Project Plan](https://notion.so/Workspace/Page-123)") {
		t.Fatalf("expected label in markdown output")
	}
}

func TestWriteNotionIndexHTMLWithLabels(t *testing.T) {
	tempDir := t.TempDir()
	posts := []notionPostLinks{
		{
			Title:   "Post",
			RelPath: "post.html",
			Links:   []string{"https://notion.so/Workspace/Page-123"},
		},
	}
	labels := map[string]string{
		"https://notion.so/Workspace/Page-123": "Project Plan",
	}

	if err := writeNotionIndexHTML(tempDir, posts, labels); err != nil {
		t.Fatalf("writeNotionIndexHTML error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tempDir, notionLinksHTML))
	if err != nil {
		t.Fatalf("read output error: %v", err)
	}
	if !strings.Contains(string(content), ">Project Plan<") {
		t.Fatalf("expected label in HTML output")
	}
}
