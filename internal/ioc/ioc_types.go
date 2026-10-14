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
