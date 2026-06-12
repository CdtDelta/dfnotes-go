CREATE TABLE IF NOT EXISTS case_facts (
    fact_id          TEXT PRIMARY KEY,
    case_id          TEXT NOT NULL REFERENCES cases(case_id),
    evidence_item_id TEXT REFERENCES evidence_items(evidence_item_id),
    type             TEXT NOT NULL,
    label            TEXT NOT NULL,
    value            TEXT NOT NULL,
    source_ioc_id    TEXT REFERENCES ioc_entries(ioc_id),
    source_block_id  TEXT REFERENCES note_blocks(block_id),
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    user_id          TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_case_facts_case_id ON case_facts(case_id);
