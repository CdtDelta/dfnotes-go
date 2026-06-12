package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

type CaseFactService struct {
	repo      models.CaseFactRepository
	auditRepo models.AuditLogRepository
	session   *Session
}

func NewCaseFactService(
	repo models.CaseFactRepository,
	auditRepo models.AuditLogRepository,
	session *Session,
) *CaseFactService {
	return &CaseFactService{
		repo:      repo,
		auditRepo: auditRepo,
		session:   session,
	}
}

func (s *CaseFactService) CreateFact(ctx context.Context, userID string, req models.CreateCaseFactRequest) (models.CaseFact, error) {
	if !s.session.IsAuthenticated() {
		return models.CaseFact{}, errors.New("not authenticated")
	}
	if req.CaseID == "" {
		return models.CaseFact{}, errors.New("case_id is required")
	}
	if req.Type == "" {
		return models.CaseFact{}, errors.New("type is required")
	}
	if req.Label == "" {
		return models.CaseFact{}, errors.New("label is required")
	}
	if req.Value == "" {
		return models.CaseFact{}, errors.New("value is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	fact := models.CaseFact{
		FactID:         uuid.New().String(),
		CaseID:         req.CaseID,
		EvidenceItemID: req.EvidenceItemID,
		Type:           req.Type,
		Label:          req.Label,
		Value:          req.Value,
		SourceIOCID:    req.SourceIOCID,
		SourceBlockID:  req.SourceBlockID,
		Notes:          req.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
		UserID:         userID,
	}

	if err := s.repo.Create(ctx, fact); err != nil {
		return models.CaseFact{}, err
	}

	caseID := req.CaseID
	details, _ := json.Marshal(map[string]string{"fact_id": fact.FactID, "type": fact.Type, "label": fact.Label})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &caseID,
		UserID:     userID,
		Action:     models.AuditActionCreate,
		EntityType: "case_fact",
		EntityID:   fact.FactID,
		Details:    details,
		CreatedAt:  time.Now().UTC(),
	})

	return fact, nil
}

func (s *CaseFactService) GetFacts(ctx context.Context, caseID string) ([]models.CaseFact, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	return s.repo.GetByCase(ctx, caseID)
}

func (s *CaseFactService) UpdateFact(ctx context.Context, factID string, req models.UpdateCaseFactRequest) (models.CaseFact, error) {
	if !s.session.IsAuthenticated() {
		return models.CaseFact{}, errors.New("not authenticated")
	}
	if req.Type == "" {
		return models.CaseFact{}, errors.New("type is required")
	}
	if req.Label == "" {
		return models.CaseFact{}, errors.New("label is required")
	}
	if req.Value == "" {
		return models.CaseFact{}, errors.New("value is required")
	}

	fact, err := s.repo.GetByID(ctx, factID)
	if err != nil {
		return models.CaseFact{}, err
	}

	fact.EvidenceItemID = req.EvidenceItemID
	fact.Type = req.Type
	fact.Label = req.Label
	fact.Value = req.Value
	fact.SourceBlockID = req.SourceBlockID
	fact.Notes = req.Notes
	fact.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.repo.Update(ctx, fact); err != nil {
		return models.CaseFact{}, err
	}

	caseID := fact.CaseID
	userID := s.session.User().UserID
	details, _ := json.Marshal(map[string]string{"fact_id": factID, "type": fact.Type})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &caseID,
		UserID:     userID,
		Action:     models.AuditActionUpdate,
		EntityType: "case_fact",
		EntityID:   factID,
		Details:    details,
		CreatedAt:  time.Now().UTC(),
	})

	return fact, nil
}

func (s *CaseFactService) DeleteFact(ctx context.Context, factID string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	fact, err := s.repo.GetByID(ctx, factID)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, factID); err != nil {
		return err
	}

	caseID := fact.CaseID
	userID := s.session.User().UserID
	details, _ := json.Marshal(map[string]string{"fact_id": factID, "type": fact.Type, "label": fact.Label})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &caseID,
		UserID:     userID,
		Action:     models.AuditActionDelete,
		EntityType: "case_fact",
		EntityID:   factID,
		Details:    details,
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}
