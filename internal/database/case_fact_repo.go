package database

import (
	"context"
	"database/sql"

	"dfnotes-go/internal/models"
)

type CaseFactRepo struct {
	db *DB
}

func NewCaseFactRepo(db *DB) *CaseFactRepo {
	return &CaseFactRepo{db: db}
}

func (r *CaseFactRepo) Create(ctx context.Context, fact models.CaseFact) error {
	_, err := r.db.ExecContext(ctx,
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

func (r *CaseFactRepo) GetByCase(ctx context.Context, caseID string) ([]models.CaseFact, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT fact_id, case_id, evidence_item_id, type, label, value,
		        source_ioc_id, source_block_id, notes, created_at, updated_at, user_id
		 FROM case_facts WHERE case_id = ? ORDER BY created_at ASC`, caseID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()
	return r.scanFacts(rows)
}

func (r *CaseFactRepo) GetByID(ctx context.Context, factID string) (models.CaseFact, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT fact_id, case_id, evidence_item_id, type, label, value,
		        source_ioc_id, source_block_id, notes, created_at, updated_at, user_id
		 FROM case_facts WHERE fact_id = ?`, factID)
	fact, err := r.scanFact(row)
	if err != nil {
		return models.CaseFact{}, err
	}
	return *fact, nil
}

func (r *CaseFactRepo) Update(ctx context.Context, fact models.CaseFact) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE case_facts
		 SET evidence_item_id = ?, type = ?, label = ?, value = ?,
		     source_block_id = ?, notes = ?, updated_at = ?
		 WHERE fact_id = ?`,
		nullString(fact.EvidenceItemID),
		fact.Type,
		fact.Label,
		fact.Value,
		nullString(fact.SourceBlockID),
		fact.Notes,
		fact.UpdatedAt,
		fact.FactID,
	)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *CaseFactRepo) Delete(ctx context.Context, factID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM case_facts WHERE fact_id = ?`, factID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *CaseFactRepo) scanFacts(rows *sql.Rows) ([]models.CaseFact, error) {
	var facts []models.CaseFact
	for rows.Next() {
		fact, err := r.scanFact(rows)
		if err != nil {
			return nil, err
		}
		facts = append(facts, *fact)
	}
	if facts == nil {
		facts = []models.CaseFact{}
	}
	return facts, rows.Err()
}

func (r *CaseFactRepo) scanFact(s scanner) (*models.CaseFact, error) {
	var fact models.CaseFact
	var evidenceItemID, sourceIOCID, sourceBlockID sql.NullString

	err := s.Scan(
		&fact.FactID, &fact.CaseID, &evidenceItemID,
		&fact.Type, &fact.Label, &fact.Value,
		&sourceIOCID, &sourceBlockID, &fact.Notes,
		&fact.CreatedAt, &fact.UpdatedAt, &fact.UserID,
	)
	if err != nil {
		return nil, wrapError(err)
	}

	if evidenceItemID.Valid {
		fact.EvidenceItemID = &evidenceItemID.String
	}
	if sourceIOCID.Valid {
		fact.SourceIOCID = &sourceIOCID.String
	}
	if sourceBlockID.Valid {
		fact.SourceBlockID = &sourceBlockID.String
	}
	return &fact, nil
}
