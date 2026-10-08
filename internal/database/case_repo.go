package database

import (
	"context"

	"dfnotes-go/internal/models"
)

type CaseRepo struct {
	db *DB
}

func NewCaseRepo(db *DB) *CaseRepo {
	return &CaseRepo{db: db}
}

func (r *CaseRepo) Create(ctx context.Context, c *models.Case) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO cases (case_id, case_number, title, description, classification, ticket_number, examiner_name, organization, evidence_prefix, evidence_seq_digits, salt, encrypted_key, created_by, created_at, updated_at, attorney_client_privilege, case_password_required)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.CaseID, c.CaseNumber, c.Title, c.Description, string(c.Classification),
		c.TicketNumber, c.ExaminerName, c.Organization,
		c.EvidencePrefix, c.EvidenceSeqDigits,
		nullBytes(c.Salt), nullBytes(c.EncryptedKey),
		c.CreatedBy, FormatTime(c.CreatedAt), FormatTime(c.UpdatedAt),
		// The column is case_password_required (DEFAULT 1, fail-closed in SQL)
		// while the model field is NoCasePassword (zero value fail-closed in Go).
		// This repo is the only place the two meet, so the inversion is deliberate.
		c.AttorneyClientPrivilege, !c.NoCasePassword,
	)
	return wrapError(err)
}

func (r *CaseRepo) GetByID(ctx context.Context, caseID string) (*models.Case, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT case_id, case_number, title, description, classification, ticket_number, examiner_name, organization, evidence_prefix, evidence_seq_digits, salt, encrypted_key, created_by, created_at, updated_at, attorney_client_privilege, case_password_required
		 FROM cases WHERE case_id = ?`, caseID)

	var c models.Case
	var classification, createdAt, updatedAt string
	var passwordRequired bool

	err := row.Scan(&c.CaseID, &c.CaseNumber, &c.Title, &c.Description, &classification,
		&c.TicketNumber, &c.ExaminerName, &c.Organization,
		&c.EvidencePrefix, &c.EvidenceSeqDigits,
		&c.Salt, &c.EncryptedKey,
		&c.CreatedBy, &createdAt, &updatedAt, &c.AttorneyClientPrivilege, &passwordRequired)
	if err != nil {
		return nil, wrapError(err)
	}
	// Deliberate inversion, fail-closed; see the comment in Create.
	c.NoCasePassword = !passwordRequired

	c.Classification = models.ClassificationLevel(classification)
	c.CreatedAt, _ = ParseTime(createdAt)
	c.UpdatedAt, _ = ParseTime(updatedAt)
	return &c, nil
}

func (r *CaseRepo) List(ctx context.Context) ([]models.Case, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT case_id, case_number, title, description, classification, ticket_number, examiner_name, organization, evidence_prefix, evidence_seq_digits, salt, encrypted_key, created_by, created_at, updated_at, attorney_client_privilege, case_password_required
		 FROM cases ORDER BY created_at DESC`)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var cases []models.Case
	for rows.Next() {
		var c models.Case
		var classification, createdAt, updatedAt string
		var passwordRequired bool
		if err := rows.Scan(&c.CaseID, &c.CaseNumber, &c.Title, &c.Description, &classification,
			&c.TicketNumber, &c.ExaminerName, &c.Organization,
			&c.EvidencePrefix, &c.EvidenceSeqDigits,
			&c.Salt, &c.EncryptedKey,
			&c.CreatedBy, &createdAt, &updatedAt, &c.AttorneyClientPrivilege, &passwordRequired); err != nil {
			return nil, wrapError(err)
		}
		// Deliberate inversion, fail-closed; see the comment in Create.
		c.NoCasePassword = !passwordRequired
		c.Classification = models.ClassificationLevel(classification)
		c.CreatedAt, _ = ParseTime(createdAt)
		c.UpdatedAt, _ = ParseTime(updatedAt)
		cases = append(cases, c)
	}
	return cases, rows.Err()
}

func (r *CaseRepo) Update(ctx context.Context, c *models.Case) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE cases SET case_number=?, title=?, description=?, classification=?, ticket_number=?, examiner_name=?, organization=?, salt=?, encrypted_key=?, updated_at=? WHERE case_id=?`,
		c.CaseNumber, c.Title, c.Description, string(c.Classification),
		c.TicketNumber, c.ExaminerName, c.Organization,
		nullBytes(c.Salt), nullBytes(c.EncryptedKey),
		FormatTime(c.UpdatedAt), c.CaseID,
	)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *CaseRepo) UpdateAttorneyClientPrivilege(ctx context.Context, caseID string, value bool, updatedAt string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE cases SET attorney_client_privilege=?, updated_at=? WHERE case_id=?`,
		value, updatedAt, caseID,
	)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *CaseRepo) Delete(ctx context.Context, caseID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM cases WHERE case_id = ?`, caseID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}
