package ioc

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
	IOCTypeFile        IOCType = "file"
	IOCTypeRegistryKey IOCType = "registry_key"
	IOCTypeCVE         IOCType = "cve"
)

type IOCStatus string

const (
	IOCStatusDetected      IOCStatus = "detected"
	IOCStatusConfirmed     IOCStatus = "confirmed"
	IOCStatusFalsePositive IOCStatus = "false_positive"
	IOCStatusPromoted      IOCStatus = "promoted"
)

const (
	DetectionMethodAuto   = "auto"
	DetectionMethodManual = "manual"
)

type IOCMatch struct {
	Type  IOCType
	Value string
}

// IOCEntry is the flat per-occurrence row the 7z and PDF exports still read,
// built by IOCService.ListExportRows.
//
// TODO(v0.12.0 Pass 6): remove with the export compatibility helper
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
