package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = writer

	fn()

	writer.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, reader)
	reader.Close()

	return buf.String()
}

func TestLogEventJSON(t *testing.T) {
	originalFormat := logFormat
	logFormat = logFormatJSON
	defer func() { logFormat = originalFormat }()

	output := captureStdout(t, func() {
		logEvent("download.summary", map[string]any{
			"downloaded": 2,
			"skipped":    1,
			"failed":     0,
		})
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if payload["event"] != "download.summary" {
		t.Fatalf("expected event download.summary, got %v", payload["event"])
	}
	if payload["downloaded"] != float64(2) {
		t.Fatalf("expected downloaded 2, got %v", payload["downloaded"])
	}
}
