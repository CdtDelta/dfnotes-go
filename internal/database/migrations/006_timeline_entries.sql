-- Replace the Phase 1 timeline_entries schema (event_time/title/description)
-- with the Phase 3b schema (timestamp/event_description/investigator_notes).
-- The old table was a stub with no service layer and no real data.
DROP TABLE IF EXISTS timeline_entries;

CREATE TABLE timeline_entries (
    entry_id            TEXT PRIMARY KEY,
    case_id             TEXT NOT NULL REFERENCES cases(case_id),
    evidence_item_id    TEXT REFERENCES evidence_items(evidence_item_id),
    timestamp           TEXT NOT NULL,
    display_timezone    TEXT,
    event_description   TEXT NOT NULL,
    investigator_notes  TEXT,
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    user_id             TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_timeline_entries_case_id ON timeline_entries(case_id);
CREATE INDEX IF NOT EXISTS idx_timeline_entries_timestamp ON timeline_entries(timestamp);
