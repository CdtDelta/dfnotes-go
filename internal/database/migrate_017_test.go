package database

import (
	"bytes"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const preTimelineNormalizeVersion = 16

// seedTimelineRow inserts an entry at schema version 16. updated_at is fixed so
// tests can prove the backfill leaves it alone.
func seedTimelineRow(t *testing.T, db *DB, id, timestamp string) {
	t.Helper()
	mustExec(t, db, `INSERT INTO timeline_entries (entry_id, case_id, timestamp, event_description,
		created_at, updated_at, user_id)
		VALUES (?, 'case-1', ?, 'event', ?, ?, 'user-1')`, id, timestamp, ts(0), ts(1))
}

func readTimelineRow(t *testing.T, db *DB, id string) (timestamp string, asEntered sql.NullString, updatedAt string) {
	t.Helper()
	err := db.QueryRow("SELECT timestamp, timestamp_as_entered, updated_at FROM timeline_entries WHERE entry_id = ?", id).
		Scan(&timestamp, &asEntered, &updatedAt)
	if err != nil {
		t.Fatalf("read timeline entry %s: %v", id, err)
	}
	return timestamp, asEntered, updatedAt
}

func TestMigrate017TimelineTimestamps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db := openAtVersion(t, path, preTimelineNormalizeVersion)
	defer db.Close()

	mustExec(t, db, `INSERT INTO users (user_id, name, public_key, encrypted_private_key, salt, created_at)
		VALUES ('user-1', 'Examiner', x'00', x'00', x'00', ?)`, ts(0))
	mustExec(t, db, `INSERT INTO cases (case_id, title, created_by, created_at, updated_at)
		VALUES ('case-1', 'case-1', 'user-1', ?, ?)`, ts(0), ts(0))

	seedTimelineRow(t, db, "offset", "2026-10-30T10:00:00-05:00")
	seedTimelineRow(t, db, "zero-offset", "2026-10-30T15:00:00+00:00")
	seedTimelineRow(t, db, "fraction", "2026-10-30T15:00:00.500Z")
	seedTimelineRow(t, db, "canonical", "2026-10-30T15:00:00Z")
	seedTimelineRow(t, db, "bad", "not a timestamp")
	seedTimelineRow(t, db, "ten-digit-fraction", "2026-10-30T15:00:05.1234567899Z")

	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	if err := db.migrateTo(path, 17); err != nil {
		t.Fatalf("migrateTo(17): %v", err)
	}

	tests := []struct {
		id        string
		timestamp string
		asEntered string // empty means NULL
	}{
		{"offset", "2026-10-30T15:00:00Z", "2026-10-30T10:00:00-05:00"},
		{"zero-offset", "2026-10-30T15:00:00Z", "2026-10-30T15:00:00+00:00"},
		{"fraction", "2026-10-30T15:00:00.5Z", ""},
		{"canonical", "2026-10-30T15:00:00Z", ""},
		{"bad", "not a timestamp", ""},
		{"ten-digit-fraction", "2026-10-30T15:00:05.1234567899Z", ""},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			timestamp, asEntered, updatedAt := readTimelineRow(t, db, tt.id)
			if timestamp != tt.timestamp {
				t.Errorf("timestamp = %q, want %q", timestamp, tt.timestamp)
			}
			if tt.asEntered == "" && asEntered.Valid {
				t.Errorf("timestamp_as_entered = %q, want NULL", asEntered.String)
			}
			if tt.asEntered != "" && asEntered.String != tt.asEntered {
				t.Errorf("timestamp_as_entered = %q, want %q", asEntered.String, tt.asEntered)
			}
			if updatedAt != ts(1) {
				t.Errorf("updated_at = %q, want it unchanged at %q", updatedAt, ts(1))
			}
		})
	}

	if v := schemaVersion(t, db.DB); v != 17 {
		t.Errorf("schema_version = %d, want 17", v)
	}

	want := "3 timeline timestamps normalized, 2 offsets preserved, 2 skipped as unparseable"
	if !strings.Contains(logged.String(), want) {
		t.Errorf("log output = %q, want it to contain %q", logged.String(), want)
	}
}

func TestMigrate017NoRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db := openAtVersion(t, path, preTimelineNormalizeVersion)
	defer db.Close()

	if err := db.migrateTo(path, 17); err != nil {
		t.Fatalf("migrateTo(17): %v", err)
	}
	if v := schemaVersion(t, db.DB); v != 17 {
		t.Errorf("schema_version = %d, want 17", v)
	}
}
