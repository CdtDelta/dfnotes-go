package models

import "time"

type Attachment struct {
	AttachmentID  string    `json:"attachment_id"`
	CaseID        string    `json:"case_id"`
	Filename      string    `json:"filename"`
	ContentType   string    `json:"content_type"`
	EncryptedData []byte    `json:"encrypted_data"`
	CreatedAt     time.Time `json:"created_at"`
}
