package models

import "time"

type IOCType string

const (
	IOCTypeIPv4       IOCType = "IPv4"
	IOCTypeIPv6       IOCType = "IPv6"
	IOCTypeDomain     IOCType = "DOMAIN"
	IOCTypeURL        IOCType = "URL"
	IOCTypeHash       IOCType = "HASH"
	IOCTypeEmail      IOCType = "EMAIL"
	IOCTypeFilename   IOCType = "FILENAME"
	IOCTypeRegistryKey IOCType = "REGISTRY_KEY"
	IOCTypeMutex      IOCType = "MUTEX"
	IOCTypeOther      IOCType = "OTHER"
)

type DetectionMethod string

const (
	DetectionManual    DetectionMethod = "MANUAL"
	DetectionAutomatic DetectionMethod = "AUTOMATIC"
	DetectionImported  DetectionMethod = "IMPORTED"
)

type IOCEntry struct {
	IOCID           string          `json:"ioc_id"`
	CaseID          string          `json:"case_id"`
	IOCType         IOCType         `json:"ioc_type"`
	Value           string          `json:"value"`
	Description     string          `json:"description"`
	DetectionMethod DetectionMethod `json:"detection_method"`
	SourceBlockID   *string         `json:"source_block_id,omitempty"`
	CreatedBy       string          `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
}
