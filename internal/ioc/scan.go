package ioc

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// KnownMatch is one place a known indicator's value appears in note content.
type KnownMatch struct {
	IndicatorID string
	Type        IOCType
	RawValue    string // the text exactly as it appears in the content
}

// FindKnownValues returns every place a candidate indicator's value appears
// in content as a standalone token. A hit is accepted only when its MatchKey
// equals the candidate's, so a match can never land on a different indicator.
// Results follow candidate order, then content order, deduplicated by
// (IndicatorID, RawValue).
func FindKnownValues(content string, candidates []Indicator) []KnownMatch {
	out := []KnownMatch{}
	if content == "" {
		return out
	}

	type seenKey struct{ id, raw string }
	seen := make(map[seenKey]struct{})

	for _, c := range candidates {
		// MatchKey trims, so the search uses the trimmed value too; padding
		// on the stored display value would otherwise never match.
		value := strings.TrimSpace(c.Value)
		if value == "" {
			continue
		}
		pattern := regexp.QuoteMeta(value)
		if foldsCase(c.Type, value) {
			pattern = "(?i)" + pattern
		}
		re := regexp.MustCompile(pattern)

		// Offsets index the original content; lowercasing it first could
		// change byte lengths under Unicode case folding.
		for _, loc := range re.FindAllStringIndex(content, -1) {
			if !standalone(content, loc[0], loc[1], c.Type) {
				continue
			}
			found := content[loc[0]:loc[1]]
			if MatchKey(c.Type, found) != c.MatchKey {
				continue
			}
			k := seenKey{c.IndicatorID, found}
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, KnownMatch{IndicatorID: c.IndicatorID, Type: c.Type, RawValue: found})
		}
	}
	return out
}

// foldsCase reports whether MatchKey treats values of this type as case
// insensitive, so the search should be too. The match-key guard still has
// the final say.
func foldsCase(t IOCType, value string) bool {
	switch t {
	case IOCTypeDomain, IOCTypeEmail, IOCTypeMD5, IOCTypeSHA1, IOCTypeSHA256,
		IOCTypeCVE, IOCTypeRegistryKey, IOCTypeURL, IOCTypeIPv6:
		return true
	case IOCTypeFilePath:
		return isWindowsPath(value)
	default:
		return false
	}
}

// standalone reports whether content[start:end] is not part of a longer
// token. Separators such as path slashes, colons, spaces, quotes and brackets
// before the value are allowed, so a file name inside a full path matches.
func standalone(content string, start, end int, t IOCType) bool {
	if start > 0 {
		r, size := utf8.DecodeLastRuneInString(content[:start])
		if isWordRune(r) {
			return false
		}
		if r == '.' && start-size > 0 {
			prev, _ := utf8.DecodeLastRuneInString(content[:start-size])
			if isAlnum(prev) {
				return false
			}
		}
	}
	if end < len(content) {
		r, size := utf8.DecodeRuneInString(content[end:])
		if isWordRune(r) {
			return false
		}
		if r == '.' && end+size < len(content) {
			next, _ := utf8.DecodeRuneInString(content[end+size:])
			if isAlnum(next) {
				return false
			}
		}
		// A path-like value followed by a separator is the prefix of a
		// longer path, not the value itself.
		switch t {
		case IOCTypeURL, IOCTypeFilePath, IOCTypeRegistryKey:
			if r == '/' || r == '\\' {
				return false
			}
		}
	}
	return true
}

func isAlnum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isWordRune(r rune) bool {
	return isAlnum(r) || r == '_' || r == '-'
}
