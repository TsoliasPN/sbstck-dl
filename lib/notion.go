package lib

import (
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var notionURLRegex = regexp.MustCompile(`https?://[^\s\)"'>\]]+`)

// ExtractNotionLinks parses content for Notion links and returns normalized, deduped URLs.
func ExtractNotionLinks(content string, format string) []string {
	links := make(map[string]struct{})
	format = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(format)), ".")

	if format == "html" {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err == nil {
			doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
				href, ok := s.Attr("href")
				if !ok {
					return
				}
				if normalized, ok := NormalizeNotionURL(href); ok {
					links[normalized] = struct{}{}
				}
			})
		}
	} else {
		matches := notionURLRegex.FindAllString(content, -1)
		for _, match := range matches {
			candidate := trimURLCandidate(match)
			if normalized, ok := NormalizeNotionURL(candidate); ok {
				links[normalized] = struct{}{}
			}
		}
	}

	results := make([]string, 0, len(links))
	for link := range links {
		results = append(results, link)
	}
	sort.Strings(results)
	return results
}

// NormalizeNotionURL canonicalizes Notion URLs for deduping.
func NormalizeNotionURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}

	host := strings.ToLower(parsed.Host)
	host = strings.TrimPrefix(host, "www.")
	if !strings.HasSuffix(host, "notion.so") && !strings.HasSuffix(host, "notion.site") {
		return "", false
	}

	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path == "" {
		path = "/"
	}

	normalized := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   path,
	}
	return normalized.String(), true
}

func trimURLCandidate(value string) string {
	return strings.TrimRight(value, ".,);:]\"'")
}
