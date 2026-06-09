package models

import "time"

type ClassificationLevel string

const (
	ClassificationUnclassified ClassificationLevel = "UNCLASSIFIED"
	ClassificationConfidential ClassificationLevel = "CONFIDENTIAL"
	ClassificationRestricted   ClassificationLevel = "RESTRICTED"
	ClassificationSecret       ClassificationLevel = "SECRET"
	ClassificationTopSecret    ClassificationLevel = "TOP SECRET"
)

type Case struct {
	CaseID              string              `json:"case_id"`
	CaseNumber          string              `json:"case_number"`
	Title               string              `json:"title"`
	Description         string              `json:"description"`
	Classification      ClassificationLevel `json:"classification"`
	TicketNumber        string              `json:"ticket_number"`
	ExaminerName        string              `json:"examiner_name"`
	Organization        string              `json:"organization"`
	EvidencePrefix      string              `json:"evidence_prefix"`
	EvidenceSeqDigits   int                 `json:"evidence_seq_digits"`
	Salt                []byte              `json:"salt"`
	EncryptedKey        []byte              `json:"encrypted_key"`
	CreatedBy                 string              `json:"created_by"`
	CreatedAt                 time.Time           `json:"created_at"`
	UpdatedAt                 time.Time           `json:"updated_at"`
	AttorneyClientPrivilege   bool                `json:"attorney_client_privilege"`
}
