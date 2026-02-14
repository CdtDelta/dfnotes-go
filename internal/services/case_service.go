package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

type CreateCaseRequest struct {
	CaseNumber     string                    `json:"case_number"`
	Title          string                    `json:"title"`
	Classification models.ClassificationLevel `json:"classification"`
	TicketNumber   string                    `json:"ticket_number"`
	Description    string                    `json:"description"`
	CasePassword   string                    `json:"case_password"`
}

type CaseResponse struct {
	CaseID         string `json:"case_id"`
	CaseNumber     string `json:"case_number"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Classification string `json:"classification"`
	TicketNumber   string `json:"ticket_number"`
	ExaminerName   string `json:"examiner_name"`
	Organization   string `json:"organization"`
	CreatedBy      string `json:"created_by"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type CaseService struct {
	caseRepo  models.CaseRepository
	auditRepo models.AuditLogRepository
	session   *Session
}

func NewCaseService(caseRepo models.CaseRepository, auditRepo models.AuditLogRepository, session *Session) *CaseService {
	return &CaseService{
		caseRepo:  caseRepo,
		auditRepo: auditRepo,
		session:   session,
	}
}

func (s *CaseService) CreateCase(ctx context.Context, req CreateCaseRequest) (*CaseResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if req.CaseNumber == "" || req.Title == "" || req.CasePassword == "" {
		return nil, errors.New("case number, title, and case password are required")
	}

	if req.Classification == "" {
		req.Classification = models.ClassificationUnclassified
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, err
	}

	caseKey := crypto.DeriveKey(req.CasePassword, salt)

	// Encrypt the case key with the user's master derived key
	encryptedKey, err := crypto.Encrypt(s.session.DerivedKey(), caseKey)
	if err != nil {
		return nil, err
	}

	user := s.session.User()
	now := time.Now().UTC()
	caseID := uuid.New().String()

	c := &models.Case{
		CaseID:         caseID,
		CaseNumber:     req.CaseNumber,
		Title:          req.Title,
		Description:    req.Description,
		Classification: req.Classification,
		TicketNumber:   req.TicketNumber,
		ExaminerName:   user.Name,
		Organization:   user.Organization,
		Salt:           salt,
		EncryptedKey:   encryptedKey,
		CreatedBy:      user.UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.caseRepo.Create(ctx, c); err != nil {
		return nil, err
	}

	details, _ := json.Marshal(map[string]string{"action": "create_case", "case_id": caseID, "case_number": req.CaseNumber})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &caseID,
		UserID:     user.UserID,
		Action:     models.AuditActionCreate,
		EntityType: "case",
		EntityID:   caseID,
		Details:    details,
		CreatedAt:  now,
	})

	return caseToResponse(c), nil
}

func (s *CaseService) ListCases(ctx context.Context) ([]CaseResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	cases, err := s.caseRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]CaseResponse, len(cases))
	for i, c := range cases {
		responses[i] = *caseToResponse(&c)
	}
	return responses, nil
}

func (s *CaseService) GetCase(ctx context.Context, caseID string) (*CaseResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	c, err := s.caseRepo.GetByID(ctx, caseID)
	if err != nil {
		return nil, err
	}

	return caseToResponse(c), nil
}

func caseToResponse(c *models.Case) *CaseResponse {
	return &CaseResponse{
		CaseID:         c.CaseID,
		CaseNumber:     c.CaseNumber,
		Title:          c.Title,
		Description:    c.Description,
		Classification: string(c.Classification),
		TicketNumber:   c.TicketNumber,
		ExaminerName:   c.ExaminerName,
		Organization:   c.Organization,
		CreatedBy:      c.CreatedBy,
		CreatedAt:      c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
