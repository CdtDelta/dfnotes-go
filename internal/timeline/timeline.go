// Package timeline holds the canonical timestamp form for timeline entries and
// the single authoritative sort order for displaying them.
package timeline

import (
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"dfnotes-go/internal/models"
)

// strictTimestamp is checked before parsing because time.Parse is looser than
// RFC 3339: it accepts comma separators and single-digit hours, and silently
// drops fractional digits past the ninth, which would alter a forensic record.
var strictTimestamp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?(Z|[+-]\d{2}:\d{2})$`)

var errTimestampFormat = errors.New("Timestamp must look like 2026-10-30T15:00:05Z (up to 9 fractional digits; offsets like -05:00 if enabled in Settings)")

// Normalize validates a timestamp and returns its canonical UTC form. asEntered
// holds the trimmed input only when it carries a non-Z offset, because a Z input
// loses nothing in canonicalization.
func Normalize(input string, allowOffsets bool) (canonical string, asEntered *string, err error) {
	trimmed := strings.TrimSpace(input)

	if !strictTimestamp.MatchString(trimmed) {
		return "", nil, errTimestampFormat
	}
	t, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return "", nil, errTimestampFormat
	}

	isUTC := strings.HasSuffix(trimmed, "Z")
	if !allowOffsets && !isUTC {
		return "", nil, errors.New("Timestamp must be in UTC (ending in Z); UTC offsets can be enabled in Settings")
	}
	if !isUTC && !validOffset(trimmed[len(trimmed)-6:]) {
		return "", nil, errTimestampFormat
	}

	canonical = t.UTC().Format(time.RFC3339Nano)
	if !isUTC {
		asEntered = &trimmed
	}
	return canonical, asEntered, nil
}

// validOffset range-checks a shape-validated +HH:MM or -HH:MM offset, because
// time.Parse accepts values up to +24:00.
func validOffset(offset string) bool {
	hours, err := strconv.Atoi(offset[1:3])
	if err != nil {
		return false
	}
	minutes, err := strconv.Atoi(offset[4:6])
	if err != nil {
		return false
	}
	return hours <= 23 && minutes <= 59
}

// sortKey caches the parsed timestamp so comparisons do not reparse it.
type sortKey struct {
	entry    models.TimelineEntry
	parsed   time.Time
	parsedOK bool
}

// Sort orders entries in place by parsed time ascending, breaking ties by
// created_at then entry_id. Rows whose timestamp does not parse go after all
// parseable rows, ordered by raw string, so they stay visible for correction.
func Sort(entries []models.TimelineEntry) {
	keys := make([]sortKey, len(entries))
	for i, e := range entries {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(e.Timestamp))
		keys[i] = sortKey{entry: e, parsed: parsed, parsedOK: err == nil}
	}

	sort.SliceStable(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.parsedOK != b.parsedOK {
			return a.parsedOK
		}
		if a.parsedOK {
			if !a.parsed.Equal(b.parsed) {
				return a.parsed.Before(b.parsed)
			}
		} else if a.entry.Timestamp != b.entry.Timestamp {
			return a.entry.Timestamp < b.entry.Timestamp
		}
		if a.entry.CreatedAt != b.entry.CreatedAt {
			return a.entry.CreatedAt < b.entry.CreatedAt
		}
		return a.entry.EntryID < b.entry.EntryID
	})

	for i, k := range keys {
		entries[i] = k.entry
	}
}
