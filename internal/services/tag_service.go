package services

import (
	"context"
	"errors"
	"strings"

	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

type TagResponse struct {
	TagID string `json:"tag_id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TagBlockRequest struct {
	BlockID string `json:"block_id"`
	TagID   string `json:"tag_id"`
}

type TagEvidenceRequest struct {
	EvidenceItemID string `json:"evidence_item_id"`
	TagID          string `json:"tag_id"`
}

type TagService struct {
	tagRepo models.TagRepository
	session *Session
}

func NewTagService(tagRepo models.TagRepository, session *Session) *TagService {
	return &TagService{tagRepo: tagRepo, session: session}
}

func (s *TagService) ListTags(ctx context.Context) ([]TagResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	tags, err := s.tagRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]TagResponse, len(tags))
	for i, tag := range tags {
		responses[i] = TagResponse{TagID: tag.TagID, Name: tag.Name, Color: tag.Color}
	}
	return responses, nil
}

func (s *TagService) CreateTag(ctx context.Context, req CreateTagRequest) (*TagResponse, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("tag name is required")
	}
	color := strings.TrimSpace(req.Color)
	if color == "" {
		color = "#6B7280"
	}

	tag := &models.Tag{
		TagID: uuid.New().String(),
		Name:  name,
		Color: color,
	}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return &TagResponse{TagID: tag.TagID, Name: tag.Name, Color: tag.Color}, nil
}

func (s *TagService) DeleteTag(ctx context.Context, tagID string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	return s.tagRepo.Delete(ctx, tagID)
}

func (s *TagService) TagBlock(ctx context.Context, req TagBlockRequest) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	if req.BlockID == "" || req.TagID == "" {
		return errors.New("block ID and tag ID are required")
	}
	return s.tagRepo.AttachToBlock(ctx, req.BlockID, req.TagID)
}

func (s *TagService) UntagBlock(ctx context.Context, req TagBlockRequest) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	if req.BlockID == "" || req.TagID == "" {
		return errors.New("block ID and tag ID are required")
	}
	return s.tagRepo.DetachFromBlock(ctx, req.BlockID, req.TagID)
}

func (s *TagService) TagEvidence(ctx context.Context, req TagEvidenceRequest) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	if req.EvidenceItemID == "" || req.TagID == "" {
		return errors.New("evidence item ID and tag ID are required")
	}
	return s.tagRepo.AttachToEvidence(ctx, req.EvidenceItemID, req.TagID)
}

func (s *TagService) UntagEvidence(ctx context.Context, req TagEvidenceRequest) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	if req.EvidenceItemID == "" || req.TagID == "" {
		return errors.New("evidence item ID and tag ID are required")
	}
	return s.tagRepo.DetachFromEvidence(ctx, req.EvidenceItemID, req.TagID)
}
