ALTER TABLE evidence_items ADD COLUMN item_number TEXT NOT NULL DEFAULT '';

WITH numbered AS (
    SELECT
        e.evidence_item_id,
        printf(
            '%s%0' || c.evidence_seq_digits || 'd',
            c.evidence_prefix,
            ROW_NUMBER() OVER (PARTITION BY e.case_id ORDER BY e.created_at)
        ) AS new_item_number
    FROM evidence_items e
    JOIN cases c ON e.case_id = c.case_id
)
UPDATE evidence_items
SET item_number = (
    SELECT new_item_number FROM numbered
    WHERE numbered.evidence_item_id = evidence_items.evidence_item_id
);
