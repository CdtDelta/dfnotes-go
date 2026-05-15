package ioc

import (
	"regexp"
	"strings"
)

// Detection order matters: run more-specific patterns first so values are
// stored under the most-specific type. URL before domain; SHA256 before SHA1
// before MD5. Each matched value is tracked to prevent double-reporting.

var (
	reURL = regexp.MustCompile(
		`(?i)\b(https?|ftp)://[^\s<>"']+`,
	)

	reIPv4 = regexp.MustCompile(
		`\b(?:(?:25[0-5]|2[0-4]\d|1\d{2}|[1-9]\d|\d)\.){3}(?:25[0-5]|2[0-4]\d|1\d{2}|[1-9]\d|\d)\b`,
	)

	reIPv6 = regexp.MustCompile(
		`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:)*::(?:[0-9a-fA-F]{1,4}:)*[0-9a-fA-F]{1,4}\b|` +
			`\b::(?:[0-9a-fA-F]{1,4}:)*[0-9a-fA-F]{1,4}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:)+:\b`,
	)

	reEmail = regexp.MustCompile(
		`\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,6}\b`,
	)

	reSHA256 = regexp.MustCompile(`\b[0-9a-fA-F]{64}\b`)
	reSHA1   = regexp.MustCompile(`\b[0-9a-fA-F]{40}\b`)
	reMD5    = regexp.MustCompile(`\b[0-9a-fA-F]{32}\b`)

	// Require a TLD (2-6 alpha chars) -- bare hostnames with no dot are excluded.
	// High FP rate; domain matches inside URL strings are suppressed downstream.
	reDomain = regexp.MustCompile(
		`\b(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}\b`,
	)

	reWinPath = regexp.MustCompile(
		`\b[A-Za-z]:\\(?:[^\\/:*?"<>|\r\n]+\\)*[^\s\\/:*?"<>|\r\n]*`,
	)

	// Group 1 captures the path; leading whitespace/quote is not part of the match.
	reUnixPath = regexp.MustCompile(
		`(?:^|[\s"'(])(/[a-zA-Z0-9._\-]+(?:/[a-zA-Z0-9._\-]+)+)`,
	)

	reRegistry = regexp.MustCompile(
		`(?i)\b(?:HKEY_LOCAL_MACHINE|HKEY_CURRENT_USER|HKEY_CLASSES_ROOT|HKEY_USERS|HKEY_CURRENT_CONFIG|HKLM|HKCU|HKCR)\\[^\s"'<>]+`,
	)

	reCVE = regexp.MustCompile(`\bCVE-\d{4}-\d{4,7}\b`)
)

// File extensions that look like TLDs but are not domain indicators.
var nonTLDExtensions = map[string]struct{}{
	"dll": {}, "exe": {}, "sys": {}, "bat": {}, "ps1": {}, "sh": {},
	"log": {}, "txt": {}, "json": {}, "xml": {}, "csv": {},
	"zip": {}, "tar": {}, "gz": {},
}

// DetectIOCs scans plaintext content and returns deduplicated IOC matches.
// Order of evaluation determines which type wins when a value could fit
// multiple patterns (e.g. a URL is matched as URL, not also as domain).
func DetectIOCs(content string) []IOCMatch {
	seen := make(map[string]struct{})
	var results []IOCMatch

	add := func(t IOCType, value string) {
		key := string(t) + ":" + value
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			results = append(results, IOCMatch{Type: t, Value: value})
		}
	}

	// CVE -- very precise, run first.
	for _, m := range reCVE.FindAllString(content, -1) {
		add(IOCTypeCVE, m)
	}

	// Registry keys -- anchored on known hive prefixes, low FP rate.
	for _, m := range reRegistry.FindAllString(content, -1) {
		add(IOCTypeRegistryKey, m)
	}

	// URL before domain (more specific).
	urlMatches := reURL.FindAllString(content, -1)
	urlSet := make(map[string]struct{}, len(urlMatches))
	for _, m := range urlMatches {
		add(IOCTypeURL, m)
		urlSet[m] = struct{}{}
	}

	// Email before domain (more specific).
	for _, m := range reEmail.FindAllString(content, -1) {
		add(IOCTypeEmail, m)
	}

	// Hashes: SHA256 before SHA1 before MD5 (longer = more specific).
	sha256Matches := reSHA256.FindAllString(content, -1)
	sha256Set := make(map[string]struct{}, len(sha256Matches))
	for _, m := range sha256Matches {
		add(IOCTypeSHA256, m)
		sha256Set[m] = struct{}{}
	}

	sha1Matches := reSHA1.FindAllString(content, -1)
	sha1Set := make(map[string]struct{}, len(sha1Matches))
	for _, m := range sha1Matches {
		if _, isSHA256 := sha256Set[m]; !isSHA256 {
			add(IOCTypeSHA1, m)
		}
		sha1Set[m] = struct{}{}
	}

	for _, m := range reMD5.FindAllString(content, -1) {
		if _, isSHA256 := sha256Set[m]; isSHA256 {
			continue
		}
		if _, isSHA1 := sha1Set[m]; isSHA1 {
			continue
		}
		add(IOCTypeMD5, m)
	}

	// IPv4. Post-filter: skip matches where a fifth dot-digit segment follows
	// (e.g. version strings like 1.2.3.4.5). RE2 has no lookaheads.
	for _, loc := range reIPv4.FindAllStringIndex(content, -1) {
		end := loc[1]
		if end < len(content) && content[end] == '.' &&
			end+1 < len(content) && content[end+1] >= '0' && content[end+1] <= '9' {
			continue
		}
		add(IOCTypeIPv4, content[loc[0]:loc[1]])
	}

	// IPv6.
	for _, m := range reIPv6.FindAllString(content, -1) {
		add(IOCTypeIPv6, m)
	}

	// Windows file paths -- run before domain to avoid matching extensions as TLDs.
	for _, m := range reWinPath.FindAllString(content, -1) {
		add(IOCTypeFilePath, m)
	}

	// Unix file paths (submatch group 1).
	for _, m := range reUnixPath.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			add(IOCTypeFilePath, m[1])
		}
	}

	// Domain -- skip values already captured as part of a URL or with non-TLD extensions.
	for _, m := range reDomain.FindAllString(content, -1) {
		inURL := false
		for u := range urlSet {
			if strings.Contains(u, m) {
				inURL = true
				break
			}
		}
		if inURL {
			continue
		}
		lastDot := strings.LastIndex(m, ".")
		if lastDot >= 0 {
			ext := strings.ToLower(m[lastDot+1:])
			if _, isNonTLD := nonTLDExtensions[ext]; isNonTLD {
				continue
			}
		}
		add(IOCTypeDomain, m)
	}

	return results
}

