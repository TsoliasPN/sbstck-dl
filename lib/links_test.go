package lib

import (
	"testing"
)

func TestExtractLinkDomainsHTML(t *testing.T) {
	content := `<html><body>
<a href="https://www.notion.so/Workspace/Page-123">Notion</a>
<a href="https://github.com/org/repo">GitHub</a>
<a href="/relative/path">Relative</a>
</body></html>`
	domains := ExtractLinkDomains(content, "html")
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0] != "github.com" || domains[1] != "notion.so" {
		t.Fatalf("unexpected domains: %+v", domains)
	}
}

func TestExtractLinkDomainsMarkdown(t *testing.T) {
	content := "See https://docs.google.com/spreadsheets/d/abc and https://example.com."
	domains := ExtractLinkDomains(content, "md")
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0] != "docs.google.com" || domains[1] != "example.com" {
		t.Fatalf("unexpected domains: %+v", domains)
	}
}
