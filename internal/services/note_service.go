package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

type UnlockCaseRequest struct {
	CaseID       string `json:"case_id"`
	CasePassword string `json:"case_password"`
}

type CommitNoteRequest struct {
	CaseID         string `json:"case_id"`
	Content        string `json:"content"`
	EvidenceItemID string `json:"evidence_item_id"`
}

type NoteBlockResponse struct {
	BlockID     string        `json:"block_id"`
	CaseID      string        `json:"case_id"`
	Content     string        `json:"content"`
	ContentHash string        `json:"content_hash"`
	PrevHash    string        `json:"prev_hash"`
	AuthorID    string        `json:"author_id"`
	CreatedAt   string        `json:"created_at"`
	Verified    bool          `json:"verified"`
	Tags        []TagResponse `json:"tags"`
}

type SaveAttachmentRequest struct {
	CaseID      string `json:"case_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
}

type AttachmentResponse struct {
	AttachmentID string `json:"attachment_id"`
	Filename     string `json:"filename"`
	ContentType  string `json:"content_type"`
	Data         string `json:"data"`
}

type NoteService struct {
	noteBlockRepo  models.NoteBlockRepository
	caseRepo       models.CaseRepository
	auditRepo      models.AuditLogRepository
	attachmentRepo models.AttachmentRepository
	session        *Session
	mu             sync.RWMutex
	caseKeys       map[string][]byte // caseID → decrypted case key
}

func NewNoteService(
	noteBlockRepo models.NoteBlockRepository,
	caseRepo models.CaseRepository,
	auditRepo models.AuditLogRepository,
	attachmentRepo models.AttachmentRepository,
	session *Session,
) *NoteService {
	return &NoteService{
		noteBlockRepo:  noteBlockRepo,
		caseRepo:       caseRepo,
		auditRepo:      auditRepo,
		attachmentRepo: attachmentRepo,
		session:        session,
		caseKeys:       make(map[string][]byte),
	}
}

func (s *NoteService) UnlockCase(ctx context.Context, req UnlockCaseRequest) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	if req.CaseID == "" || req.CasePassword == "" {
		return errors.New("case ID and case password are required")
	}

	c, err := s.caseRepo.GetByID(ctx, req.CaseID)
	if err != nil {
		return err
	}

	// Derive what the case key should be from the entered password
	candidateKey := crypto.DeriveKey(req.CasePassword, c.Salt)

	// Unwrap the real case key stored at creation time (encrypted under user's master key)
	storedKey, err := crypto.Decrypt(s.session.DerivedKey(), c.EncryptedKey)
	if err != nil {
		return errors.New("invalid case password")
	}

	// Compare: if they match, the password is correct
	if !bytes.Equal(candidateKey, storedKey) {
		return errors.New("invalid case password")
	}

	s.mu.Lock()
	s.caseKeys[req.CaseID] = candidateKey
	s.mu.Unlock()

	return nil
}

// HasActiveCases returns true if any case is currently unlocked.
func (s *NoteService) HasActiveCases() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.caseKeys) > 0
}

// FirstActiveCaseID returns the ID of the first unlocked case, or "" if none.
func (s *NoteService) FirstActiveCaseID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id := range s.caseKeys {
		return id
	}
	return ""
}

func (s *NoteService) LockCase(ctx context.Context, caseID string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if key, ok := s.caseKeys[caseID]; ok {
		for i := range key {
			key[i] = 0
		}
		delete(s.caseKeys, caseID)
	}

	return nil
}

func (s *NoteService) getCaseKey(caseID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.caseKeys[caseID]
	if !ok {
		return nil, errors.New("case is locked")
	}
	return key, nil
}

// DecryptBlockContent decrypts a note block's encrypted body using the active case key.
// Returns an error if the case is locked.
func (s *NoteService) DecryptBlockContent(caseID string, encryptedBody []byte) (string, error) {
	key, err := s.getCaseKey(caseID)
	if err != nil {
		return "", err
	}
	plaintext, err := crypto.Decrypt(key, encryptedBody)
	if err != nil {
		return "", fmt.Errorf("decrypt block: %w", err)
	}
	return string(plaintext), nil
}

func (s *NoteService) CommitNote(ctx context.Context, req CommitNoteRequest) (*NoteBlockResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if req.CaseID == "" || req.Content == "" {
		return nil, errors.New("case ID and content are required")
	}

	caseKey, err := s.getCaseKey(req.CaseID)
	if err != nil {
		return nil, err
	}

	// Get previous block hash for chaining
	prevHash := "genesis"
	lastBlock, err := s.noteBlockRepo.GetLastBlock(ctx, req.CaseID)
	if err != nil && err != models.ErrNotFound {
		return nil, err
	}
	if lastBlock != nil {
		prevHash = lastBlock.ContentHash
	}

	// Hash the content
	contentHash := crypto.HashContent([]byte(req.Content))

	// Encrypt the content with the case key
	encryptedBody, err := crypto.Encrypt(caseKey, []byte(req.Content))
	if err != nil {
		return nil, err
	}

	// Sign the content hash with the user's private key
	signature := crypto.Sign(s.session.PrivateKey(), []byte(contentHash))

	user := s.session.User()
	now := time.Now().UTC()
	blockID := uuid.New().String()

	block := &models.NoteBlock{
		BlockID:       blockID,
		CaseID:        req.CaseID,
		ContentHash:   contentHash,
		PrevHash:      prevHash,
		Signature:     signature,
		EncryptedBody: encryptedBody,
		AuthorID:      user.UserID,
		CreatedAt:     now,
	}

	if req.EvidenceItemID != "" {
		block.EvidenceItemID = &req.EvidenceItemID
	}

	if err := s.noteBlockRepo.Create(ctx, block); err != nil {
		return nil, err
	}

	// Audit log
	details, _ := json.Marshal(map[string]string{
		"action":       "commit_note",
		"block_id":     blockID,
		"case_id":      req.CaseID,
		"content_hash": contentHash,
	})
	s.auditRepo.Create(ctx, &models.AuditLog{
		LogID:      uuid.New().String(),
		CaseID:     &req.CaseID,
		UserID:     user.UserID,
		Action:     models.AuditActionSign,
		EntityType: "note_block",
		EntityID:   blockID,
		Details:    details,
		CreatedAt:  now,
	})

	return &NoteBlockResponse{
		BlockID:     blockID,
		CaseID:      req.CaseID,
		Content:     req.Content,
		ContentHash: contentHash,
		PrevHash:    prevHash,
		AuthorID:    user.UserID,
		CreatedAt:   now.Format(time.RFC3339),
		Verified:    true,
		Tags:        []TagResponse{},
	}, nil
}

func (s *NoteService) ListNotes(ctx context.Context, caseID string) ([]NoteBlockResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	caseKey, err := s.getCaseKey(caseID)
	if err != nil {
		return nil, err
	}

	allBlocks, err := s.noteBlockRepo.ListByCase(ctx, caseID)
	if err != nil {
		return nil, err
	}

	// Filter to case-level notes only (exclude evidence-linked blocks)
	var blocks []models.NoteBlock
	for _, block := range allBlocks {
		if block.EvidenceItemID == nil {
			blocks = append(blocks, block)
		}
	}

	user := s.session.User()
	responses := make([]NoteBlockResponse, len(blocks))
	for i, block := range blocks {
		blockTags := tagsToResponse(block.Tags)

		plaintext, err := crypto.Decrypt(caseKey, block.EncryptedBody)
		if err != nil {
			responses[i] = NoteBlockResponse{
				BlockID:     block.BlockID,
				CaseID:      block.CaseID,
				Content:     "[decryption failed]",
				ContentHash: block.ContentHash,
				PrevHash:    block.PrevHash,
				AuthorID:    block.AuthorID,
				CreatedAt:   block.CreatedAt.UTC().Format(time.RFC3339),
				Verified:    false,
				Tags:        blockTags,
			}
			continue
		}

		recomputedHash := crypto.HashContent(plaintext)
		hashValid := recomputedHash == block.ContentHash
		sigValid := crypto.Verify(user.PublicKey, []byte(block.ContentHash), block.Signature)

		responses[i] = NoteBlockResponse{
			BlockID:     block.BlockID,
			CaseID:      block.CaseID,
			Content:     string(plaintext),
			ContentHash: block.ContentHash,
			PrevHash:    block.PrevHash,
			AuthorID:    block.AuthorID,
			CreatedAt:   block.CreatedAt.UTC().Format(time.RFC3339),
			Verified:    hashValid && sigValid,
			Tags:        blockTags,
		}
	}

	return responses, nil
}

func (s *NoteService) ListEvidenceNotes(ctx context.Context, caseID, evidenceItemID string) ([]NoteBlockResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	caseKey, err := s.getCaseKey(caseID)
	if err != nil {
		return nil, err
	}

	blocks, err := s.noteBlockRepo.ListByEvidence(ctx, evidenceItemID)
	if err != nil {
		return nil, err
	}

	user := s.session.User()
	responses := make([]NoteBlockResponse, len(blocks))
	for i, block := range blocks {
		blockTags := tagsToResponse(block.Tags)

		plaintext, err := crypto.Decrypt(caseKey, block.EncryptedBody)
		if err != nil {
			responses[i] = NoteBlockResponse{
				BlockID:     block.BlockID,
				CaseID:      block.CaseID,
				Content:     "[decryption failed]",
				ContentHash: block.ContentHash,
				PrevHash:    block.PrevHash,
				AuthorID:    block.AuthorID,
				CreatedAt:   block.CreatedAt.UTC().Format(time.RFC3339),
				Verified:    false,
				Tags:        blockTags,
			}
			continue
		}

		recomputedHash := crypto.HashContent(plaintext)
		hashValid := recomputedHash == block.ContentHash
		sigValid := crypto.Verify(user.PublicKey, []byte(block.ContentHash), block.Signature)

		responses[i] = NoteBlockResponse{
			BlockID:     block.BlockID,
			CaseID:      block.CaseID,
			Content:     string(plaintext),
			ContentHash: block.ContentHash,
			PrevHash:    block.PrevHash,
			AuthorID:    block.AuthorID,
			CreatedAt:   block.CreatedAt.UTC().Format(time.RFC3339),
			Verified:    hashValid && sigValid,
			Tags:        blockTags,
		}
	}

	return responses, nil
}

func tagsToResponse(tags []models.Tag) []TagResponse {
	result := make([]TagResponse, len(tags))
	for i, t := range tags {
		result[i] = TagResponse{TagID: t.TagID, Name: t.Name, Color: t.Color}
	}
	return result
}

func (s *NoteService) SaveAttachment(ctx context.Context, req SaveAttachmentRequest) (*AttachmentResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if req.CaseID == "" || req.Data == "" {
		return nil, errors.New("case ID and data are required")
	}

	caseKey, err := s.getCaseKey(req.CaseID)
	if err != nil {
		return nil, err
	}

	rawData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		return nil, errors.New("invalid base64 data")
	}

	encrypted, err := crypto.Encrypt(caseKey, rawData)
	if err != nil {
		return nil, err
	}

	attID := uuid.New().String()
	att := &models.Attachment{
		AttachmentID:  attID,
		CaseID:        req.CaseID,
		Filename:      req.Filename,
		ContentType:   req.ContentType,
		EncryptedData: encrypted,
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.attachmentRepo.Create(ctx, att); err != nil {
		return nil, err
	}

	return &AttachmentResponse{
		AttachmentID: attID,
		Filename:     req.Filename,
		ContentType:  req.ContentType,
	}, nil
}

func (s *NoteService) GetAttachment(ctx context.Context, caseID, attachmentID string) (*AttachmentResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	caseKey, err := s.getCaseKey(caseID)
	if err != nil {
		return nil, err
	}

	att, err := s.attachmentRepo.GetByID(ctx, attachmentID)
	if err != nil {
		return nil, err
	}

	plaintext, err := crypto.Decrypt(caseKey, att.EncryptedData)
	if err != nil {
		return nil, errors.New("failed to decrypt attachment")
	}

	return &AttachmentResponse{
		AttachmentID: att.AttachmentID,
		Filename:     att.Filename,
		ContentType:  att.ContentType,
		Data:         base64.StdEncoding.EncodeToString(plaintext),
	}, nil
}

func (s *NoteService) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, key := range s.caseKeys {
		for i := range key {
			key[i] = 0
		}
		delete(s.caseKeys, id)
	}
}
