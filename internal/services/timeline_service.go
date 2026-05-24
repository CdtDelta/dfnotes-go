package services

import (
	"context"
	"errors"
	"time"

	"dfnotes-go/internal/models"
	"dfnotes-go/internal/timer"

	"github.com/google/uuid"
)

type TimelineService struct {
	timelineRepo models.TimelineRepository
	session      *Session
	timerService timer.Service
}

func NewTimelineService(timelineRepo models.TimelineRepository, session *Session, timerService timer.Service) *TimelineService {
	return &TimelineService{
		timelineRepo: timelineRepo,
		session:      session,
		timerService: timerService,
	}
}

func (s *TimelineService) GetTimelineEntries(ctx context.Context, caseID string) ([]models.TimelineEntry, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	return s.timelineRepo.ListByCase(ctx, caseID)
}

func (s *TimelineService) CreateTimelineEntry(ctx context.Context, req models.CreateTimelineEntryRequest) (*models.TimelineEntry, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if req.CaseID == "" {
		return nil, errors.New("case_id is required")
	}
	if req.EventDescription == "" {
		return nil, errors.New("event_description is required")
	}
	if _, err := time.Parse(time.RFC3339, req.Timestamp); err != nil {
		return nil, errors.New("timestamp must be a valid ISO 8601 date-time string")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry := &models.TimelineEntry{
		EntryID:           uuid.New().String(),
		CaseID:            req.CaseID,
		EvidenceItemID:    req.EvidenceItemID,
		Timestamp:         req.Timestamp,
		DisplayTimezone:   req.DisplayTimezone,
		EventDescription:  req.EventDescription,
		InvestigatorNotes: req.InvestigatorNotes,
		CreatedAt:         now,
		UpdatedAt:         now,
		UserID:            s.session.User().UserID,
	}

	if err := s.timelineRepo.Create(ctx, entry); err != nil {
		return nil, err
	}
	s.timerService.ResetFull()
	return entry, nil
}

func (s *TimelineService) UpdateTimelineEntry(ctx context.Context, req models.UpdateTimelineEntryRequest) (*models.TimelineEntry, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if req.EntryID == "" {
		return nil, errors.New("entry_id is required")
	}
	if req.EventDescription == "" {
		return nil, errors.New("event_description is required")
	}
	if _, err := time.Parse(time.RFC3339, req.Timestamp); err != nil {
		return nil, errors.New("timestamp must be a valid ISO 8601 date-time string")
	}

	entry, err := s.timelineRepo.GetByID(ctx, req.EntryID)
	if err != nil {
		return nil, err
	}

	entry.Timestamp = req.Timestamp
	entry.DisplayTimezone = req.DisplayTimezone
	entry.EventDescription = req.EventDescription
	entry.InvestigatorNotes = req.InvestigatorNotes
	entry.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.timelineRepo.Update(ctx, entry); err != nil {
		return nil, err
	}
	s.timerService.ResetFull()
	return entry, nil
}

func (s *TimelineService) DeleteTimelineEntry(ctx context.Context, entryID string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	if entryID == "" {
		return errors.New("entry_id is required")
	}
	return s.timelineRepo.Delete(ctx, entryID)
}
