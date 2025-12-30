package lib

import (
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var linkURLRegex = regexp.MustCompile(`https?://[^\s\)"'>\]]+`)

// ExtractLinkDomains parses content and returns normalized, deduped link domains.
func ExtractLinkDomains(content string, format string) []string {
	urls := extractLinkURLs(content, format)
	domains := make(map[string]struct{})

	for _, raw := range urls {
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			continue
		}
		host := strings.ToLower(parsed.Host)
		host = strings.TrimPrefix(host, "www.")
		domains[host] = struct{}{}
	}

	results := make([]string, 0, len(domains))
	for domain := range domains {
		results = append(results, domain)
	}
	sort.Strings(results)
	return results
}

func extractLinkURLs(content string, format string) []string {
	format = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(format)), ".")
	if format == "" {
		format = "html"
	}

	urls := make(map[string]struct{})
	if format == "html" {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err == nil {
			doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
				href, ok := s.Attr("href")
				if !ok {
					return
				}
				if trimmed := strings.TrimSpace(href); trimmed != "" {
					urls[trimmed] = struct{}{}
				}
			})
		}
	} else {
		matches := linkURLRegex.FindAllString(content, -1)
		for _, match := range matches {
			candidate := trimURLCandidate(match)
			if candidate != "" {
				urls[candidate] = struct{}{}
			}
		}
	}

	results := make([]string, 0, len(urls))
	for u := range urls {
		results = append(results, u)
	}
	sort.Strings(results)
	return results
}
