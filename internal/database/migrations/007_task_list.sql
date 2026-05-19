CREATE TABLE IF NOT EXISTS tasks (
    task_id          TEXT PRIMARY KEY,
    case_id          TEXT NOT NULL REFERENCES cases(case_id) ON DELETE CASCADE,
    evidence_item_id TEXT REFERENCES evidence_items(evidence_item_id),
    user_id          TEXT NOT NULL,
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'open'
                     CHECK(status IN ('open','in_progress','blocked','complete','not_applicable')),
    template_name    TEXT,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    completed_at     TEXT
);

CREATE TABLE IF NOT EXISTS task_note_links (
    task_id   TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    block_id  TEXT NOT NULL REFERENCES note_blocks(block_id) ON DELETE CASCADE,
    linked_at TEXT NOT NULL,
    PRIMARY KEY (task_id, block_id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_case_id ON tasks(case_id);
CREATE INDEX IF NOT EXISTS idx_tasks_evidence_item_id ON tasks(evidence_item_id);
CREATE INDEX IF NOT EXISTS idx_task_note_links_task_id ON task_note_links(task_id);
CREATE INDEX IF NOT EXISTS idx_task_note_links_block_id ON task_note_links(block_id);
