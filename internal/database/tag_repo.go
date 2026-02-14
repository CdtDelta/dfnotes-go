package database

import (
	"context"

	"dfnotes-go/internal/models"
)

type TagRepo struct {
	db *DB
}

func NewTagRepo(db *DB) *TagRepo {
	return &TagRepo{db: db}
}

func (r *TagRepo) Create(ctx context.Context, tag *models.Tag) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tags (tag_id, name, color) VALUES (?, ?, ?)`,
		tag.TagID, tag.Name, tag.Color,
	)
	return wrapError(err)
}

func (r *TagRepo) GetByID(ctx context.Context, tagID string) (*models.Tag, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT tag_id, name, color FROM tags WHERE tag_id = ?`, tagID)

	var tag models.Tag
	err := row.Scan(&tag.TagID, &tag.Name, &tag.Color)
	if err != nil {
		return nil, wrapError(err)
	}
	return &tag, nil
}

func (r *TagRepo) List(ctx context.Context) ([]models.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tag_id, name, color FROM tags ORDER BY name`)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.TagID, &tag.Name, &tag.Color); err != nil {
			return nil, wrapError(err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *TagRepo) Delete(ctx context.Context, tagID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE tag_id = ?`, tagID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *TagRepo) AttachToBlock(ctx context.Context, blockID, tagID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO block_tags (block_id, tag_id) VALUES (?, ?)`, blockID, tagID)
	return wrapError(err)
}

func (r *TagRepo) DetachFromBlock(ctx context.Context, blockID, tagID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM block_tags WHERE block_id = ? AND tag_id = ?`, blockID, tagID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *TagRepo) ListByBlock(ctx context.Context, blockID string) ([]models.Tag, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.tag_id, t.name, t.color FROM tags t
		 JOIN block_tags bt ON bt.tag_id = t.tag_id
		 WHERE bt.block_id = ?`, blockID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.TagID, &tag.Name, &tag.Color); err != nil {
			return nil, wrapError(err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *TagRepo) AttachToEvidence(ctx context.Context, evidenceItemID, tagID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO evidence_tags (evidence_item_id, tag_id) VALUES (?, ?)`, evidenceItemID, tagID)
	return wrapError(err)
}

func (r *TagRepo) DetachFromEvidence(ctx context.Context, evidenceItemID, tagID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM evidence_tags WHERE evidence_item_id = ? AND tag_id = ?`, evidenceItemID, tagID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *TagRepo) ListByEvidence(ctx context.Context, evidenceItemID string) ([]models.Tag, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.tag_id, t.name, t.color FROM tags t
		 JOIN evidence_tags et ON et.tag_id = t.tag_id
		 WHERE et.evidence_item_id = ?`, evidenceItemID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.TagID, &tag.Name, &tag.Color); err != nil {
			return nil, wrapError(err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}
