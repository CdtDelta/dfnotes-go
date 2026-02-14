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
		`INSERT INTO cases (case_id, case_number, title, description, classification, ticket_number, examiner_name, organization, salt, encrypted_key, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.CaseID, c.CaseNumber, c.Title, c.Description, string(c.Classification),
		c.TicketNumber, c.ExaminerName, c.Organization,
		nullBytes(c.Salt), nullBytes(c.EncryptedKey),
		c.CreatedBy, FormatTime(c.CreatedAt), FormatTime(c.UpdatedAt),
	)
	return wrapError(err)
}

func (r *CaseRepo) GetByID(ctx context.Context, caseID string) (*models.Case, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT case_id, case_number, title, description, classification, ticket_number, examiner_name, organization, salt, encrypted_key, created_by, created_at, updated_at
		 FROM cases WHERE case_id = ?`, caseID)

	var c models.Case
	var classification, createdAt, updatedAt string

	err := row.Scan(&c.CaseID, &c.CaseNumber, &c.Title, &c.Description, &classification,
		&c.TicketNumber, &c.ExaminerName, &c.Organization,
		&c.Salt, &c.EncryptedKey,
		&c.CreatedBy, &createdAt, &updatedAt)
	if err != nil {
		return nil, wrapError(err)
	}

	c.Classification = models.ClassificationLevel(classification)
	c.CreatedAt, _ = ParseTime(createdAt)
	c.UpdatedAt, _ = ParseTime(updatedAt)
	return &c, nil
}

func (r *CaseRepo) List(ctx context.Context) ([]models.Case, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT case_id, case_number, title, description, classification, ticket_number, examiner_name, organization, salt, encrypted_key, created_by, created_at, updated_at
		 FROM cases ORDER BY created_at DESC`)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var cases []models.Case
	for rows.Next() {
		var c models.Case
		var classification, createdAt, updatedAt string
		if err := rows.Scan(&c.CaseID, &c.CaseNumber, &c.Title, &c.Description, &classification,
			&c.TicketNumber, &c.ExaminerName, &c.Organization,
			&c.Salt, &c.EncryptedKey,
			&c.CreatedBy, &createdAt, &updatedAt); err != nil {
			return nil, wrapError(err)
		}
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

func (r *CaseRepo) Delete(ctx context.Context, caseID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM cases WHERE case_id = ?`, caseID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}
