# Archive Index Page Feature Specification

## 1. Overview

### 1.1 Purpose
Add support for generating organized index pages that link all downloaded posts with their metadata. This feature enables users to create beautiful, browseable archives of their downloaded Substack content with comprehensive post information and navigation.

### 1.2 Success Criteria
- Users can generate archive index pages using command-line flags
- Archive pages are created in matching format (HTML/Markdown/Text) to downloaded posts
- Index pages display comprehensive post metadata including titles, dates, descriptions, and cover images
- Posts are automatically sorted by publication date (newest first)
- Archive pages use relative file paths for maximum portability
- Integration works seamlessly with both single post and bulk downloads
- Archive generation includes comprehensive error handling and validation

### 1.3 Scope Boundaries
**In Scope:**
- Generation of index pages in HTML, Markdown, and Text formats
- Extraction and display of post metadata (title, dates, description, cover image)
- Automatic sorting by publication date with fallback sorting
- Relative path generation for downloaded post links
- Integration with existing CLI infrastructure and output patterns
- Support for both single post downloads and bulk archive downloads

**Out of Scope:**
- Archive page theming or advanced styling customization
- Search functionality within archive pages
- Archive page regeneration from existing files (without re-downloading)
- Multiple archive page formats in a single run
- Archive page pagination for very large collections

## 2. Technical Architecture

### 2.1 Architecture Alignment
This feature follows the established sbstck-dl patterns:
- **Modular Design**: New `Archive` and `ArchiveEntry` structs in existing extractor.go
- **Consistent Interface**: Integration with existing CLI flags and format selection
- **Content Generation**: Similar approach to post content generation with format-specific methods
- **File Operations**: Consistent with existing file writing patterns and directory structures

### 2.2 Core Components

#### 2.2.1 Archive Data Structures
```go
type ArchiveEntry struct {
    Post         Post
    FilePath     string
    DownloadTime time.Time
}

type Archive struct {
    Entries []ArchiveEntry
}
```

#### 2.2.2 Archive Generation Interface
```go
func NewArchive() *Archive
func (a *Archive) AddEntry(post Post, filePath string, downloadTime time.Time)
func (a *Archive) sortEntries()
func (a *Archive) GenerateHTML(outputDir string) error
func (a *Archive) GenerateMarkdown(outputDir string) error
func (a *Archive) GenerateText(outputDir string) error
```

### 2.3 Post Metadata Enhancement

#### 2.3.1 Enhanced Post Structure
Extended the existing `Post` struct with new metadata fields:
```go
type Post struct {
    // ... existing fields
    Subtitle string `json:"subtitle,omitempty"` // NEW: from .subtitle CSS selector
    // CoverImage string - enhanced extraction from og:image meta tag
}
```

#### 2.3.2 Metadata Extraction Strategy
- **Subtitle Extraction**: Parse `.subtitle` CSS selector from post HTML
- **Cover Image Enhancement**: Extract from `og:image` meta property when CoverImage field is empty
- **Graceful Fallbacks**: Use Description field when Subtitle is not available

## 3. Command Line Interface

### 3.1 New CLI Flag

```go
// New flag added to cmd/download.go
var createArchive bool // --create-archive
```

### 3.2 Flag Definition

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--create-archive` | | `false` | Create an archive index page linking all downloaded posts |

### 3.3 Usage Examples

```bash
# Download entire archive and create index page
sbstck-dl download --url https://example.substack.com --create-archive

# Create archive index in Markdown format
sbstck-dl download --url https://example.substack.com --create-archive --format md

# Build archive over time with single posts
sbstck-dl download --url https://example.substack.com/p/post-title --create-archive

# Complete download with all features
sbstck-dl download --url https://example.substack.com --download-images --download-files --create-archive

# Custom directory structure with archive
sbstck-dl download --url https://example.substack.com --create-archive --images-dir assets --files-dir attachments
```

## 4. Implementation Details

### 4.1 Archive Entry Collection

1. **Initialization**: Create Archive instance when `--create-archive` flag is set
2. **Entry Collection**: Add entries during both single post and bulk download flows
3. **Metadata Capture**: Record post details, file path, and download timestamp
4. **Automatic Sorting**: Sort entries by publication date (newest first) on each addition

### 4.2 Archive Generation Formats

#### 4.2.1 HTML Format
- **Styled Output**: Professional styling with CSS embedded in the HTML
- **Post Cards**: Each post displayed as a card with image, title, metadata, and description
- **Responsive Design**: Mobile-friendly layout with flexible containers
- **Cover Images**: Display cover images with proper scaling and alignment
- **File**: `index.html` in output directory root

#### 4.2.2 Markdown Format  
- **Clean Structure**: Headers, links, and metadata in standard Markdown format
- **Image References**: Cover images included as standard Markdown image syntax
- **Metadata Formatting**: Bold formatting for dates and consistent structure
- **File**: `index.md` in output directory root

#### 4.2.3 Text Format
- **Plain Text**: Maximum compatibility with simple text structure
- **Clear Separators**: Consistent formatting with horizontal line separators
- **All Metadata**: Complete information including file paths and descriptions
- **File**: `index.txt` in output directory root

### 4.3 Sorting Algorithm

```go
func (a *Archive) sortEntries() {
    sort.Slice(a.Entries, func(i, j int) bool {
        // Parse post dates and compare (newest first)
        dateI, errI := time.Parse(time.RFC3339, a.Entries[i].Post.PostDate)
        dateJ, errJ := time.Parse(time.RFC3339, a.Entries[j].Post.PostDate)
        
        if errI != nil || errJ != nil {
            // If parsing fails, sort by title alphabetically
            return a.Entries[i].Post.Title < a.Entries[j].Post.Title
        }
        
        return dateI.After(dateJ) // newest first
    })
}
```

### 4.4 File Path Management

- **Relative Paths**: All post links use `filepath.Rel()` for portability
- **Cross-Platform Compatibility**: Proper path separators for all operating systems
- **Directory Structure Preservation**: Maintains existing file organization patterns

## 5. Integration Points

### 5.1 Download Flow Integration

```go
// Archive initialization in download command
var archive *lib.Archive
if createArchive {
    archive = lib.NewArchive()
}

// Entry collection during download processing
if archive != nil {
    archive.AddEntry(post, path, time.Now())
}

// Archive generation after downloads complete
if archive != nil && len(archive.Entries) > 0 {
    var archiveErr error
    switch format {
    case "html":
        archiveErr = archive.GenerateHTML(outputFolder)
    case "md":
        archiveErr = archive.GenerateMarkdown(outputFolder)
    case "txt":
        archiveErr = archive.GenerateText(outputFolder)
    }
}
```

### 5.2 Format Consistency

- **Output Format Matching**: Archive format automatically matches selected post format
- **Content Alignment**: Archive styling and structure complement post formatting
- **Directory Structure**: Archive placed in root output directory alongside posts

## 6. Archive Content Structure

### 6.1 Post Metadata Display

Each archive entry includes:
- **Title**: Clickable link to downloaded post file
- **Publication Date**: Original Substack publication date (formatted: "January 2, 2006")
- **Download Date**: Local download timestamp (formatted: "January 2, 2006 15:04")
- **Description**: Post subtitle (priority) or description (fallback)
- **Cover Image**: Featured post image when available

### 6.2 Content Prioritization

```go
// Description selection logic
description := entry.Post.Subtitle
if description == "" {
    description = entry.Post.Description
}
```

### 6.3 Date Formatting

- **Publication Date**: Human-readable format ("January 2, 2006")
- **Download Date**: Includes time for precise tracking ("January 2, 2006 15:04")
- **Sorting**: Uses RFC3339 format for accurate chronological ordering

## 7. Error Handling Strategy

### 7.1 Archive Generation Errors

- **Directory Creation**: Automatic creation of output directory if missing
- **File Writing**: Graceful handling of permission and disk space issues
- **Format Validation**: Error reporting for unknown or unsupported formats

### 7.2 Metadata Processing

- **Date Parsing**: Fallback to title-based sorting for unparseable dates  
- **Missing Fields**: Graceful handling of empty subtitles, descriptions, or cover images
- **Path Generation**: Error handling for invalid file paths or relative path calculation failures

### 7.3 Content Validation

- **Empty Archives**: Skip generation when no entries are present
- **Invalid Entries**: Continue processing valid entries when individual entries have issues
- **HTML Escaping**: Proper escaping of user content in HTML format

## 8. Performance Considerations

### 8.1 Memory Management

- **Incremental Building**: Archive entries added incrementally during download process
- **Efficient Sorting**: In-place sorting using standard library algorithms
- **Content Generation**: String building optimized for each format type

### 8.2 File I/O Optimization

- **Single Write Operations**: Generate complete content before writing to disk
- **Relative Path Caching**: Efficient path calculation using filepath.Rel()
- **Format-Specific Generation**: Only generate requested format to minimize overhead

## 9. Testing Strategy

### 9.1 Unit Tests

```go
// Comprehensive test coverage areas
func TestNewArchive(t *testing.T)
func TestArchive_AddEntry(t *testing.T)
func TestArchive_sortEntries(t *testing.T)
func TestArchive_GenerateHTML(t *testing.T)
func TestArchive_GenerateMarkdown(t *testing.T)
func TestArchive_GenerateText(t *testing.T)
func TestEnhancedPostExtraction(t *testing.T)
```

### 9.2 Integration Tests

```go
func TestArchiveWorkflow(t *testing.T)
func TestCommandFlags(t *testing.T)
func TestArchivePageGeneration(t *testing.T)
```

### 9.3 Test Coverage Areas

- **Data Structure Operations**: Archive creation, entry management, sorting
- **Format Generation**: Content generation for all three formats
- **Error Scenarios**: Invalid dates, missing fields, empty archives
- **Integration**: End-to-end workflows with CLI flag integration
- **Post Enhancement**: Subtitle and cover image extraction functionality

## 10. Security Considerations

### 10.1 Content Security

- **HTML Escaping**: Proper escaping of post titles and descriptions in HTML format
- **Path Validation**: Safe relative path generation preventing directory traversal
- **Input Sanitization**: Clean handling of user-provided post content

### 10.2 File System Security

- **Directory Containment**: Archive files created only in designated output directory
- **Permission Handling**: Graceful handling of file system permission restrictions
- **Path Safety**: Cross-platform safe path generation and validation

## 11. Directory Structure Impact

### 11.1 Output Structure with Archive

```
output/
├── index.html                    # Archive index page
├── 20231201_120000_post-title.html
├── 20231115_090000_another-post.html
├── images/
│   ├── post-title/
│   │   └── image1_1456x819.jpeg
│   └── another-post/
│       └── image2_848x636.png
└── files/
    ├── post-title/
    │   └── document.pdf
    └── another-post/
        └── spreadsheet.xlsx
```

### 11.2 Archive Index Formats

- **HTML**: `index.html` - Styled webpage with embedded CSS
- **Markdown**: `index.md` - Clean markdown for documentation systems
- **Text**: `index.txt` - Plain text for maximum compatibility

## 12. Migration and Rollout

### 12.1 Backward Compatibility

- **Opt-in Feature**: Archive generation only when `--create-archive` flag is used
- **No Breaking Changes**: Existing CLI behavior unchanged when flag not present
- **Format Consistency**: Archive format automatically matches post format selection

### 12.2 Progressive Enhancement

- **Single Post Support**: Build archives incrementally with individual post downloads
- **Bulk Download Integration**: Seamless operation with existing bulk download workflows
- **Feature Combination**: Full compatibility with image and file download features

## 13. Future Enhancements

### 13.1 Potential Extensions

- **Custom Templates**: User-provided HTML/Markdown templates for archive pages
- **Theme Support**: Multiple built-in themes for HTML archive format
- **Pagination**: Support for paginated archives with very large post collections
- **Search Integration**: Client-side search functionality for archive pages

### 13.2 Advanced Features

- **Archive Regeneration**: Rebuild archive from existing downloaded files
- **Multiple Formats**: Generate archive in multiple formats simultaneously
- **RSS Generation**: Create RSS/Atom feeds from archive content
- **Static Site Integration**: Export formats compatible with static site generators

---

**Specification Status**: Implemented v1.0  
**Last Updated**: 2025-01-03  
**Dependencies**: Existing sbstck-dl codebase (fetcher.go, extractor.go), enhanced Post struct  
**Implementation**: Complete with comprehensive test coverage