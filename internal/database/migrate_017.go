package database

import (
	"database/sql"
	"fmt"
	"log"

	"dfnotes-go/internal/timeline"
)

// legacyTimelineRow is one timeline_entries row as it exists before the backfill.
type legacyTimelineRow struct {
	entryID   string
	timestamp string
}

// migrate017TimelineTimestamps rewrites stored timeline timestamps to canonical
// UTC. Offsets are accepted regardless of the timeline_allow_offsets setting,
// because existing rows were validated as RFC 3339 and may contain them. A row
// that does not parse is left as is rather than failing the migration, since
// that would lock the examiner out of the whole database over one bad timestamp.
// updated_at is not touched: this is not an examiner edit.
func migrate017TimelineTimestamps(tx *sql.Tx) error {
	rows, err := readLegacyTimelineRows(tx)
	if err != nil {
		return err
	}

	var normalized, offsetsPreserved, skipped int
	for _, r := range rows {
		canonical, asEntered, err := timeline.Normalize(r.timestamp, true)
		if err != nil {
			skipped++
			continue
		}
		if canonical == r.timestamp && asEntered == nil {
			continue
		}
		if _, err := tx.Exec(
			"UPDATE timeline_entries SET timestamp = ?, timestamp_as_entered = ? WHERE entry_id = ?",
			canonical, asEntered, r.entryID,
		); err != nil {
			return fmt.Errorf("update timeline entry %s: %w", r.entryID, err)
		}
		normalized++
		if asEntered != nil {
			offsetsPreserved++
		}
	}

	log.Printf("Migration 017: %d timeline timestamps normalized, %d offsets preserved, %d skipped as unparseable",
		normalized, offsetsPreserved, skipped)
	return nil
}

// readLegacyTimelineRows returns with the cursor closed, because the single
// connection cannot run the backfill updates while a query is open.
func readLegacyTimelineRows(tx *sql.Tx) ([]legacyTimelineRow, error) {
	rs, err := tx.Query("SELECT entry_id, timestamp FROM timeline_entries")
	if err != nil {
		return nil, fmt.Errorf("read timeline_entries: %w", err)
	}
	defer rs.Close()

	var out []legacyTimelineRow
	for rs.Next() {
		var r legacyTimelineRow
		if err := rs.Scan(&r.entryID, &r.timestamp); err != nil {
			return nil, fmt.Errorf("scan timeline_entries: %w", err)
		}
		out = append(out, r)
	}
	if err := rs.Err(); err != nil {
		return nil, fmt.Errorf("iterate timeline_entries: %w", err)
	}
	return out, nil
}
