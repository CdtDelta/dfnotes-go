-- Recreate ioc_entries with the Phase 3a schema.
-- The original table (migration 001) had incompatible columns; rename it away,
-- create the new table, migrate any existing rows, then drop the old one.

ALTER TABLE ioc_entries RENAME TO ioc_entries_v0;

CREATE TABLE IF NOT EXISTS ioc_entries (
    ioc_id          TEXT PRIMARY KEY,
    case_id         TEXT NOT NULL REFERENCES cases(case_id),
    block_id        TEXT NOT NULL REFERENCES note_blocks(block_id),
    evidence_item_id TEXT REFERENCES evidence_items(evidence_item_id),
    type            TEXT NOT NULL CHECK(type IN (
                        'ipv4', 'ipv6', 'domain', 'url', 'email',
                        'md5', 'sha1', 'sha256',
                        'file_path', 'registry_key', 'cve'
                    )),
    value           TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'detected' CHECK(status IN (
                        'detected', 'confirmed', 'false_positive'
                    )),
    detection_method TEXT NOT NULL DEFAULT 'auto' CHECK(detection_method IN ('auto', 'manual')),
    notes           TEXT,
    created_at      TEXT NOT NULL,
    confirmed_at    TEXT,
    user_id         TEXT NOT NULL REFERENCES users(user_id),
    UNIQUE(block_id, type, value),
    FOREIGN KEY (case_id) REFERENCES cases(case_id)
);

CREATE INDEX IF NOT EXISTS idx_ioc_entries_case_id  ON ioc_entries(case_id);
CREATE INDEX IF NOT EXISTS idx_ioc_entries_block_id  ON ioc_entries(block_id);
CREATE INDEX IF NOT EXISTS idx_ioc_entries_status    ON ioc_entries(status);
CREATE INDEX IF NOT EXISTS idx_ioc_entries_type      ON ioc_entries(type);

DROP TABLE ioc_entries_v0;
