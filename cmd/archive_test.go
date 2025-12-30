package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
	"github.com/stretchr/testify/require"
)

func TestBuildArchiveEntries(t *testing.T) {
	tempDir := t.TempDir()

	firstPath := filepath.Join(tempDir, "20230101_000000_first.html")
	require.NoError(t, os.WriteFile(firstPath, []byte("content"), 0644))

	firstPost := lib.Post{
		Slug:         "first",
		Title:        "First Title",
		CanonicalUrl: "https://example.substack.com/p/first",
		PostDate:     "2023-01-01T00:00:00Z",
	}
	downloadedAt := time.Date(2023, time.January, 2, 3, 4, 5, 0, time.UTC)
	writeMetadataSidecar(firstPost, firstPath, tempDir, "html", downloadedAt, "2023-01-01")

	manifest := lib.NewManifest()
	require.NoError(t, manifest.UpdateEntry(firstPost.CanonicalUrl, firstPath, tempDir, "html", downloadedAt, "2023-01-01"))
	require.NoError(t, manifest.Save(filepath.Join(tempDir, lib.ManifestFilename)))

	secondPath := filepath.Join(tempDir, "20230201_000000_second.html")
	require.NoError(t, os.WriteFile(secondPath, []byte("content"), 0644))

	entries, err := buildArchiveEntries(tempDir, "html")
	require.NoError(t, err)
	require.Len(t, entries, 2)

	var firstEntry *archiveEntryData
	var secondEntry *archiveEntryData
	for i := range entries {
		switch entries[i].Path {
		case firstPath:
			firstEntry = &entries[i]
		case secondPath:
			secondEntry = &entries[i]
		}
	}

	require.NotNil(t, firstEntry)
	require.Equal(t, "First Title", firstEntry.Post.Title)
	require.Equal(t, "first", firstEntry.Post.Slug)
	require.Equal(t, firstPost.CanonicalUrl, firstEntry.Post.CanonicalUrl)

	require.NotNil(t, secondEntry)
	require.Equal(t, "second", secondEntry.Post.Title)
	require.Equal(t, "second", secondEntry.Post.Slug)
	require.NotEmpty(t, secondEntry.Post.PostDate)
}
