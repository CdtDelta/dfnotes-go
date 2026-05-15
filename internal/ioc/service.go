package ioc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type IOCService struct {
	repo IOCRepository
}

func NewIOCService(repo IOCRepository) *IOCService {
	return &IOCService{repo: repo}
}

// DetectAndStore scans plaintext content and persists any discovered IOCs.
// Called after a note block is committed. This is best-effort: callers must
// not propagate returned errors as commit failures.
func (s *IOCService) DetectAndStore(
	ctx context.Context,
	caseID, blockID string,
	evidenceItemID *string,
	content, userID string,
) error {
	matches := DetectIOCs(content)
	if len(matches) == 0 {
		return nil
	}

	existing, err := s.repo.GetExistingByBlock(ctx, blockID)
	if err != nil {
		return fmt.Errorf("fetch existing iocs block=%s: %w", blockID, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var firstErr error

	for _, m := range matches {
		if _, dup := existing[string(m.Type)+":"+m.Value]; dup {
			continue
		}

		entry := &IOCEntry{
			IOCID:           uuid.New().String(),
			CaseID:          caseID,
			BlockID:         blockID,
			EvidenceItemID:  evidenceItemID,
			Type:            m.Type,
			Value:           m.Value,
			Status:          IOCStatusDetected,
			DetectionMethod: DetectionMethodAuto,
			CreatedAt:       now,
			UserID:          userID,
		}
		if err := s.repo.Create(ctx, entry); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("create ioc block=%s value=%s: %w", blockID, m.Value, err)
			}
		}
	}

	return firstErr
}

// GetCaseIOCs returns IOC entries for a case.
// When includeAll is false, false_positive records are excluded.
func (s *IOCService) GetCaseIOCs(ctx context.Context, caseID string, includeAll bool) ([]IOCEntry, error) {
	return s.repo.ListByCase(ctx, caseID, includeAll)
}

// UpdateIOCStatus changes the status of an IOC and manages confirmed_at.
func (s *IOCService) UpdateIOCStatus(ctx context.Context, iocID, status string) error {
	switch IOCStatus(status) {
	case IOCStatusDetected, IOCStatusConfirmed, IOCStatusFalsePositive:
	default:
		return errors.New("invalid status: must be detected, confirmed, or false_positive")
	}

	var confirmedAt *string
	if IOCStatus(status) == IOCStatusConfirmed {
		t := time.Now().UTC().Format(time.RFC3339)
		confirmedAt = &t
	}

	return s.repo.UpdateStatus(ctx, iocID, IOCStatus(status), confirmedAt)
}

// GetBlockIOCs returns all IOC entries for a specific committed block.
func (s *IOCService) GetBlockIOCs(ctx context.Context, blockID string) ([]IOCEntry, error) {
	return s.repo.GetByBlock(ctx, blockID)
}
