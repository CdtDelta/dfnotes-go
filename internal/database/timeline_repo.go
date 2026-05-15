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
		`INSERT INTO timeline_entries
		    (entry_id, case_id, evidence_item_id, timestamp, display_timezone,
		     event_description, investigator_notes, created_at, updated_at, user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.EntryID,
		entry.CaseID,
		nullString(entry.EvidenceItemID),
		entry.Timestamp,
		nullString(entry.DisplayTimezone),
		entry.EventDescription,
		entry.InvestigatorNotes,
		entry.CreatedAt,
		entry.UpdatedAt,
		entry.UserID,
	)
	return wrapError(err)
}

func (r *TimelineRepo) GetByID(ctx context.Context, entryID string) (*models.TimelineEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT entry_id, case_id, evidence_item_id, timestamp, display_timezone,
		        event_description, investigator_notes, created_at, updated_at, user_id
		 FROM timeline_entries WHERE entry_id = ?`, entryID)
	return r.scanEntry(row)
}

func (r *TimelineRepo) ListByCase(ctx context.Context, caseID string) ([]models.TimelineEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT entry_id, case_id, evidence_item_id, timestamp, display_timezone,
		        event_description, investigator_notes, created_at, updated_at, user_id
		 FROM timeline_entries WHERE case_id = ? ORDER BY timestamp ASC`, caseID)
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
	if entries == nil {
		entries = []models.TimelineEntry{}
	}
	return entries, rows.Err()
}

func (r *TimelineRepo) Update(ctx context.Context, entry *models.TimelineEntry) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE timeline_entries
		 SET timestamp = ?, display_timezone = ?, event_description = ?,
		     investigator_notes = ?, updated_at = ?
		 WHERE entry_id = ?`,
		entry.Timestamp,
		nullString(entry.DisplayTimezone),
		entry.EventDescription,
		entry.InvestigatorNotes,
		entry.UpdatedAt,
		entry.EntryID,
	)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
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
	var evidenceItemID, displayTimezone sql.NullString

	err := s.Scan(
		&entry.EntryID, &entry.CaseID, &evidenceItemID, &entry.Timestamp,
		&displayTimezone, &entry.EventDescription, &entry.InvestigatorNotes,
		&entry.CreatedAt, &entry.UpdatedAt, &entry.UserID,
	)
	if err != nil {
		return nil, wrapError(err)
	}

	if evidenceItemID.Valid {
		entry.EvidenceItemID = &evidenceItemID.String
	}
	if displayTimezone.Valid {
		entry.DisplayTimezone = &displayTimezone.String
	}
	return &entry, nil
}
