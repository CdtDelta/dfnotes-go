package ioc

import (
	"context"
	"errors"
	"fmt"

	"dfnotes-go/internal/models"
)

// DetectionMethodPropagated marks an occurrence added because the value was
// already known in the case, rather than detected or entered by hand.
const DetectionMethodPropagated = "propagated"

var (
	// ErrIndicatorNotFound wraps models.ErrNotFound so callers that only
	// check the generic sentinel still match.
	ErrIndicatorNotFound       = fmt.Errorf("indicator %w", models.ErrNotFound)
	ErrPromotedLocked          = errors.New("indicator is promoted; restore it before changing type or status")
	ErrInvalidStatusTransition = errors.New("invalid indicator status transition")
	ErrInvalidType             = errors.New("invalid ioc type")
	ErrInvalidDetectionMethod  = errors.New("invalid detection method")
)

// ValidType reports whether t is one of the known IOC types.
func ValidType(t IOCType) bool {
	switch t {
	case IOCTypeIPv4, IOCTypeIPv6, IOCTypeDomain, IOCTypeURL, IOCTypeEmail,
		IOCTypeMD5, IOCTypeSHA1, IOCTypeSHA256, IOCTypeFilePath, IOCTypeFile,
		IOCTypeRegistryKey, IOCTypeCVE:
		return true
	}
	return false
}

// ValidDetectionMethod reports whether m is a known detection method.
func ValidDetectionMethod(m string) bool {
	switch m {
	case DetectionMethodAuto, DetectionMethodManual, DetectionMethodPropagated:
		return true
	}
	return false
}

type Indicator struct {
	IndicatorID     string       `json:"indicator_id"`
	CaseID          string       `json:"case_id"`
	Type            IOCType      `json:"type"`
	Value           string       `json:"value"`
	MatchKey        string       `json:"match_key"`
	MatchKeyVersion int          `json:"match_key_version"`
	Status          IOCStatus    `json:"status"`
	CreatedAt       string       `json:"created_at"`
	UpdatedAt       string       `json:"updated_at"`
	ConfirmedAt     *string      `json:"confirmed_at,omitempty"`
	UserID          string       `json:"user_id"`
	Occurrences     []Occurrence `json:"occurrences"`
}

type Occurrence struct {
	OccurrenceID    string  `json:"occurrence_id"`
	IndicatorID     string  `json:"indicator_id"`
	BlockID         string  `json:"block_id"`
	EvidenceItemID  *string `json:"evidence_item_id,omitempty"` // from note_blocks
	CommittedAt     string  `json:"committed_at"`               // from note_blocks
	RawValue        string  `json:"raw_value"`
	DetectionMethod string  `json:"detection_method"`
	CreatedAt       string  `json:"created_at"`
	UserID          string  `json:"user_id"`
}

// BlockIOC is one occurrence in a block, carrying the indicator's current
// type and status.
type BlockIOC struct {
	OccurrenceID    string    `json:"occurrence_id"`
	IndicatorID     string    `json:"indicator_id"`
	BlockID         string    `json:"block_id"`
	Type            IOCType   `json:"type"`
	RawValue        string    `json:"raw_value"`
	Status          IOCStatus `json:"status"`
	DetectionMethod string    `json:"detection_method"`
}

type UpsertOccurrenceInput struct {
	CaseID, BlockID string
	Type            IOCType
	RawValue        string
	DetectionMethod string // "auto", "manual", "propagated"
	UserID          string
}

type UpsertResult struct {
	IndicatorID    string    `json:"indicator_id"`
	Status         IOCStatus `json:"status"`
	Created        bool      `json:"created"`         // a new indicator was created
	AlreadyTracked bool      `json:"already_tracked"` // manual mark on a block that already has this indicator; nothing changed
	StatusChanged  bool      `json:"status_changed"`
}

type TypeCollision struct {
	TargetIndicatorID string    `json:"target_indicator_id"`
	TargetStatus      IOCStatus `json:"target_status"`
	OccurrencesToMove int       `json:"occurrences_to_move"`
}

type TypeChangeResult struct {
	Changed   bool           `json:"changed"`
	Merged    bool           `json:"merged"`
	Collision *TypeCollision `json:"collision,omitempty"` // set when a merge is needed but not confirmed
}

// IndicatorRepository is defined here so the ioc package owns the interface;
// database.IndicatorRepo implements it without creating a circular import.
// Every write method runs in a single transaction, audit entries included.
type IndicatorRepository interface {
	UpsertOccurrence(ctx context.Context, in UpsertOccurrenceInput) (UpsertResult, error)
	GetIndicator(ctx context.Context, indicatorID string) (*Indicator, error)
	ListIndicatorsByCase(ctx context.Context, caseID string, includeFalsePositives bool) ([]Indicator, error)
	ListBlockIOCs(ctx context.Context, blockID string) ([]BlockIOC, error)
	ListLinkedFacts(ctx context.Context, indicatorID string) ([]models.CaseFact, error)
	UpdateIndicatorStatus(ctx context.Context, indicatorID string, status IOCStatus, userID string) error
	UpdateIndicatorType(ctx context.Context, indicatorID string, newType IOCType, confirmMerge bool, userID string) (TypeChangeResult, error)
	PromoteIndicator(ctx context.Context, indicatorID string, req models.CreateCaseFactRequest, userID string) (models.CaseFact, error)
	RestoreIndicator(ctx context.Context, indicatorID, userID string) ([]string, error)
}
