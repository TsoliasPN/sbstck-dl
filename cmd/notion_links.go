package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alexferrari88/sbstck-dl/lib"
)

const (
	notionLinksHTML = "notion-links.html"
	notionLinksMD   = "notion-links.md"
)

type notionPostLinks struct {
	Title   string
	RelPath string
	Links   []string
}

func generateNotionIndex(outputFolder string, format string) error {
	labelMap, err := loadNotionLabelMap(notionLabelsPath)
	if err != nil {
		return err
	}
	posts, err := buildNotionIndex(outputFolder, format)
	if err != nil && logFormat != logFormatJSON {
		log.Printf("Warning: %v\n", err)
	}
	if err := writeNotionIndexHTML(outputFolder, posts, labelMap); err != nil {
		return err
	}
	if err := writeNotionIndexMarkdown(outputFolder, posts, labelMap); err != nil {
		return err
	}
	return nil
}

func buildNotionIndex(outputFolder string, format string) ([]notionPostLinks, error) {
	var firstErr error
	paths, err := scanDownloadedFiles(outputFolder, format)
	if err != nil {
		firstErr = err
	}

	posts := make([]notionPostLinks, 0)
	for _, path := range paths {
		meta, _ := readMetadataSidecar(path)

		var links []string
		if meta.NotionLinks != nil {
			links = meta.NotionLinks
		} else {
			content, err := os.ReadFile(path)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			links = extractNotionLinks(string(content), format)
		}
		if len(links) == 0 {
			continue
		}

		title := meta.Title
		if title == "" {
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			title = slugFromFilename(base)
			if title == "" {
				title = base
			}
		}

		relPath := path
		if outputFolder != "" {
			if rel, err := filepath.Rel(outputFolder, path); err == nil {
				relPath = filepath.ToSlash(rel)
			}
		}

		posts = append(posts, notionPostLinks{
			Title:   title,
			RelPath: relPath,
			Links:   links,
		})
	}

	sort.Slice(posts, func(i, j int) bool {
		return strings.ToLower(posts[i].Title) < strings.ToLower(posts[j].Title)
	})

	return posts, firstErr
}

func extractNotionLinks(content string, format string) []string {
	return lib.ExtractNotionLinks(content, format)
}

func normalizeNotionURL(raw string) (string, bool) {
	return lib.NormalizeNotionURL(raw)
}

func writeNotionIndexHTML(outputFolder string, posts []notionPostLinks, labels map[string]string) error {
	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"UTF-8\">\n")
	buf.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	buf.WriteString("<title>Notion Links</title>\n<style>")
	buf.WriteString("body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;margin:0;padding:24px;background:#fafafa;color:#111}")
	buf.WriteString(".container{max-width:900px;margin:0 auto}")
	buf.WriteString("h1{margin-top:0} .post{background:#fff;border:1px solid #e5e7eb;border-radius:12px;padding:16px;margin-bottom:16px}")
	buf.WriteString(".post h2{margin:0 0 8px;font-size:18px} .post ul{margin:0;padding-left:18px}")
	buf.WriteString(".meta{color:#6b7280;font-size:13px;margin-bottom:12px}")
	buf.WriteString(".link-url{color:#6b7280;font-size:12px;margin-left:6px;word-break:break-all}")
	buf.WriteString("</style>\n</head>\n<body>\n<div class=\"container\">\n")
	buf.WriteString("<h1>Notion Links</h1>\n")
	buf.WriteString(fmt.Sprintf("<div class=\"meta\">Generated %s</div>\n", time.Now().Format("January 2, 2006 15:04")))

	if len(posts) == 0 {
		buf.WriteString("<p>No Notion links found.</p>\n")
	} else {
		for _, post := range posts {
			buf.WriteString("<div class=\"post\">\n")
			buf.WriteString(fmt.Sprintf("<h2><a href=\"%s\">%s</a></h2>\n", post.RelPath, htmlEscape(post.Title)))
			buf.WriteString("<ul>\n")
			for _, link := range post.Links {
				if label := notionLabelForLink(link, labels); label != "" {
					buf.WriteString(fmt.Sprintf("<li><a href=\"%s\" target=\"_blank\" rel=\"noreferrer\">%s</a><span class=\"link-url\">%s</span></li>\n", link, htmlEscape(label), link))
				} else {
					buf.WriteString(fmt.Sprintf("<li><a href=\"%s\" target=\"_blank\" rel=\"noreferrer\">%s</a></li>\n", link, link))
				}
			}
			buf.WriteString("</ul>\n</div>\n")
		}
	}

	buf.WriteString("</div>\n</body>\n</html>\n")
	path := filepath.Join(outputFolder, notionLinksHTML)
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func writeNotionIndexMarkdown(outputFolder string, posts []notionPostLinks, labels map[string]string) error {
	var buf bytes.Buffer
	buf.WriteString("# Notion Links\n\n")
	buf.WriteString(fmt.Sprintf("_Generated %s_\n\n", time.Now().Format("January 2, 2006 15:04")))
	if len(posts) == 0 {
		buf.WriteString("No Notion links found.\n")
	} else {
		for _, post := range posts {
			buf.WriteString(fmt.Sprintf("## [%s](%s)\n\n", post.Title, post.RelPath))
			for _, link := range post.Links {
				if label := notionLabelForLink(link, labels); label != "" {
					buf.WriteString(fmt.Sprintf("- [%s](%s)\n", label, link))
				} else {
					buf.WriteString(fmt.Sprintf("- %s\n", link))
				}
			}
			buf.WriteString("\n")
		}
	}
	path := filepath.Join(outputFolder, notionLinksMD)
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func notionLabelForLink(link string, labels map[string]string) string {
	if labels == nil {
		return ""
	}
	return labels[link]
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")
	return replacer.Replace(value)
}
