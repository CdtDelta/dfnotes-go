package ioc

import "context"

type IOCType string

const (
	IOCTypeIPv4        IOCType = "ipv4"
	IOCTypeIPv6        IOCType = "ipv6"
	IOCTypeDomain      IOCType = "domain"
	IOCTypeURL         IOCType = "url"
	IOCTypeEmail       IOCType = "email"
	IOCTypeMD5         IOCType = "md5"
	IOCTypeSHA1        IOCType = "sha1"
	IOCTypeSHA256      IOCType = "sha256"
	IOCTypeFilePath    IOCType = "file_path"
	IOCTypeRegistryKey IOCType = "registry_key"
	IOCTypeCVE         IOCType = "cve"
)

type IOCStatus string

const (
	IOCStatusDetected      IOCStatus = "detected"
	IOCStatusConfirmed     IOCStatus = "confirmed"
	IOCStatusFalsePositive IOCStatus = "false_positive"
)

const (
	DetectionMethodAuto   = "auto"
	DetectionMethodManual = "manual"
)

type IOCMatch struct {
	Type  IOCType
	Value string
}

type IOCEntry struct {
	IOCID           string    `json:"ioc_id"`
	CaseID          string    `json:"case_id"`
	BlockID         string    `json:"block_id"`
	EvidenceItemID  *string   `json:"evidence_item_id,omitempty"`
	Type            IOCType   `json:"type"`
	Value           string    `json:"value"`
	Status          IOCStatus `json:"status"`
	DetectionMethod string    `json:"detection_method"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedAt       string    `json:"created_at"`
	ConfirmedAt     *string   `json:"confirmed_at,omitempty"`
	UserID          string    `json:"user_id"`
}

// IOCRepository is defined here so the ioc package owns the interface;
// database.IOCRepo implements it without creating a circular import.
type IOCRepository interface {
	Create(ctx context.Context, entry *IOCEntry) error
	GetByID(ctx context.Context, iocID string) (*IOCEntry, error)
	GetByBlock(ctx context.Context, blockID string) ([]IOCEntry, error)
	ListByCase(ctx context.Context, caseID string, includeAll bool) ([]IOCEntry, error)
	UpdateStatus(ctx context.Context, iocID string, status IOCStatus, confirmedAt *string) error
	// GetExistingByBlock returns a set of "type:value" keys already stored for blockID.
	// Used by DetectAndStore to skip duplicates in one round-trip instead of N.
	GetExistingByBlock(ctx context.Context, blockID string) (map[string]struct{}, error)
}
