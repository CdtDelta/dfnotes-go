package database

import (
	"context"
	"encoding/json"

	"dfnotes-go/internal/models"
)

type EvidenceRepo struct {
	db *DB
}

func NewEvidenceRepo(db *DB) *EvidenceRepo {
	return &EvidenceRepo{db: db}
}

func (r *EvidenceRepo) Create(ctx context.Context, item *models.EvidenceItem) error {
	custodyJSON, err := json.Marshal(item.CustodyLog)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO evidence_items (evidence_item_id, case_id, name, description, evidence_type, status, content_hash, custody_log, collected_by, collected_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.EvidenceItemID, item.CaseID, item.Name, item.Description,
		string(item.EvidenceType), string(item.Status), item.ContentHash,
		string(custodyJSON), item.CollectedBy,
		FormatTime(item.CollectedAt), FormatTime(item.CreatedAt), FormatTime(item.UpdatedAt),
	)
	return wrapError(err)
}

func (r *EvidenceRepo) GetByID(ctx context.Context, evidenceItemID string) (*models.EvidenceItem, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT evidence_item_id, case_id, name, description, evidence_type, status, content_hash, custody_log, collected_by, collected_at, created_at, updated_at
		 FROM evidence_items WHERE evidence_item_id = ?`, evidenceItemID)

	item, err := r.scanItem(row)
	if err != nil {
		return nil, err
	}

	tags, err := r.loadTags(ctx, item.EvidenceItemID)
	if err != nil {
		return nil, err
	}
	item.Tags = tags
	return item, nil
}

func (r *EvidenceRepo) ListByCase(ctx context.Context, caseID string) ([]models.EvidenceItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT evidence_item_id, case_id, name, description, evidence_type, status, content_hash, custody_log, collected_by, collected_at, created_at, updated_at
		 FROM evidence_items WHERE case_id = ? ORDER BY created_at DESC`, caseID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var items []models.EvidenceItem
	for rows.Next() {
		item, err := r.scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range items {
		tags, err := r.loadTags(ctx, items[i].EvidenceItemID)
		if err != nil {
			return nil, err
		}
		items[i].Tags = tags
	}
	return items, nil
}

func (r *EvidenceRepo) Update(ctx context.Context, item *models.EvidenceItem) error {
	custodyJSON, err := json.Marshal(item.CustodyLog)
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE evidence_items SET name=?, description=?, evidence_type=?, status=?, content_hash=?, custody_log=?, updated_at=?
		 WHERE evidence_item_id=?`,
		item.Name, item.Description, string(item.EvidenceType), string(item.Status),
		item.ContentHash, string(custodyJSON), FormatTime(item.UpdatedAt), item.EvidenceItemID,
	)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *EvidenceRepo) Delete(ctx context.Context, evidenceItemID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM evidence_items WHERE evidence_item_id = ?`, evidenceItemID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *EvidenceRepo) loadTags(ctx context.Context, evidenceItemID string) ([]models.Tag, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.tag_id, t.name, t.color FROM tags t
		 JOIN evidence_tags et ON et.tag_id = t.tag_id
		 WHERE et.evidence_item_id = ?`, evidenceItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.TagID, &tag.Name, &tag.Color); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (r *EvidenceRepo) scanItem(s scanner) (*models.EvidenceItem, error) {
	var item models.EvidenceItem
	var evType, status, custodyJSON, collectedAt, createdAt, updatedAt string

	err := s.Scan(&item.EvidenceItemID, &item.CaseID, &item.Name, &item.Description,
		&evType, &status, &item.ContentHash, &custodyJSON,
		&item.CollectedBy, &collectedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, wrapError(err)
	}

	item.EvidenceType = models.EvidenceType(evType)
	item.Status = models.EvidenceStatus(status)
	item.CollectedAt, _ = ParseTime(collectedAt)
	item.CreatedAt, _ = ParseTime(createdAt)
	item.UpdatedAt, _ = ParseTime(updatedAt)

	if err := json.Unmarshal([]byte(custodyJSON), &item.CustodyLog); err != nil {
		item.CustodyLog = []models.CustodyEntry{}
	}

	return &item, nil
}
