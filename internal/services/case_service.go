package services

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

var evidencePrefixRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type CreateCaseRequest struct {
	CaseNumber                string                    `json:"case_number"`
	Title                     string                    `json:"title"`
	Classification            models.ClassificationLevel `json:"classification"`
	TicketNumber              string                    `json:"ticket_number"`
	Description               string                    `json:"description"`
	CasePassword              string                    `json:"case_password"`
	EvidencePrefix            string                    `json:"evidence_prefix"`
	EvidenceSeqDigits         int                       `json:"evidence_seq_digits"`
	AttorneyClientPrivilege   bool                      `json:"attorney_client_privilege"`
}

type CaseResponse struct {
	CaseID                    string `json:"case_id"`
	CaseNumber                string `json:"case_number"`
	Title                     string `json:"title"`
	Description               string `json:"description"`
	Classification            string `json:"classification"`
	TicketNumber              string `json:"ticket_number"`
	ExaminerName              string `json:"examiner_name"`
	Organization              string `json:"organization"`
	EvidencePrefix            string `json:"evidence_prefix"`
	EvidenceSeqDigits         int    `json:"evidence_seq_digits"`
	CreatedBy                 string `json:"created_by"`
	CreatedAt                 string `json:"created_at"`
	UpdatedAt                 string `json:"updated_at"`
	AttorneyClientPrivilege   bool   `json:"attorney_client_privilege"`
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

	if req.EvidencePrefix == "" {
		req.EvidencePrefix = "E"
	}
	if !evidencePrefixRegex.MatchString(req.EvidencePrefix) {
		return nil, errors.New("evidence prefix must contain only letters, digits, hyphens, and underscores")
	}
	if req.EvidenceSeqDigits < 1 {
		req.EvidenceSeqDigits = 1
	} else if req.EvidenceSeqDigits > 6 {
		req.EvidenceSeqDigits = 6
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
		CaseID:                  caseID,
		CaseNumber:              req.CaseNumber,
		Title:                   req.Title,
		Description:             req.Description,
		Classification:          req.Classification,
		TicketNumber:            req.TicketNumber,
		ExaminerName:            user.Name,
		Organization:            user.Organization,
		EvidencePrefix:          req.EvidencePrefix,
		EvidenceSeqDigits:       req.EvidenceSeqDigits,
		Salt:                    salt,
		EncryptedKey:            encryptedKey,
		CreatedBy:               user.UserID,
		CreatedAt:               now,
		UpdatedAt:               now,
		AttorneyClientPrivilege: req.AttorneyClientPrivilege,
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

// UpdateCaseClassification changes the classification level of a case.
func (s *CaseService) UpdateCaseClassification(ctx context.Context, caseID, newLevel string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	switch models.ClassificationLevel(newLevel) {
	case models.ClassificationUnclassified, models.ClassificationConfidential,
		models.ClassificationRestricted, models.ClassificationSecret, models.ClassificationTopSecret:
	default:
		return errors.New("invalid classification level")
	}

	c, err := s.caseRepo.GetByID(ctx, caseID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	c.Classification = models.ClassificationLevel(newLevel)
	c.UpdatedAt = now

	if err := s.caseRepo.Update(ctx, c); err != nil {
		return err
	}

	user := s.session.User()
	details, _ := json.Marshal(map[string]string{"field": "classification_level", "new_value": newLevel})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &caseID,
		UserID:     user.UserID,
		Action:     models.AuditActionUpdate,
		EntityType: "case",
		EntityID:   caseID,
		Details:    details,
		CreatedAt:  now,
	})

	return nil
}

func (s *CaseService) ToggleAttorneyClientPrivilege(ctx context.Context, caseID string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	c, err := s.caseRepo.GetByID(ctx, caseID)
	if err != nil {
		return err
	}

	newValue := !c.AttorneyClientPrivilege
	now := time.Now().UTC()

	if err := s.caseRepo.UpdateAttorneyClientPrivilege(ctx, caseID, newValue, now.Format(time.RFC3339)); err != nil {
		return err
	}

	newValueStr := "false"
	if newValue {
		newValueStr = "true"
	}
	user := s.session.User()
	details, _ := json.Marshal(map[string]string{"field": "attorney_client_privilege", "new_value": newValueStr})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &caseID,
		UserID:     user.UserID,
		Action:     models.AuditActionUpdate,
		EntityType: "case",
		EntityID:   caseID,
		Details:    details,
		CreatedAt:  now,
	})

	return nil
}

func caseToResponse(c *models.Case) *CaseResponse {
	return &CaseResponse{
		CaseID:                  c.CaseID,
		CaseNumber:              c.CaseNumber,
		Title:                   c.Title,
		Description:             c.Description,
		Classification:          string(c.Classification),
		TicketNumber:            c.TicketNumber,
		ExaminerName:            c.ExaminerName,
		Organization:            c.Organization,
		EvidencePrefix:          c.EvidencePrefix,
		EvidenceSeqDigits:       c.EvidenceSeqDigits,
		CreatedBy:               c.CreatedBy,
		CreatedAt:               c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:               c.UpdatedAt.UTC().Format(time.RFC3339),
		AttorneyClientPrivilege: c.AttorneyClientPrivilege,
	}
}
