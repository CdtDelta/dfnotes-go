package database

import (
	"context"
	"database/sql"

	"dfnotes-go/internal/models"
)

type TimelineRepo struct {
	db *DB
}

func NewTimelineRepo(db *DB) *TimelineRepo {
	return &TimelineRepo{db: db}
}

func (r *TimelineRepo) Create(ctx context.Context, entry *models.TimelineEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO timeline_entries (entry_id, case_id, event_time, title, description, source_block_id, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.EntryID, entry.CaseID, FormatTime(entry.EventTime), entry.Title, entry.Description,
		nullString(entry.SourceBlockID), entry.CreatedBy, FormatTime(entry.CreatedAt),
	)
	return wrapError(err)
}

func (r *TimelineRepo) GetByID(ctx context.Context, entryID string) (*models.TimelineEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT entry_id, case_id, event_time, title, description, source_block_id, created_by, created_at
		 FROM timeline_entries WHERE entry_id = ?`, entryID)

	return r.scanEntry(row)
}

func (r *TimelineRepo) ListByCase(ctx context.Context, caseID string) ([]models.TimelineEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT entry_id, case_id, event_time, title, description, source_block_id, created_by, created_at
		 FROM timeline_entries WHERE case_id = ? ORDER BY event_time ASC`, caseID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var entries []models.TimelineEntry
	for rows.Next() {
		entry, err := r.scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, rows.Err()
}

func (r *TimelineRepo) Delete(ctx context.Context, entryID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM timeline_entries WHERE entry_id = ?`, entryID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *TimelineRepo) scanEntry(s scanner) (*models.TimelineEntry, error) {
	var entry models.TimelineEntry
	var eventTime, createdAt string
	var sourceBlockID sql.NullString

	err := s.Scan(&entry.EntryID, &entry.CaseID, &eventTime, &entry.Title, &entry.Description,
		&sourceBlockID, &entry.CreatedBy, &createdAt)
	if err != nil {
		return nil, wrapError(err)
	}

	entry.EventTime, _ = ParseTime(eventTime)
	if sourceBlockID.Valid {
		entry.SourceBlockID = &sourceBlockID.String
	}
	entry.CreatedAt, _ = ParseTime(createdAt)
	return &entry, nil
}
