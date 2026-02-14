package models

import "time"

type EvidenceStatus string

const (
	EvidenceStatusCollected EvidenceStatus = "COLLECTED"
	EvidenceStatusAnalyzing EvidenceStatus = "ANALYZING"
	EvidenceStatusProcessed EvidenceStatus = "PROCESSED"
	EvidenceStatusArchived  EvidenceStatus = "ARCHIVED"
	EvidenceStatusWithdrawn EvidenceStatus = "WITHDRAWN"
)

type EvidenceType string

const (
	EvidenceTypeDisk    EvidenceType = "DISK"
	EvidenceTypeMemory  EvidenceType = "MEMORY"
	EvidenceTypeNetwork EvidenceType = "NETWORK"
	EvidenceTypeLogs    EvidenceType = "LOGS"
	EvidenceTypeMalware EvidenceType = "MALWARE"
	EvidenceTypeOther   EvidenceType = "OTHER"
)

type CustodyEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Handler     string    `json:"handler"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
}

type EvidenceItem struct {
	EvidenceItemID string           `json:"evidence_item_id"`
	CaseID         string           `json:"case_id"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	EvidenceType   EvidenceType     `json:"evidence_type"`
	Status         EvidenceStatus   `json:"status"`
	ContentHash    string           `json:"content_hash"`
	CustodyLog     []CustodyEntry   `json:"custody_log"`
	CollectedBy    string           `json:"collected_by"`
	CollectedAt    time.Time        `json:"collected_at"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	Tags           []Tag            `json:"tags,omitempty"`
}
