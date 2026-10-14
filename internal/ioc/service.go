package ioc

import (
	"context"
	"errors"
	"fmt"

	"dfnotes-go/internal/models"
)

// Session is satisfied by *services.Session. Defined here so the ioc package
// does not import services.
type Session interface {
	IsAuthenticated() bool
	User() *models.UserIdentity
}

// ErrNotAuthenticated is returned when no user session is active. The
// message matches the one the services package returns.
var ErrNotAuthenticated = errors.New("not authenticated")

// userError carries a plain-English message that the frontend shows as is,
// while keeping the repository sentinel matchable with errors.Is.
type userError struct {
	msg string
	err error
}

func (e *userError) Error() string { return e.msg }
func (e *userError) Unwrap() error { return e.err }

// operation selects the wording for errors whose meaning depends on what
// the user was trying to do.
type operation int

const (
	opRead operation = iota
	opStatus
	opType
	opMark
	opPromote
	opRestore
)

const (
	msgNotFound      = "This indicator no longer exists. Refresh the IOC list and try again."
	msgLockedStatus  = "This indicator is promoted to a case fact. Restore it before changing its status."
	msgLockedType    = "This indicator is promoted to a case fact. Restore it before changing its type."
	msgInvalidType   = "That is not a valid IOC type."
	msgStatusValues  = "Status must be detected, confirmed, or false positive."
	msgPromoteFrom   = "Only a detected or confirmed indicator can be promoted to a case fact."
	msgRestoreFrom   = "Only a promoted indicator can be restored."
	msgInvalidChange = "This status change is not allowed."
)

// friendlyError replaces a repository sentinel with a plain-English message.
// Errors that are not sentinels pass through unchanged.
func friendlyError(err error, op operation) error {
	if err == nil {
		return nil
	}
	var msg string
	switch {
	case errors.Is(err, ErrIndicatorNotFound):
		msg = msgNotFound
	case errors.Is(err, ErrPromotedLocked):
		msg = msgLockedStatus
		if op == opType {
			msg = msgLockedType
		}
	case errors.Is(err, ErrInvalidType):
		msg = msgInvalidType
	case errors.Is(err, ErrInvalidStatusTransition):
		switch op {
		case opStatus:
			msg = msgStatusValues
		case opPromote:
			msg = msgPromoteFrom
		case opRestore:
			msg = msgRestoreFrom
		default:
			msg = msgInvalidChange
		}
	default:
		return err
	}
	return &userError{msg: msg, err: err}
}

type IOCService struct {
	repo    IndicatorRepository
	session Session
}

func NewIOCService(repo IndicatorRepository, session Session) *IOCService {
	return &IOCService{repo: repo, session: session}
}

// currentUserID returns the authenticated user's id, or ErrNotAuthenticated.
// User is checked for nil as well because the session can be cleared between
// the two calls.
func (s *IOCService) currentUserID() (string, error) {
	if !s.session.IsAuthenticated() {
		return "", ErrNotAuthenticated
	}
	u := s.session.User()
	if u == nil {
		return "", ErrNotAuthenticated
	}
	return u.UserID, nil
}

func (s *IOCService) requireAuth() error {
	_, err := s.currentUserID()
	return err
}

// DetectAndStore scans plaintext content and records an auto occurrence for
// every regex match, then a propagated occurrence for every known manual
// indicator found in the content. Called after a note block is committed.
// This is best-effort: callers must not propagate returned errors as commit
// failures. userID is the block author, passed in by the caller.
func (s *IOCService) DetectAndStore(ctx context.Context, caseID, blockID, content, userID string) error {
	if err := s.requireAuth(); err != nil {
		return err
	}

	var firstErr error
	for _, m := range DetectIOCs(content) {
		_, err := s.repo.UpsertOccurrence(ctx, UpsertOccurrenceInput{
			CaseID:          caseID,
			BlockID:         blockID,
			Type:            m.Type,
			RawValue:        m.Value,
			DetectionMethod: DetectionMethodAuto,
			UserID:          userID,
		})
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("store ioc block=%s type=%s: %w", blockID, m.Type, err)
		}
	}

	if err := s.propagateKnown(ctx, caseID, blockID, content, userID); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// propagateKnown records manual indicators the regexes cannot find. It runs
// after the regex pass, so a value already stored as auto in this block hits
// the unique (indicator, block, raw_value) constraint and is skipped.
func (s *IOCService) propagateKnown(ctx context.Context, caseID, blockID, content, userID string) error {
	known, err := s.repo.ListManualIndicators(ctx, caseID)
	if err != nil {
		return fmt.Errorf("list manual indicators case=%s: %w", caseID, err)
	}
	if len(known) == 0 {
		return nil
	}

	var firstErr error
	for _, m := range FindKnownValues(content, known) {
		_, err := s.repo.UpsertOccurrence(ctx, UpsertOccurrenceInput{
			CaseID:          caseID,
			BlockID:         blockID,
			Type:            m.Type,
			RawValue:        m.RawValue,
			DetectionMethod: DetectionMethodPropagated,
			UserID:          userID,
		})
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("store propagated ioc block=%s indicator=%s: %w", blockID, m.IndicatorID, err)
		}
	}
	return firstErr
}

// GetCaseIndicators returns the case's indicators with their occurrences.
// False positives are left out unless includeFalsePositives is set.
func (s *IOCService) GetCaseIndicators(ctx context.Context, caseID string, includeFalsePositives bool) ([]Indicator, error) {
	if err := s.requireAuth(); err != nil {
		return nil, err
	}
	indicators, err := s.repo.ListIndicatorsByCase(ctx, caseID, includeFalsePositives)
	return indicators, friendlyError(err, opRead)
}

// GetBlockIndicators returns every occurrence in a committed block with the
// indicator's current type and status.
func (s *IOCService) GetBlockIndicators(ctx context.Context, blockID string) ([]BlockIOC, error) {
	if err := s.requireAuth(); err != nil {
		return nil, err
	}
	iocs, err := s.repo.ListBlockIOCs(ctx, blockID)
	return iocs, friendlyError(err, opRead)
}

// UpdateIndicatorStatus sets an indicator to detected, confirmed or
// false_positive. Promotion and restore have their own methods.
func (s *IOCService) UpdateIndicatorStatus(ctx context.Context, indicatorID, status string) error {
	userID, err := s.currentUserID()
	if err != nil {
		return err
	}
	st := IOCStatus(status)
	switch st {
	case IOCStatusDetected, IOCStatusConfirmed, IOCStatusFalsePositive:
	default:
		return friendlyError(fmt.Errorf("%w: cannot set status %q", ErrInvalidStatusTransition, status), opStatus)
	}
	return friendlyError(s.repo.UpdateIndicatorStatus(ctx, indicatorID, st, userID), opStatus)
}

// UpdateIndicatorType changes an indicator's type. If another indicator
// already holds the new type and value, the first call reports a Collision
// and changes nothing; calling again with confirmMerge merges the two.
func (s *IOCService) UpdateIndicatorType(ctx context.Context, indicatorID, iocType string, confirmMerge bool) (TypeChangeResult, error) {
	userID, err := s.currentUserID()
	if err != nil {
		return TypeChangeResult{}, err
	}
	t := IOCType(iocType)
	if !ValidType(t) {
		return TypeChangeResult{}, friendlyError(fmt.Errorf("%w: %q", ErrInvalidType, iocType), opType)
	}
	res, err := s.repo.UpdateIndicatorType(ctx, indicatorID, t, confirmMerge, userID)
	if err != nil {
		return TypeChangeResult{}, friendlyError(err, opType)
	}
	return res, nil
}

// MarkManualIOC records a value the analyst selected by hand. A new value
// becomes a confirmed indicator. A value already tracked in this block
// returns AlreadyTracked, which is not an error.
func (s *IOCService) MarkManualIOC(ctx context.Context, caseID, blockID, iocType, value string) (UpsertResult, error) {
	userID, err := s.currentUserID()
	if err != nil {
		return UpsertResult{}, err
	}
	t := IOCType(iocType)
	if !ValidType(t) {
		return UpsertResult{}, friendlyError(fmt.Errorf("%w: %q", ErrInvalidType, iocType), opMark)
	}
	res, err := s.repo.UpsertOccurrence(ctx, UpsertOccurrenceInput{
		CaseID:          caseID,
		BlockID:         blockID,
		Type:            t,
		RawValue:        value,
		DetectionMethod: DetectionMethodManual,
		UserID:          userID,
	})
	if err != nil {
		return UpsertResult{}, friendlyError(err, opMark)
	}
	return res, nil
}

// PromoteIndicator creates a case fact from the indicator and marks it
// promoted, in one transaction.
func (s *IOCService) PromoteIndicator(ctx context.Context, indicatorID string, req models.CreateCaseFactRequest) (models.CaseFact, error) {
	userID, err := s.currentUserID()
	if err != nil {
		return models.CaseFact{}, err
	}
	fact, err := s.repo.PromoteIndicator(ctx, indicatorID, req, userID)
	if err != nil {
		return models.CaseFact{}, friendlyError(err, opPromote)
	}
	return fact, nil
}

// RestoreIndicator deletes the facts linked to a promoted indicator and
// returns it to detected, in one transaction. Returns the deleted fact ids.
func (s *IOCService) RestoreIndicator(ctx context.Context, indicatorID string) ([]string, error) {
	userID, err := s.currentUserID()
	if err != nil {
		return nil, err
	}
	deleted, err := s.repo.RestoreIndicator(ctx, indicatorID, userID)
	if err != nil {
		return nil, friendlyError(err, opRestore)
	}
	return deleted, nil
}

// GetLinkedFacts returns the case facts that reference an indicator, for the
// restore confirmation dialog.
func (s *IOCService) GetLinkedFacts(ctx context.Context, indicatorID string) ([]models.CaseFact, error) {
	if err := s.requireAuth(); err != nil {
		return nil, err
	}
	facts, err := s.repo.ListLinkedFacts(ctx, indicatorID)
	return facts, friendlyError(err, opRead)
}

// ListExportRows flattens every indicator in the case, false positives
// included, into one IOCEntry per occurrence so the existing 7z and PDF
// export code keeps its output shape. Indicators come in repository order
// and occurrences in chain order within each indicator.
//
// TODO(v0.12.0 Pass 6): replace with ioc_summary.json v2 and the grouped PDF table
func (s *IOCService) ListExportRows(ctx context.Context, caseID string) ([]IOCEntry, error) {
	if err := s.requireAuth(); err != nil {
		return nil, err
	}
	indicators, err := s.repo.ListIndicatorsByCase(ctx, caseID, true)
	if err != nil {
		return nil, fmt.Errorf("list indicators for export: %w", err)
	}

	rows := []IOCEntry{}
	for _, ind := range indicators {
		for _, o := range ind.Occurrences {
			rows = append(rows, IOCEntry{
				IOCID:           o.OccurrenceID,
				CaseID:          ind.CaseID,
				BlockID:         o.BlockID,
				EvidenceItemID:  o.EvidenceItemID,
				Type:            ind.Type,
				Value:           o.RawValue,
				Status:          ind.Status,
				DetectionMethod: o.DetectionMethod,
				Notes:           nil,
				CreatedAt:       o.CreatedAt,
				ConfirmedAt:     ind.ConfirmedAt,
				UserID:          o.UserID,
			})
		}
	}
	return rows, nil
}
