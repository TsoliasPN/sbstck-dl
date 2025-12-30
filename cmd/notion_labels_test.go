package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNotionLabelMapJSON(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "labels.json")
	data := `{"https://www.notion.so/Workspace/Page-123?utm_source=x":"Project Plan","invalid":"Skip"}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write labels: %v", err)
	}

	labels, err := loadNotionLabelMap(path)
	if err != nil {
		t.Fatalf("loadNotionLabelMap error: %v", err)
	}
	if labels["https://notion.so/Workspace/Page-123"] != "Project Plan" {
		t.Fatalf("expected normalized label, got %+v", labels)
	}
	if _, ok := labels["invalid"]; ok {
		t.Fatalf("expected invalid key to be ignored")
	}
}

func TestLoadNotionLabelMapYAML(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "labels.yaml")
	data := "https://www.notion.so/Workspace/Page-123?utm_source=x: Project Plan\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write labels: %v", err)
	}

	labels, err := loadNotionLabelMap(path)
	if err != nil {
		t.Fatalf("loadNotionLabelMap error: %v", err)
	}
	if labels["https://notion.so/Workspace/Page-123"] != "Project Plan" {
		t.Fatalf("expected normalized label, got %+v", labels)
	}
}
