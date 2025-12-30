package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	ManifestFilename = "manifest.json"
	ManifestVersion  = 1
)

type Manifest struct {
	Version   int                      `json:"version"`
	UpdatedAt string                   `json:"updated_at"`
	Entries   map[string]ManifestEntry `json:"entries"`
}

type ManifestEntry struct {
	CanonicalURL string `json:"canonical_url"`
	FilePath     string `json:"file_path"`
	DownloadedAt string `json:"downloaded_at"`
	ContentHash  string `json:"content_hash"`
	Format       string `json:"format"`
}

func NewManifest() *Manifest {
	return &Manifest{
		Version: ManifestVersion,
		Entries: make(map[string]ManifestEntry),
	}
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewManifest(), nil
		}
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Version == 0 {
		manifest.Version = ManifestVersion
	}
	if manifest.Entries == nil {
		manifest.Entries = make(map[string]ManifestEntry)
	}
	return &manifest, nil
}

func (m *Manifest) UpdateEntry(canonicalURL, filePath, outputDir, format string, downloadedAt time.Time) error {
	if canonicalURL == "" {
		return fmt.Errorf("canonical URL is required")
	}
	if m.Entries == nil {
		m.Entries = make(map[string]ManifestEntry)
	}

	relPath, err := filepath.Rel(outputDir, filePath)
	if err != nil {
		relPath = filePath
	}
	relPath = filepath.ToSlash(relPath)

	contentHash, err := HashFileSHA256(filePath)
	if err != nil {
		return err
	}

	m.Entries[canonicalURL] = ManifestEntry{
		CanonicalURL: canonicalURL,
		FilePath:     relPath,
		DownloadedAt: downloadedAt.UTC().Format(time.RFC3339),
		ContentHash:  contentHash,
		Format:       format,
	}

	return nil
}

func (m *Manifest) Save(path string) error {
	if m.Version == 0 {
		m.Version = ManifestVersion
	}
	if m.Entries == nil {
		m.Entries = make(map[string]ManifestEntry)
	}

	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func HashFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
