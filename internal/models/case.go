package models

import "time"

type ClassificationLevel string

const (
	ClassificationUnclassified ClassificationLevel = "UNCLASSIFIED"
	ClassificationConfidential ClassificationLevel = "CONFIDENTIAL"
	ClassificationSecret       ClassificationLevel = "SECRET"
	ClassificationTopSecret    ClassificationLevel = "TOP SECRET"
)

type Case struct {
	CaseID         string              `json:"case_id"`
	CaseNumber     string              `json:"case_number"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	Classification ClassificationLevel `json:"classification"`
	TicketNumber   string              `json:"ticket_number"`
	ExaminerName   string              `json:"examiner_name"`
	Organization   string              `json:"organization"`
	Salt           []byte              `json:"salt"`
	EncryptedKey   []byte              `json:"encrypted_key"`
	CreatedBy      string              `json:"created_by"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}
