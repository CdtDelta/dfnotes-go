package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

type AddEvidenceRequest struct {
	CaseID       string `json:"case_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	EvidenceType string `json:"evidence_type"`
	ContentHash  string `json:"content_hash"`
}

type UpdateEvidenceStatusRequest struct {
	EvidenceItemID string `json:"evidence_item_id"`
	Status         string `json:"status"`
}

type AddCustodyEntryRequest struct {
	EvidenceItemID string `json:"evidence_item_id"`
	Action         string `json:"action"`
	Description    string `json:"description"`
}

type CustodyEntryResponse struct {
	Timestamp   string `json:"timestamp"`
	Handler     string `json:"handler"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type EvidenceResponse struct {
	EvidenceItemID string                 `json:"evidence_item_id"`
	CaseID         string                 `json:"case_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	EvidenceType   string                 `json:"evidence_type"`
	Status         string                 `json:"status"`
	ContentHash    string                 `json:"content_hash"`
	CustodyLog     []CustodyEntryResponse `json:"custody_log"`
	CollectedBy    string                 `json:"collected_by"`
	CollectedAt    string                 `json:"collected_at"`
	CreatedAt      string                 `json:"created_at"`
	Tags           []TagResponse          `json:"tags"`
}

type EvidenceService struct {
	evidenceRepo models.EvidenceRepository
	auditRepo    models.AuditLogRepository
	session      *Session
}

func NewEvidenceService(
	evidenceRepo models.EvidenceRepository,
	auditRepo models.AuditLogRepository,
	session *Session,
) *EvidenceService {
	return &EvidenceService{
		evidenceRepo: evidenceRepo,
		auditRepo:    auditRepo,
		session:      session,
	}
}

func (s *EvidenceService) AddEvidence(ctx context.Context, req AddEvidenceRequest) (*EvidenceResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if req.CaseID == "" || req.Name == "" {
		return nil, errors.New("case ID and name are required")
	}

	user := s.session.User()
	now := time.Now().UTC()
	itemID := uuid.New().String()

	item := &models.EvidenceItem{
		EvidenceItemID: itemID,
		CaseID:         req.CaseID,
		Name:           req.Name,
		Description:    req.Description,
		EvidenceType:   models.EvidenceType(req.EvidenceType),
		Status:         models.EvidenceStatusCollected,
		ContentHash:    req.ContentHash,
		CustodyLog: []models.CustodyEntry{
			{
				Timestamp:   now,
				Handler:     user.Name,
				Action:      "COLLECTED",
				Description: "Evidence collected",
			},
		},
		CollectedBy: user.UserID,
		CollectedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.evidenceRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	details, _ := json.Marshal(map[string]string{
		"action":      "add_evidence",
		"evidence_id": itemID,
		"case_id":     req.CaseID,
		"name":        req.Name,
	})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &req.CaseID,
		UserID:     user.UserID,
		Action:     models.AuditActionCreate,
		EntityType: "evidence_item",
		EntityID:   itemID,
		Details:    details,
		CreatedAt:  now,
	})

	return s.evidenceToResponse(item), nil
}

func (s *EvidenceService) ListEvidence(ctx context.Context, caseID string) ([]EvidenceResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	items, err := s.evidenceRepo.ListByCase(ctx, caseID)
	if err != nil {
		return nil, err
	}

	responses := make([]EvidenceResponse, len(items))
	for i, item := range items {
		responses[i] = *s.evidenceToResponse(&item)
	}
	return responses, nil
}

func (s *EvidenceService) GetEvidence(ctx context.Context, evidenceItemID string) (*EvidenceResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	item, err := s.evidenceRepo.GetByID(ctx, evidenceItemID)
	if err != nil {
		return nil, err
	}

	return s.evidenceToResponse(item), nil
}

func (s *EvidenceService) UpdateEvidenceStatus(ctx context.Context, req UpdateEvidenceStatusRequest) (*EvidenceResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if req.EvidenceItemID == "" || req.Status == "" {
		return nil, errors.New("evidence item ID and status are required")
	}

	item, err := s.evidenceRepo.GetByID(ctx, req.EvidenceItemID)
	if err != nil {
		return nil, err
	}

	if item.Status == models.EvidenceStatusWithdrawn {
		return nil, errors.New("cannot change status of withdrawn evidence")
	}

	user := s.session.User()
	now := time.Now().UTC()
	newStatus := models.EvidenceStatus(req.Status)

	item.Status = newStatus
	item.UpdatedAt = now
	item.CustodyLog = append(item.CustodyLog, models.CustodyEntry{
		Timestamp:   now,
		Handler:     user.Name,
		Action:      "STATUS_CHANGE",
		Description: fmt.Sprintf("Status changed to %s", req.Status),
	})

	if err := s.evidenceRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	details, _ := json.Marshal(map[string]string{
		"action":      "update_evidence_status",
		"evidence_id": item.EvidenceItemID,
		"new_status":  req.Status,
	})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &item.CaseID,
		UserID:     user.UserID,
		Action:     models.AuditActionUpdate,
		EntityType: "evidence_item",
		EntityID:   item.EvidenceItemID,
		Details:    details,
		CreatedAt:  now,
	})

	return s.evidenceToResponse(item), nil
}

func (s *EvidenceService) AddCustodyEntry(ctx context.Context, req AddCustodyEntryRequest) (*EvidenceResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if req.EvidenceItemID == "" || req.Action == "" {
		return nil, errors.New("evidence item ID and action are required")
	}

	item, err := s.evidenceRepo.GetByID(ctx, req.EvidenceItemID)
	if err != nil {
		return nil, err
	}

	user := s.session.User()
	now := time.Now().UTC()

	item.UpdatedAt = now
	item.CustodyLog = append(item.CustodyLog, models.CustodyEntry{
		Timestamp:   now,
		Handler:     user.Name,
		Action:      req.Action,
		Description: req.Description,
	})

	if err := s.evidenceRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	details, _ := json.Marshal(map[string]string{
		"action":      "add_custody_entry",
		"evidence_id": item.EvidenceItemID,
		"entry_action": req.Action,
	})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &item.CaseID,
		UserID:     user.UserID,
		Action:     models.AuditActionCustody,
		EntityType: "evidence_item",
		EntityID:   item.EvidenceItemID,
		Details:    details,
		CreatedAt:  now,
	})

	return s.evidenceToResponse(item), nil
}

func (s *EvidenceService) evidenceToResponse(item *models.EvidenceItem) *EvidenceResponse {
	// Resolve collected_by user ID to display name
	collectedBy := item.CollectedBy
	if user := s.session.User(); user != nil && user.UserID == item.CollectedBy {
		collectedBy = user.Name
	}
	custodyLog := make([]CustodyEntryResponse, len(item.CustodyLog))
	for i, entry := range item.CustodyLog {
		custodyLog[i] = CustodyEntryResponse{
			Timestamp:   entry.Timestamp.UTC().Format(time.RFC3339),
			Handler:     entry.Handler,
			Action:      entry.Action,
			Description: entry.Description,
		}
	}

	tags := make([]TagResponse, len(item.Tags))
	for i, t := range item.Tags {
		tags[i] = TagResponse{TagID: t.TagID, Name: t.Name, Color: t.Color}
	}

	return &EvidenceResponse{
		EvidenceItemID: item.EvidenceItemID,
		CaseID:         item.CaseID,
		Name:           item.Name,
		Description:    item.Description,
		EvidenceType:   string(item.EvidenceType),
		Status:         string(item.Status),
		ContentHash:    item.ContentHash,
		CustodyLog:     custodyLog,
		CollectedBy:    collectedBy,
		CollectedAt:    item.CollectedAt.UTC().Format(time.RFC3339),
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		Tags:           tags,
	}
}
