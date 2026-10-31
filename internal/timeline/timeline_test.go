package timeline

import (
	"strings"
	"testing"

	"dfnotes-go/internal/models"
)

func TestNormalizeAccepted(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		allowOffsets bool
		canonical    string
		asEntered    string // empty means nil
	}{
		{"z unchanged", "2026-10-30T15:00:00Z", false, "2026-10-30T15:00:00Z", ""},
		{"z unchanged with offsets allowed", "2026-10-30T15:00:00Z", true, "2026-10-30T15:00:00Z", ""},
		{"trailing zero fraction trimmed", "2026-10-30T15:00:00.500Z", false, "2026-10-30T15:00:00.5Z", ""},
		{"all zero fraction dropped", "2026-10-30T15:00:00.000Z", false, "2026-10-30T15:00:00Z", ""},
		{"nanosecond fraction preserved", "2026-10-30T15:00:04.999999999Z", false, "2026-10-30T15:00:04.999999999Z", ""},
		{"surrounding whitespace trimmed", "  2026-10-30T15:00:00Z\n", false, "2026-10-30T15:00:00Z", ""},
		{"negative offset converted", "2026-10-30T10:00:00-05:00", true, "2026-10-30T15:00:00Z", "2026-10-30T10:00:00-05:00"},
		{"zero offset kept as entered", "2026-10-30T15:00:00+00:00", true, "2026-10-30T15:00:00Z", "2026-10-30T15:00:00+00:00"},
		{"offset with fraction", "2026-10-30T10:00:00.250-05:00", true, "2026-10-30T15:00:00.25Z", "2026-10-30T10:00:00.250-05:00"},
		{"offset as entered is trimmed", " 2026-10-30T10:00:00-05:00 ", true, "2026-10-30T15:00:00Z", "2026-10-30T10:00:00-05:00"},
		{"largest valid offset", "2026-10-30T15:00:00+23:59", true, "2026-10-29T15:01:00Z", "2026-10-30T15:00:00+23:59"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, asEntered, err := Normalize(tt.input, tt.allowOffsets)
			if err != nil {
				t.Fatalf("Normalize(%q): %v", tt.input, err)
			}
			if canonical != tt.canonical {
				t.Errorf("canonical = %q, want %q", canonical, tt.canonical)
			}
			if tt.asEntered == "" {
				if asEntered != nil {
					t.Errorf("asEntered = %q, want nil", *asEntered)
				}
			} else if asEntered == nil || *asEntered != tt.asEntered {
				t.Errorf("asEntered = %v, want %q", asEntered, tt.asEntered)
			}
		})
	}
}

func TestNormalizeRejected(t *testing.T) {
	const (
		badFormat   = "must look like 2026-10-30T15:00:05Z"
		utcRequired = "must be in UTC"
	)
	tests := []struct {
		name         string
		input        string
		allowOffsets bool
		wantInErr    string
	}{
		{"offset when disallowed", "2026-10-30T10:00:00-05:00", false, utcRequired},
		{"zero offset when disallowed", "2026-10-30T15:00:00+00:00", false, utcRequired},
		{"date only", "2026-10-30", false, badFormat},
		{"date only offsets allowed", "2026-10-30", true, badFormat},
		{"offset-less", "2026-10-30T15:00:00", false, badFormat},
		{"offset-less offsets allowed", "2026-10-30T15:00:00", true, badFormat},
		{"lowercase z", "2026-10-30T15:00:00z", true, badFormat},
		{"lowercase t", "2026-10-30t15:00:00Z", true, badFormat},
		{"ten fractional digits", "2026-10-30T15:00:05.1234567899Z", true, badFormat},
		{"comma separator", "2026-10-30T15:00:05,5Z", true, badFormat},
		{"single-digit hour", "2026-10-30T5:00:05Z", true, badFormat},
		{"empty fraction", "2026-10-30T15:00:05.Z", true, badFormat},
		{"offset hours over 23", "2026-10-30T15:00:05+24:00", true, badFormat},
		{"offset minutes over 59", "2026-10-30T15:00:05+05:60", true, badFormat},
		{"offset without colon", "2026-10-30T15:00:05+0500", true, badFormat},
		{"trailing newline inside value", "2026-10-30T15:00:05Z\nx", true, badFormat},
		{"impossible date", "2026-02-30T15:00:00Z", false, badFormat},
		{"seconds out of range", "2026-10-30T15:00:60Z", false, badFormat},
		{"garbage", "yesterday at noon", true, badFormat},
		{"empty", "", false, badFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, asEntered, err := Normalize(tt.input, tt.allowOffsets)
			if err == nil {
				t.Fatalf("Normalize(%q) = %q, want error", tt.input, canonical)
			}
			if !strings.Contains(err.Error(), tt.wantInErr) {
				t.Errorf("error = %q, want it to mention %q", err, tt.wantInErr)
			}
			if canonical != "" || asEntered != nil {
				t.Errorf("got canonical %q, asEntered %v alongside error", canonical, asEntered)
			}
		})
	}
}

func entryIDs(entries []models.TimelineEntry) string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.EntryID
	}
	return strings.Join(ids, " ")
}

func TestSort(t *testing.T) {
	const created = "2026-10-30T00:00:00Z"
	tests := []struct {
		name    string
		entries []models.TimelineEntry
		want    string
	}{
		{
			name: "mixed precision orders by time not text",
			entries: []models.TimelineEntry{
				{EntryID: "c", Timestamp: "2026-10-30T15:00:05.5Z", CreatedAt: created},
				{EntryID: "b", Timestamp: "2026-10-30T15:00:05Z", CreatedAt: created},
				{EntryID: "a", Timestamp: "2026-10-30T15:00:04.999999999Z", CreatedAt: created},
			},
			want: "a b c",
		},
		{
			name: "offset compares by instant",
			entries: []models.TimelineEntry{
				{EntryID: "later", Timestamp: "2026-10-30T15:00:01Z", CreatedAt: created},
				{EntryID: "earlier", Timestamp: "2026-10-30T10:00:00-05:00", CreatedAt: created},
			},
			want: "earlier later",
		},
		{
			name: "ties break on created_at then entry_id",
			entries: []models.TimelineEntry{
				{EntryID: "z", Timestamp: "2026-10-30T15:00:00Z", CreatedAt: "2026-10-30T02:00:00Z"},
				{EntryID: "c", Timestamp: "2026-10-30T15:00:00Z", CreatedAt: "2026-10-30T01:00:00Z"},
				{EntryID: "b", Timestamp: "2026-10-30T15:00:00Z", CreatedAt: "2026-10-30T01:00:00Z"},
				{EntryID: "a", Timestamp: "2026-10-30T15:00:00Z", CreatedAt: "2026-10-30T03:00:00Z"},
			},
			want: "b c z a",
		},
		{
			name: "unparseable rows last by raw string",
			entries: []models.TimelineEntry{
				{EntryID: "bad-z", Timestamp: "zzz", CreatedAt: created},
				{EntryID: "good-late", Timestamp: "2027-01-01T00:00:00Z", CreatedAt: created},
				{EntryID: "bad-a", Timestamp: "aaa", CreatedAt: created},
				{EntryID: "good-early", Timestamp: "2026-01-01T00:00:00Z", CreatedAt: created},
			},
			want: "good-early good-late bad-a bad-z",
		},
		{
			name: "unparseable ties break on created_at then entry_id",
			entries: []models.TimelineEntry{
				{EntryID: "c", Timestamp: "bad", CreatedAt: "2026-10-30T02:00:00Z"},
				{EntryID: "b", Timestamp: "bad", CreatedAt: "2026-10-30T01:00:00Z"},
				{EntryID: "a", Timestamp: "bad", CreatedAt: "2026-10-30T02:00:00Z"},
			},
			want: "b a c",
		},
		{name: "empty", entries: nil, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Sort(tt.entries)
			if got := entryIDs(tt.entries); got != tt.want {
				t.Errorf("order = %q, want %q", got, tt.want)
			}
		})
	}
}
