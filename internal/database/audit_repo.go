package database

import (
	"context"
	"database/sql"
	"encoding/json"

	"dfnotes-go/internal/models"
)

type AuditRepo struct {
	db *DB
}

func NewAuditRepo(db *DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Create(ctx context.Context, entry *models.AuditLog) error {
	return insertAuditLog(ctx, r.db, entry)
}

func (r *AuditRepo) ListByCase(ctx context.Context, caseID string) ([]models.AuditLog, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT log_id, case_id, user_id, action, entity_type, entity_id, details, created_at
		 FROM audit_logs WHERE case_id = ? ORDER BY created_at DESC`, caseID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()
	return r.scanEntries(rows)
}

func (r *AuditRepo) ListByUser(ctx context.Context, userID string) ([]models.AuditLog, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT log_id, case_id, user_id, action, entity_type, entity_id, details, created_at
		 FROM audit_logs WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()
	return r.scanEntries(rows)
}

func (r *AuditRepo) scanEntries(rows *sql.Rows) ([]models.AuditLog, error) {
	var entries []models.AuditLog
	for rows.Next() {
		var entry models.AuditLog
		var caseID sql.NullString
		var action, createdAt string
		var details sql.NullString

		err := rows.Scan(&entry.LogID, &caseID, &entry.UserID, &action,
			&entry.EntityType, &entry.EntityID, &details, &createdAt)
		if err != nil {
			return nil, wrapError(err)
		}

		entry.Action = models.AuditAction(action)
		if caseID.Valid {
			entry.CaseID = &caseID.String
		}
		if details.Valid {
			entry.Details = json.RawMessage(details.String)
		}
		entry.CreatedAt, _ = ParseTime(createdAt)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
