-- Evidence-to-tag join table
CREATE TABLE IF NOT EXISTS evidence_tags (
    evidence_item_id TEXT NOT NULL REFERENCES evidence_items(evidence_item_id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (evidence_item_id, tag_id)
);

-- Attachments for pasted images
CREATE TABLE IF NOT EXISTS attachments (
    attachment_id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL REFERENCES cases(case_id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    encrypted_data BLOB NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_attachments_case_id ON attachments(case_id);

-- Seed additional standard tags
INSERT OR IGNORE INTO tags (tag_id, name, color) VALUES
    ('tag-malware', 'Malware', '#DC2626'),
    ('tag-phishing', 'Phishing', '#D97706'),
    ('tag-data-exfil', 'Data Exfiltration', '#B91C1C'),
    ('tag-unauth-access', 'Unauthorized Access', '#9333EA'),
    ('tag-insider-threat', 'Insider Threat', '#BE185D'),
    ('tag-ransomware', 'Ransomware', '#E11D48'),
    ('tag-follow-up', 'Follow Up', '#2563EB'),
    ('tag-confirmed', 'Confirmed', '#059669'),
    ('tag-unconfirmed', 'Unconfirmed', '#D97706'),
    ('tag-false-positive', 'False Positive', '#6B7280'),
    ('tag-critical', 'Critical', '#991B1B'),
    ('tag-high', 'High', '#DC2626'),
    ('tag-medium', 'Medium', '#F59E0B'),
    ('tag-low', 'Low', '#3B82F6'),
    ('tag-disk-image', 'Disk Image', '#1D4ED8'),
    ('tag-memory-dump', 'Memory Dump', '#7C3AED'),
    ('tag-network-capture', 'Network Capture', '#0891B2'),
    ('tag-log-file', 'Log File', '#CA8A04'),
    ('tag-mobile-device', 'Mobile Device', '#0D9488'),
    ('tag-cloud-data', 'Cloud Data', '#6366F1');
