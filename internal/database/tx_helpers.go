package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"dfnotes-go/internal/models"
)

// execer is satisfied by both *sql.Tx and the pool. Writes that must share a
// transaction take an execer so they never reach for the pool while the
// single connection is held by that transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func insertCaseFact(ctx context.Context, ex execer, fact *models.CaseFact) error {
	_, err := ex.ExecContext(ctx,
		`INSERT INTO case_facts
		    (fact_id, case_id, evidence_item_id, type, label, value,
		     source_ioc_id, source_block_id, notes, created_at, updated_at, user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fact.FactID,
		fact.CaseID,
		nullString(fact.EvidenceItemID),
		fact.Type,
		fact.Label,
		fact.Value,
		nullString(fact.SourceIOCID),
		nullString(fact.SourceBlockID),
		fact.Notes,
		fact.CreatedAt,
		fact.UpdatedAt,
		fact.UserID,
	)
	return wrapError(err)
}

func deleteCaseFact(ctx context.Context, ex execer, factID string) error {
	result, err := ex.ExecContext(ctx,
		`DELETE FROM case_facts WHERE fact_id = ?`, factID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func insertAuditLog(ctx context.Context, ex execer, entry *models.AuditLog) error {
	var details *string
	if entry.Details != nil {
		s := string(entry.Details)
		details = &s
	}

	_, err := ex.ExecContext(ctx,
		`INSERT INTO audit_logs (log_id, case_id, user_id, action, entity_type, entity_id, details, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.LogID, nullString(entry.CaseID), entry.UserID, string(entry.Action),
		entry.EntityType, entry.EntityID, details, FormatTime(entry.CreatedAt),
	)
	return wrapError(err)
}

// writeAudit marshals details and inserts an audit entry on ex.
func writeAudit(ctx context.Context, ex execer, caseID, userID string, action models.AuditAction,
	entityType, entityID string, details any) error {
	raw, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details for %s %s: %w", entityType, entityID, err)
	}
	cid := caseID
	if err := insertAuditLog(ctx, ex, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &cid,
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    raw,
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("write %s audit for %s %s: %w", action, entityType, entityID, err)
	}
	return nil
}

// caseFactAuditDetails matches the details CaseFactService writes for fact
// CREATE and DELETE, so audit history reads the same whichever path wrote it.
func caseFactAuditDetails(fact *models.CaseFact) map[string]string {
	return map[string]string{"fact_id": fact.FactID, "type": fact.Type, "label": fact.Label}
}
