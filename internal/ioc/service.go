package ioc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

// CaseFactCreator is satisfied by *services.CaseFactService. Defined here to
// avoid importing the services package (which would create a circular dependency).
type CaseFactCreator interface {
	CreateFact(ctx context.Context, userID string, req models.CreateCaseFactRequest) (models.CaseFact, error)
}

// AuditWriter is satisfied by *database.AuditRepo.
type AuditWriter interface {
	Create(ctx context.Context, entry *models.AuditLog) error
}

type IOCService struct {
	repo            IOCRepository
	caseFactCreator CaseFactCreator
	auditWriter     AuditWriter
}

func NewIOCService(repo IOCRepository) *IOCService {
	return &IOCService{repo: repo}
}

// WithCaseFactSupport wires in the dependencies needed for PromoteToFact.
func (s *IOCService) WithCaseFactSupport(creator CaseFactCreator, audit AuditWriter) *IOCService {
	s.caseFactCreator = creator
	s.auditWriter = audit
	return s
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

// UpdateIOCType changes the type of an IOC and resets its status to detected.
func (s *IOCService) UpdateIOCType(ctx context.Context, iocID, iocType string) error {
	switch IOCType(iocType) {
	case IOCTypeIPv4, IOCTypeIPv6, IOCTypeDomain, IOCTypeURL, IOCTypeEmail,
		IOCTypeMD5, IOCTypeSHA1, IOCTypeSHA256, IOCTypeFilePath, IOCTypeFile, IOCTypeRegistryKey, IOCTypeCVE:
	default:
		return errors.New("invalid type")
	}
	return s.repo.UpdateType(ctx, iocID, IOCType(iocType))
}

// GetBlockIOCs returns all IOC entries for a specific committed block.
func (s *IOCService) GetBlockIOCs(ctx context.Context, blockID string) ([]IOCEntry, error) {
	return s.repo.GetByBlock(ctx, blockID)
}

// PromoteToFact moves an IOC into Case Facts: creates a case fact record, sets
// the IOC status to promoted, and audit-logs the status change.
func (s *IOCService) PromoteToFact(ctx context.Context, userID string, iocID string, req models.CreateCaseFactRequest) (models.CaseFact, error) {
	if s.caseFactCreator == nil {
		return models.CaseFact{}, errors.New("case fact support not initialized")
	}

	entry, err := s.repo.GetByID(ctx, iocID)
	if err != nil {
		return models.CaseFact{}, fmt.Errorf("get ioc: %w", err)
	}

	// Always set SourceIOCID from the ioc being promoted.
	req.SourceIOCID = &iocID
	req.CaseID = entry.CaseID

	fact, err := s.caseFactCreator.CreateFact(ctx, userID, req)
	if err != nil {
		return models.CaseFact{}, fmt.Errorf("create case fact: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, iocID, IOCStatusPromoted, nil); err != nil {
		return models.CaseFact{}, fmt.Errorf("update ioc status: %w", err)
	}

	if s.auditWriter != nil {
		caseID := entry.CaseID
		details, _ := json.Marshal(map[string]string{
			"ioc_id":  iocID,
			"status":  "promoted",
			"fact_id": fact.FactID,
		})
		s.auditWriter.Create(ctx, &models.AuditLog{
			LogID:      uuid.New().String(),
			CaseID:     &caseID,
			UserID:     userID,
			Action:     models.AuditActionUpdate,
			EntityType: "ioc_entry",
			EntityID:   iocID,
			Details:    details,
			CreatedAt:  time.Now().UTC(),
		})
	}

	return fact, nil
}
