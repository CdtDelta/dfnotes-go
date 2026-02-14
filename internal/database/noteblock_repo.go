package database

import (
	"context"
	"database/sql"

	"dfnotes-go/internal/models"
)

type NoteBlockRepo struct {
	db *DB
}

func NewNoteBlockRepo(db *DB) *NoteBlockRepo {
	return &NoteBlockRepo{db: db}
}

func (r *NoteBlockRepo) Create(ctx context.Context, block *models.NoteBlock) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO note_blocks (block_id, case_id, evidence_item_id, amends_block_id, content_hash, prev_hash, signature, encrypted_body, author_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		block.BlockID, block.CaseID, nullString(block.EvidenceItemID), nullString(block.AmendsBlockID),
		block.ContentHash, block.PrevHash, block.Signature, block.EncryptedBody,
		block.AuthorID, FormatTime(block.CreatedAt),
	)
	return wrapError(err)
}

func (r *NoteBlockRepo) GetByID(ctx context.Context, blockID string) (*models.NoteBlock, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT block_id, case_id, evidence_item_id, amends_block_id, content_hash, prev_hash, signature, encrypted_body, author_id, created_at
		 FROM note_blocks WHERE block_id = ?`, blockID)

	block, err := r.scanBlock(row)
	if err != nil {
		return nil, err
	}

	tags, err := r.loadTags(ctx, block.BlockID)
	if err != nil {
		return nil, err
	}
	block.Tags = tags
	return block, nil
}

func (r *NoteBlockRepo) ListByCase(ctx context.Context, caseID string) ([]models.NoteBlock, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT block_id, case_id, evidence_item_id, amends_block_id, content_hash, prev_hash, signature, encrypted_body, author_id, created_at
		 FROM note_blocks WHERE case_id = ? ORDER BY created_at ASC`, caseID)
	if err != nil {
		return nil, wrapError(err)
	}

	var blocks []models.NoteBlock
	for rows.Next() {
		block, err := r.scanBlock(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		blocks = append(blocks, *block)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range blocks {
		tags, err := r.loadTags(ctx, blocks[i].BlockID)
		if err != nil {
			return nil, err
		}
		blocks[i].Tags = tags
	}
	return blocks, nil
}

func (r *NoteBlockRepo) ListByEvidence(ctx context.Context, evidenceItemID string) ([]models.NoteBlock, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT block_id, case_id, evidence_item_id, amends_block_id, content_hash, prev_hash, signature, encrypted_body, author_id, created_at
		 FROM note_blocks WHERE evidence_item_id = ? ORDER BY created_at ASC`, evidenceItemID)
	if err != nil {
		return nil, wrapError(err)
	}

	var blocks []models.NoteBlock
	for rows.Next() {
		block, err := r.scanBlock(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		blocks = append(blocks, *block)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range blocks {
		tags, err := r.loadTags(ctx, blocks[i].BlockID)
		if err != nil {
			return nil, err
		}
		blocks[i].Tags = tags
	}
	return blocks, nil
}

func (r *NoteBlockRepo) GetLastBlock(ctx context.Context, caseID string) (*models.NoteBlock, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT block_id, case_id, evidence_item_id, amends_block_id, content_hash, prev_hash, signature, encrypted_body, author_id, created_at
		 FROM note_blocks WHERE case_id = ? ORDER BY created_at DESC LIMIT 1`, caseID)

	block, err := r.scanBlock(row)
	if err != nil {
		return nil, err
	}

	tags, err := r.loadTags(ctx, block.BlockID)
	if err != nil {
		return nil, err
	}
	block.Tags = tags
	return block, nil
}

func (r *NoteBlockRepo) loadTags(ctx context.Context, blockID string) ([]models.Tag, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.tag_id, t.name, t.color FROM tags t
		 JOIN block_tags bt ON bt.tag_id = t.tag_id
		 WHERE bt.block_id = ?`, blockID)
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

func (r *NoteBlockRepo) scanBlock(s scanner) (*models.NoteBlock, error) {
	var block models.NoteBlock
	var evidenceItemID, amendsBlockID sql.NullString
	var createdAt string

	err := s.Scan(&block.BlockID, &block.CaseID, &evidenceItemID, &amendsBlockID,
		&block.ContentHash, &block.PrevHash, &block.Signature, &block.EncryptedBody,
		&block.AuthorID, &createdAt)
	if err != nil {
		return nil, wrapError(err)
	}

	if evidenceItemID.Valid {
		block.EvidenceItemID = &evidenceItemID.String
	}
	if amendsBlockID.Valid {
		block.AmendsBlockID = &amendsBlockID.String
	}
	block.CreatedAt, _ = ParseTime(createdAt)
	return &block, nil
}

func nullString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}
