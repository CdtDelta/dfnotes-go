-- Add 'promoted' as a valid IOC status. SQLite does not support ALTER COLUMN,
-- so recreate the table with the updated CHECK constraint and migrate data.

PRAGMA foreign_keys = OFF;

CREATE TABLE ioc_entries_new (
    ioc_id           TEXT PRIMARY KEY,
    case_id          TEXT NOT NULL REFERENCES cases(case_id),
    block_id         TEXT NOT NULL REFERENCES note_blocks(block_id),
    evidence_item_id TEXT REFERENCES evidence_items(evidence_item_id),
    type             TEXT NOT NULL CHECK(type IN (
                         'ipv4', 'ipv6', 'domain', 'url', 'email',
                         'md5', 'sha1', 'sha256',
                         'file_path', 'file', 'registry_key', 'cve'
                     )),
    value            TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'detected' CHECK(status IN (
                         'detected', 'confirmed', 'false_positive', 'promoted'
                     )),
    detection_method TEXT NOT NULL DEFAULT 'auto' CHECK(detection_method IN ('auto', 'manual')),
    notes            TEXT,
    created_at       TEXT NOT NULL,
    confirmed_at     TEXT,
    user_id          TEXT NOT NULL REFERENCES users(user_id),
    UNIQUE(block_id, type, value),
    FOREIGN KEY (case_id) REFERENCES cases(case_id)
);

INSERT INTO ioc_entries_new SELECT * FROM ioc_entries;

DROP TABLE ioc_entries;

ALTER TABLE ioc_entries_new RENAME TO ioc_entries;

CREATE INDEX IF NOT EXISTS idx_ioc_entries_case_id   ON ioc_entries(case_id);
CREATE INDEX IF NOT EXISTS idx_ioc_entries_block_id  ON ioc_entries(block_id);
CREATE INDEX IF NOT EXISTS idx_ioc_entries_status    ON ioc_entries(status);
CREATE INDEX IF NOT EXISTS idx_ioc_entries_type      ON ioc_entries(type);

CREATE UNIQUE INDEX uq_ioc_entries_block_type_value ON ioc_entries(block_id, type, value);

PRAGMA foreign_keys = ON;
