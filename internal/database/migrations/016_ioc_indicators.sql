-- Case-wide IOC indicators (v0.12.0). Creates the new tables only; the Go hook
-- migrate016IOCIndicators backfills them from ioc_entries, rebuilds case_facts
-- to reference indicators, and drops ioc_entries.

CREATE TABLE indicators (
    indicator_id      TEXT PRIMARY KEY,
    case_id           TEXT NOT NULL REFERENCES cases(case_id),
    type              TEXT NOT NULL CHECK(type IN (
                          'ipv4', 'ipv6', 'domain', 'url', 'email',
                          'md5', 'sha1', 'sha256',
                          'file_path', 'file', 'registry_key', 'cve'
                      )),
    value             TEXT NOT NULL,
    match_key         TEXT NOT NULL,
    match_key_version INTEGER NOT NULL,
    status            TEXT NOT NULL DEFAULT 'detected' CHECK(status IN (
                          'detected', 'confirmed', 'false_positive', 'promoted'
                      )),
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    confirmed_at      TEXT,
    user_id           TEXT NOT NULL REFERENCES users(user_id),
    UNIQUE(case_id, type, match_key)
);
CREATE INDEX idx_indicators_case_id ON indicators(case_id);
CREATE INDEX idx_indicators_status  ON indicators(status);

CREATE TABLE ioc_occurrences (
    occurrence_id    TEXT PRIMARY KEY,
    indicator_id     TEXT NOT NULL REFERENCES indicators(indicator_id) ON DELETE CASCADE,
    block_id         TEXT NOT NULL REFERENCES note_blocks(block_id),
    raw_value        TEXT NOT NULL,
    detection_method TEXT NOT NULL DEFAULT 'auto' CHECK(detection_method IN ('auto', 'manual', 'propagated')),
    created_at       TEXT NOT NULL,
    user_id          TEXT NOT NULL REFERENCES users(user_id),
    UNIQUE(indicator_id, block_id, raw_value)
);
CREATE INDEX idx_ioc_occurrences_block_id ON ioc_occurrences(block_id);
