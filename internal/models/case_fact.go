package models

type CaseFact struct {
	FactID         string  `json:"factId"`
	CaseID         string  `json:"caseId"`
	EvidenceItemID *string `json:"evidenceItemId"`
	Type           string  `json:"type"`
	Label          string  `json:"label"`
	Value          string  `json:"value"`
	SourceIOCID    *string `json:"sourceIocId"`
	SourceBlockID  *string `json:"sourceBlockId"`
	Notes          string  `json:"notes"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
	UserID         string  `json:"userId"`
}

type CreateCaseFactRequest struct {
	CaseID         string  `json:"caseId"`
	EvidenceItemID *string `json:"evidenceItemId"`
	Type           string  `json:"type"`
	Label          string  `json:"label"`
	Value          string  `json:"value"`
	SourceIOCID    *string `json:"sourceIocId"`
	SourceBlockID  *string `json:"sourceBlockId"`
	Notes          string  `json:"notes"`
}

type UpdateCaseFactRequest struct {
	EvidenceItemID *string `json:"evidenceItemId"`
	Type           string  `json:"type"`
	Label          string  `json:"label"`
	Value          string  `json:"value"`
	SourceBlockID  *string `json:"sourceBlockId"`
	Notes          string  `json:"notes"`
}

// PredefinedFactTypes is the canonical list of case fact types.
var PredefinedFactTypes = []string{
	"username",
	"hostname",
	"ip_address",
	"mac_address",
	"os_version",
	"timezone",
	"email_address",
	"account_sid",
	"full_name",
	"phone_number",
	"device_serial",
	"url",
	"file_path",
	"domain",
	"registry_key",
	"custom",
}
