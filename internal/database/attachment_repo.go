package database

import (
	"context"

	"dfnotes-go/internal/models"
)

type AttachmentRepo struct {
	db *DB
}

func NewAttachmentRepo(db *DB) *AttachmentRepo {
	return &AttachmentRepo{db: db}
}

func (r *AttachmentRepo) Create(ctx context.Context, att *models.Attachment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO attachments (attachment_id, case_id, filename, content_type, encrypted_data, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		att.AttachmentID, att.CaseID, att.Filename, att.ContentType, att.EncryptedData, FormatTime(att.CreatedAt),
	)
	return wrapError(err)
}

func (r *AttachmentRepo) GetByID(ctx context.Context, attachmentID string) (*models.Attachment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT attachment_id, case_id, filename, content_type, encrypted_data, created_at
		 FROM attachments WHERE attachment_id = ?`, attachmentID)

	var att models.Attachment
	var createdAt string
	err := row.Scan(&att.AttachmentID, &att.CaseID, &att.Filename, &att.ContentType, &att.EncryptedData, &createdAt)
	if err != nil {
		return nil, wrapError(err)
	}
	att.CreatedAt, _ = ParseTime(createdAt)
	return &att, nil
}
