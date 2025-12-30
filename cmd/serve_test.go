package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
)

func TestServeUIRoot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	serveUIRoot(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected content-type text/html, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "Substack Downloader Wizard") {
		t.Fatalf("expected UI content in response")
	}
	if !strings.Contains(rec.Body.String(), "Recommended defaults") {
		t.Fatalf("expected recommended defaults hint in response")
	}
	if !strings.Contains(rec.Body.String(), "data-error-for=\"url\"") {
		t.Fatalf("expected validation markers in response")
	}
}

func TestServeUIRootNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()

	serveUIRoot(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

type testConnectionResponsePayload struct {
	OK            bool     `json:"ok"`
	SitemapOK     bool     `json:"sitemap_ok"`
	SitemapStatus int      `json:"sitemap_status"`
	PrivateOK     bool     `json:"private_ok"`
	PrivateStatus int      `json:"private_status"`
	Errors        []string `json:"errors"`
}

type previewResponsePayload struct {
	OK         bool     `json:"ok"`
	Total      int      `json:"total"`
	ToDownload int      `json:"to_download"`
	Skipped    int      `json:"skipped"`
	OldestDate string   `json:"oldest_date"`
	NewestDate string   `json:"newest_date"`
	Errors     []string `json:"errors"`
}

func TestServeTestConnectionSuccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap.xml":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<urlset></urlset>"))
		case "/private":
			cookie, err := r.Cookie("substack.sid")
			if err != nil || cookie.Value != "token" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	payload := map[string]string{
		"publication_url": upstream.URL,
		"private_url":     upstream.URL + "/private",
		"cookie_name":     "substack.sid",
		"cookie_val":      "token",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/test-connection", bytes.NewReader(data))
	rec := httptest.NewRecorder()

	serveUIHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp testConnectionResponsePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.OK || !resp.SitemapOK || !resp.PrivateOK {
		t.Fatalf("expected ok response, got %+v", resp)
	}
	if resp.SitemapStatus != http.StatusOK || resp.PrivateStatus != http.StatusOK {
		t.Fatalf("unexpected statuses: %+v", resp)
	}
}

func TestServeTestConnectionInvalidURL(t *testing.T) {
	payload := map[string]string{
		"publication_url": "",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/test-connection", bytes.NewReader(data))
	rec := httptest.NewRecorder()

	serveUIHandler().ServeHTTP(rec, req)

	var resp testConnectionResponsePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.OK {
		t.Fatalf("expected failure response")
	}
	if len(resp.Errors) == 0 {
		t.Fatalf("expected error messages")
	}
}

func TestServePreviewCounts(t *testing.T) {
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sitemap.xml" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>` + upstream.URL + `/p/first</loc>
    <lastmod>2023-01-01</lastmod>
  </url>
  <url>
    <loc>` + upstream.URL + `/p/second</loc>
    <lastmod>2023-02-01</lastmod>
  </url>
</urlset>`))
	}))
	defer upstream.Close()

	tempDir := t.TempDir()
	existingPath := filepath.Join(tempDir, "20230101_000000_first.html")
	if err := os.WriteFile(existingPath, []byte("content"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	manifest := lib.NewManifest()
	if err := manifest.UpdateEntry(upstream.URL+"/p/first", existingPath, tempDir, "html", time.Now(), "2023-01-01"); err != nil {
		t.Fatalf("update manifest: %v", err)
	}
	if err := manifest.Save(filepath.Join(tempDir, lib.ManifestFilename)); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	payload := map[string]any{
		"publication_url": upstream.URL,
		"output":          tempDir,
		"format":          "html",
		"skip_existing":   true,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/preview", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	serveUIHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp previewResponsePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected ok response, got %+v", resp)
	}
	if resp.Total != 2 || resp.ToDownload != 1 || resp.Skipped != 1 {
		t.Fatalf("unexpected counts: %+v", resp)
	}
	if resp.OldestDate != "2023-01-01" || resp.NewestDate != "2023-02-01" {
		t.Fatalf("unexpected dates: %+v", resp)
	}
}
