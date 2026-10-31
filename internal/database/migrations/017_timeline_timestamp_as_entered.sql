-- Timeline timestamp normalization (v0.14.0). Adds the column only; the Go hook
-- migrate017TimelineTimestamps backfills canonical UTC timestamps and preserves
-- any original entered with a UTC offset.

ALTER TABLE timeline_entries ADD COLUMN timestamp_as_entered TEXT;
