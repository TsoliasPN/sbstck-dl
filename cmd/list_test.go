package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferrari88/sbstck-dl/lib"
)

func TestBuildListOutputNoMetadata(t *testing.T) {
	extractor = lib.NewExtractor(lib.NewFetcher())
	ctx = context.Background()

	urls := []string{
		"https://example.substack.com/p/first",
		"https://example.substack.com/p/second",
	}

	output, err := buildListOutput(ctx, urls, false)
	if err != nil {
		t.Fatalf("build output: %v", err)
	}
	if len(output.Posts) != len(urls) {
		t.Fatalf("unexpected length: %d", len(output.Posts))
	}
	for i, url := range urls {
		if output.Posts[i].URL != url {
			t.Fatalf("unexpected url at %d: %s", i, output.Posts[i].URL)
		}
		if output.Posts[i].Title != "" || output.Posts[i].Date != "" {
			t.Fatalf("unexpected metadata for %s", url)
		}
	}
}

func TestBuildListOutputWithMetadata(t *testing.T) {
	mockPost := lib.Post{
		Id:           123,
		Title:        "Test Post",
		Slug:         "test-post",
		PostDate:     "2023-01-01",
		BodyHTML:     "<p>This is a test post</p>",
		CanonicalUrl: "https://example.substack.com/p/test-post",
	}

	postWrapper := lib.PostWrapper{Post: mockPost}
	jsonBytes, _ := json.Marshal(postWrapper)
	escapedJSON := strings.ReplaceAll(string(jsonBytes), `"`, `\"`)
	mockHTML := fmt.Sprintf(`<!DOCTYPE html><html><body><script>window._preloads = JSON.parse("%s")</script></body></html>`, escapedJSON)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/p/test-post" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(mockHTML))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher = lib.NewFetcher()
	extractor = lib.NewExtractor(fetcher)
	ctx = context.Background()

	urls := []string{server.URL + "/p/test-post"}
	output, err := buildListOutput(ctx, urls, true)
	if err != nil {
		t.Fatalf("build output: %v", err)
	}
	if len(output.Posts) != 1 {
		t.Fatalf("unexpected length: %d", len(output.Posts))
	}
	if output.Posts[0].Title != "Test Post" {
		t.Fatalf("unexpected title: %s", output.Posts[0].Title)
	}
	if output.Posts[0].Date != "2023-01-01" {
		t.Fatalf("unexpected date: %s", output.Posts[0].Date)
	}
}
