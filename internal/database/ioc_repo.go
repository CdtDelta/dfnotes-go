package database

import (
	"context"
	"database/sql"

	"dfnotes-go/internal/ioc"
)

type IOCRepo struct {
	db *DB
}

func NewIOCRepo(db *DB) *IOCRepo {
	return &IOCRepo{db: db}
}

func (r *IOCRepo) Create(ctx context.Context, entry *ioc.IOCEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ioc_entries
		    (ioc_id, case_id, block_id, evidence_item_id, type, value, status,
		     detection_method, notes, created_at, confirmed_at, user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.IOCID,
		entry.CaseID,
		entry.BlockID,
		nullString(entry.EvidenceItemID),
		string(entry.Type),
		entry.Value,
		string(entry.Status),
		entry.DetectionMethod,
		nullString(entry.Notes),
		entry.CreatedAt,
		nullString(entry.ConfirmedAt),
		entry.UserID,
	)
	return wrapError(err)
}

func (r *IOCRepo) GetByID(ctx context.Context, iocID string) (*ioc.IOCEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT ioc_id, case_id, block_id, evidence_item_id, type, value, status,
		        detection_method, notes, created_at, confirmed_at, user_id
		 FROM ioc_entries WHERE ioc_id = ?`, iocID)
	return r.scanEntry(row)
}

func (r *IOCRepo) GetByBlock(ctx context.Context, blockID string) ([]ioc.IOCEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ioc_id, case_id, block_id, evidence_item_id, type, value, status,
		        detection_method, notes, created_at, confirmed_at, user_id
		 FROM ioc_entries WHERE block_id = ? ORDER BY created_at ASC`, blockID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()
	return r.scanEntries(rows)
}

func (r *IOCRepo) ListByCase(ctx context.Context, caseID string, includeAll bool) ([]ioc.IOCEntry, error) {
	query := `SELECT ioc_id, case_id, block_id, evidence_item_id, type, value, status,
		         detection_method, notes, created_at, confirmed_at, user_id
		  FROM ioc_entries WHERE case_id = ?`
	args := []interface{}{caseID}

	if !includeAll {
		query += ` AND status != 'false_positive'`
	}
	query += ` ORDER BY
		CASE status WHEN 'confirmed' THEN 0 WHEN 'detected' THEN 1 ELSE 2 END,
		created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()
	return r.scanEntries(rows)
}

func (r *IOCRepo) UpdateStatus(ctx context.Context, iocID string, status ioc.IOCStatus, confirmedAt *string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE ioc_entries SET status = ?, confirmed_at = ? WHERE ioc_id = ?`,
		string(status), nullString(confirmedAt), iocID,
	)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *IOCRepo) GetExistingByBlock(ctx context.Context, blockID string) (map[string]struct{}, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT type, value FROM ioc_entries WHERE block_id = ?`, blockID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	for rows.Next() {
		var t, v string
		if err := rows.Scan(&t, &v); err != nil {
			return nil, wrapError(err)
		}
		existing[t+":"+v] = struct{}{}
	}
	return existing, rows.Err()
}

func (r *IOCRepo) scanEntries(rows *sql.Rows) ([]ioc.IOCEntry, error) {
	var entries []ioc.IOCEntry
	for rows.Next() {
		entry, err := r.scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	if entries == nil {
		entries = []ioc.IOCEntry{}
	}
	return entries, rows.Err()
}

func (r *IOCRepo) scanEntry(s scanner) (*ioc.IOCEntry, error) {
	var entry ioc.IOCEntry
	var iocType, status string
	var evidenceItemID, notes, confirmedAt sql.NullString

	err := s.Scan(
		&entry.IOCID, &entry.CaseID, &entry.BlockID,
		&evidenceItemID, &iocType, &entry.Value, &status,
		&entry.DetectionMethod, &notes, &entry.CreatedAt,
		&confirmedAt, &entry.UserID,
	)
	if err != nil {
		return nil, wrapError(err)
	}

	entry.Type = ioc.IOCType(iocType)
	entry.Status = ioc.IOCStatus(status)
	if evidenceItemID.Valid {
		entry.EvidenceItemID = &evidenceItemID.String
	}
	if notes.Valid {
		entry.Notes = &notes.String
	}
	if confirmedAt.Valid {
		entry.ConfirmedAt = &confirmedAt.String
	}
	return &entry, nil
}

