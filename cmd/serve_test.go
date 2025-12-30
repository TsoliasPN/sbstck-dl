package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
