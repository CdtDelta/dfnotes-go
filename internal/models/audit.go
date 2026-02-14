package models

import (
	"encoding/json"
	"time"
)

type AuditAction string

const (
	AuditActionCreate     AuditAction = "CREATE"
	AuditActionUpdate     AuditAction = "UPDATE"
	AuditActionDelete     AuditAction = "DELETE"
	AuditActionLogin      AuditAction = "LOGIN"
	AuditActionLogout     AuditAction = "LOGOUT"
	AuditActionExport     AuditAction = "EXPORT"
	AuditActionImport     AuditAction = "IMPORT"
	AuditActionSign       AuditAction = "SIGN"
	AuditActionVerify     AuditAction = "VERIFY"
	AuditActionCustody    AuditAction = "CUSTODY"
)

type AuditLog struct {
	LogID      string          `json:"log_id"`
	CaseID     *string         `json:"case_id,omitempty"`
	UserID     string          `json:"user_id"`
	Action     AuditAction     `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Details    json.RawMessage `json:"details,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}
