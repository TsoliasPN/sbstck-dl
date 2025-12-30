package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCookieFromJar(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "cookies.txt")
	content := "# Netscape HTTP Cookie File\n" +
		".example.substack.com\tTRUE\t/\tTRUE\t2145916800\tsubstack.sid\tsubstack-value\n" +
		".example.substack.com\tTRUE\t/\tTRUE\t2145916800\tconnect.sid\tconnect-value\n" +
		".other.example.com\tTRUE\t/\tTRUE\t2145916800\tsubstack.sid\tother-value\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write cookie jar failed: %v", err)
	}

	name, value, err := readCookieFromJar(path, substackSid, "example.substack.com")
	if err != nil {
		t.Fatalf("readCookieFromJar returned error: %v", err)
	}
	if name != substackSid || value != "substack-value" {
		t.Fatalf("expected substack cookie, got %s=%s", name, value)
	}

	name, value, err = readCookieFromJar(path, "", "example.substack.com")
	if err != nil {
		t.Fatalf("readCookieFromJar returned error: %v", err)
	}
	if name != substackSid || value != "substack-value" {
		t.Fatalf("expected default substack cookie, got %s=%s", name, value)
	}
}

func TestReadCookieFromJarNoMatch(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "cookies.txt")
	content := "# Netscape HTTP Cookie File\n" +
		".example.substack.com\tTRUE\t/\tTRUE\t2145916800\tother.cookie\tvalue\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write cookie jar failed: %v", err)
	}

	_, _, err := readCookieFromJar(path, "", "example.substack.com")
	if err == nil {
		t.Fatalf("expected error for missing cookie")
	}
}
