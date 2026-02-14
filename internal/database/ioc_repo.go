package database

import (
	"context"
	"database/sql"

	"dfnotes-go/internal/models"
)

type IOCRepo struct {
	db *DB
}

func NewIOCRepo(db *DB) *IOCRepo {
	return &IOCRepo{db: db}
}

func (r *IOCRepo) Create(ctx context.Context, entry *models.IOCEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ioc_entries (ioc_id, case_id, ioc_type, value, description, detection_method, source_block_id, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.IOCID, entry.CaseID, string(entry.IOCType), entry.Value, entry.Description,
		string(entry.DetectionMethod), nullString(entry.SourceBlockID),
		entry.CreatedBy, FormatTime(entry.CreatedAt),
	)
	return wrapError(err)
}

func (r *IOCRepo) GetByID(ctx context.Context, iocID string) (*models.IOCEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT ioc_id, case_id, ioc_type, value, description, detection_method, source_block_id, created_by, created_at
		 FROM ioc_entries WHERE ioc_id = ?`, iocID)

	return r.scanEntry(row)
}

func (r *IOCRepo) ListByCase(ctx context.Context, caseID string) ([]models.IOCEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ioc_id, case_id, ioc_type, value, description, detection_method, source_block_id, created_by, created_at
		 FROM ioc_entries WHERE case_id = ? ORDER BY created_at DESC`, caseID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var entries []models.IOCEntry
	for rows.Next() {
		entry, err := r.scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, rows.Err()
}

func (r *IOCRepo) Delete(ctx context.Context, iocID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM ioc_entries WHERE ioc_id = ?`, iocID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *IOCRepo) scanEntry(s scanner) (*models.IOCEntry, error) {
	var entry models.IOCEntry
	var iocType, detection, createdAt string
	var sourceBlockID sql.NullString

	err := s.Scan(&entry.IOCID, &entry.CaseID, &iocType, &entry.Value, &entry.Description,
		&detection, &sourceBlockID, &entry.CreatedBy, &createdAt)
	if err != nil {
		return nil, wrapError(err)
	}

	entry.IOCType = models.IOCType(iocType)
	entry.DetectionMethod = models.DetectionMethod(detection)
	if sourceBlockID.Valid {
		entry.SourceBlockID = &sourceBlockID.String
	}
	entry.CreatedAt, _ = ParseTime(createdAt)
	return &entry, nil
}
