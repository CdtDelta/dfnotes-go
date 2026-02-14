CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    organization TEXT NOT NULL DEFAULT '',
    public_key BLOB NOT NULL,
    encrypted_private_key BLOB NOT NULL,
    salt BLOB NOT NULL,
    totp_enabled INTEGER NOT NULL DEFAULT 0,
    totp_secret_encrypted BLOB,
    recovery_codes_hash TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cases (
    case_id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    classification TEXT NOT NULL DEFAULT 'UNCLASSIFIED',
    created_by TEXT NOT NULL REFERENCES users(user_id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cases_created_by ON cases(created_by);

CREATE TABLE IF NOT EXISTS evidence_items (
    evidence_item_id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL REFERENCES cases(case_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    evidence_type TEXT NOT NULL DEFAULT 'OTHER',
    status TEXT NOT NULL DEFAULT 'COLLECTED',
    content_hash TEXT NOT NULL DEFAULT '',
    custody_log TEXT NOT NULL DEFAULT '[]',
    collected_by TEXT NOT NULL REFERENCES users(user_id),
    collected_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_evidence_items_case_id ON evidence_items(case_id);

CREATE TABLE IF NOT EXISTS note_blocks (
    block_id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL REFERENCES cases(case_id) ON DELETE CASCADE,
    evidence_item_id TEXT REFERENCES evidence_items(evidence_item_id),
    amends_block_id TEXT REFERENCES note_blocks(block_id),
    content_hash TEXT NOT NULL,
    prev_hash TEXT NOT NULL,
    signature BLOB NOT NULL,
    encrypted_body BLOB NOT NULL,
    author_id TEXT NOT NULL REFERENCES users(user_id),
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_note_blocks_case_id ON note_blocks(case_id);
CREATE INDEX IF NOT EXISTS idx_note_blocks_evidence_item_id ON note_blocks(evidence_item_id);
CREATE INDEX IF NOT EXISTS idx_note_blocks_amends_block_id ON note_blocks(amends_block_id);

CREATE TABLE IF NOT EXISTS tags (
    tag_id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#6B7280'
);

CREATE TABLE IF NOT EXISTS block_tags (
    block_id TEXT NOT NULL REFERENCES note_blocks(block_id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (block_id, tag_id)
);

CREATE TABLE IF NOT EXISTS ioc_entries (
    ioc_id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL REFERENCES cases(case_id) ON DELETE CASCADE,
    ioc_type TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    detection_method TEXT NOT NULL DEFAULT 'MANUAL',
    source_block_id TEXT REFERENCES note_blocks(block_id),
    created_by TEXT NOT NULL REFERENCES users(user_id),
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ioc_entries_case_id ON ioc_entries(case_id);

CREATE TABLE IF NOT EXISTS timeline_entries (
    entry_id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL REFERENCES cases(case_id) ON DELETE CASCADE,
    event_time TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_block_id TEXT REFERENCES note_blocks(block_id),
    created_by TEXT NOT NULL REFERENCES users(user_id),
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_timeline_entries_case_id ON timeline_entries(case_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    log_id TEXT PRIMARY KEY,
    case_id TEXT REFERENCES cases(case_id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(user_id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    details TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_case_id ON audit_logs(case_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);

-- Seed standard tags
INSERT OR IGNORE INTO tags (tag_id, name, color) VALUES
    ('tag-finding', 'Finding', '#EF4444'),
    ('tag-hypothesis', 'Hypothesis', '#F59E0B'),
    ('tag-evidence', 'Evidence', '#10B981'),
    ('tag-action-item', 'Action Item', '#3B82F6'),
    ('tag-ioc', 'IOC', '#8B5CF6'),
    ('tag-timeline', 'Timeline', '#EC4899'),
    ('tag-mitre', 'MITRE ATT&CK', '#F97316'),
    ('tag-remediation', 'Remediation', '#06B6D4');
