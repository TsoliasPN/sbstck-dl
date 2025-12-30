package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type jarCookie struct {
	Domain string
	Name   string
	Value  string
}

func readCookieFromJar(path string, desiredName cookieName, domainHint string) (cookieName, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	cookies := make([]jarCookie, 0, 8)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}
		cookies = append(cookies, jarCookie{
			Domain: fields[0],
			Name:   fields[5],
			Value:  fields[6],
		})
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}

	candidates := []cookieName{}
	if desiredName != "" {
		candidates = []cookieName{desiredName}
	} else {
		candidates = []cookieName{substackSid, connectSid}
	}

	for _, name := range candidates {
		if cookie, ok := findCookie(cookies, string(name), domainHint); ok {
			return name, cookie.Value, nil
		}
	}

	return "", "", fmt.Errorf("no matching cookie found in jar")
}

func findCookie(cookies []jarCookie, name string, domainHint string) (jarCookie, bool) {
	matches := make([]jarCookie, 0, 2)
	for _, cookie := range cookies {
		if cookie.Name == name {
			matches = append(matches, cookie)
		}
	}
	if len(matches) == 0 {
		return jarCookie{}, false
	}
	if domainHint == "" {
		return matches[0], true
	}

	best := jarCookie{}
	bestLen := -1
	for _, cookie := range matches {
		if !domainMatch(domainHint, cookie.Domain) {
			continue
		}
		length := len(strings.TrimPrefix(cookie.Domain, "."))
		if length > bestLen {
			best = cookie
			bestLen = length
		}
	}
	if bestLen >= 0 {
		return best, true
	}
	return matches[0], true
}

func domainMatch(domainHint string, cookieDomain string) bool {
	domain := strings.TrimPrefix(cookieDomain, ".")
	if domain == "" {
		return false
	}
	if domainHint == domain {
		return true
	}
	return strings.HasSuffix(domainHint, "."+domain)
}
