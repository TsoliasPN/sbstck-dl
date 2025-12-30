package cmd

import (
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
