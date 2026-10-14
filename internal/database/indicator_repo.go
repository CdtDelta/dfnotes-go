package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
)

// IndicatorRepo implements ioc.IndicatorRepository. The pool has a single
// connection, so every write method does all of its reads and writes on its
// own transaction and never calls back into the pool while it is open.
type IndicatorRepo struct {
	db *DB
}

var _ ioc.IndicatorRepository = (*IndicatorRepo)(nil)

func NewIndicatorRepo(db *DB) *IndicatorRepo {
	return &IndicatorRepo{db: db}
}

const indicatorColumns = `indicator_id, case_id, type, value, match_key, match_key_version,
	status, created_at, updated_at, confirmed_at, user_id`

// Occurrences are ordered by note_blocks.rowid, the same chain order
// NoteBlockRepo.ListByCaseChainOrder and the verifier use. The occurrence
// rowid breaks ties within a block by insertion order.
const occurrenceSelect = `SELECT o.occurrence_id, o.indicator_id, o.block_id,
	nb.evidence_item_id, nb.created_at, o.raw_value, o.detection_method,
	o.created_at, o.user_id
	FROM ioc_occurrences o
	JOIN note_blocks nb ON nb.block_id = o.block_id`

const occurrenceOrder = ` ORDER BY nb.rowid ASC, o.rowid ASC`

// withTx runs fn in one transaction, committing only if fn returns nil.
func (r *IndicatorRepo) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin indicator tx: %w", err)
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(err, fmt.Errorf("rollback indicator tx: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit indicator tx: %w", err)
	}
	return nil
}

func (r *IndicatorRepo) UpsertOccurrence(ctx context.Context, in ioc.UpsertOccurrenceInput) (ioc.UpsertResult, error) {
	if !ioc.ValidType(in.Type) {
		return ioc.UpsertResult{}, fmt.Errorf("%w: %q", ioc.ErrInvalidType, in.Type)
	}
	if !ioc.ValidDetectionMethod(in.DetectionMethod) {
		return ioc.UpsertResult{}, fmt.Errorf("%w: %q", ioc.ErrInvalidDetectionMethod, in.DetectionMethod)
	}
	if strings.TrimSpace(in.RawValue) == "" {
		return ioc.UpsertResult{}, errors.New("raw value is required")
	}
	manual := in.DetectionMethod == ioc.DetectionMethodManual

	var res ioc.UpsertResult
	err := r.withTx(ctx, func(tx *sql.Tx) error {
		if err := checkBlockInCase(ctx, tx, in.BlockID, in.CaseID); err != nil {
			return err
		}

		key := ioc.MatchKey(in.Type, in.RawValue)
		now := FormatTime(time.Now())

		ind, err := findIndicatorByKey(ctx, tx, in.CaseID, in.Type, key)
		if err != nil {
			return err
		}

		if ind == nil {
			ind, err = createIndicator(ctx, tx, in, key, now)
			if err != nil {
				return err
			}
			res.Created = true
		} else if manual {
			tracked, err := indicatorInBlock(ctx, tx, ind.IndicatorID, in.BlockID)
			if err != nil {
				return err
			}
			if tracked {
				res = ioc.UpsertResult{IndicatorID: ind.IndicatorID, Status: ind.Status, AlreadyTracked: true}
				return nil
			}
			if ind.Status == ioc.IOCStatusDetected || ind.Status == ioc.IOCStatusFalsePositive {
				if err := confirmByManualMark(ctx, tx, ind, in, now); err != nil {
					return err
				}
				res.StatusChanged = true
			}
		}

		if err := insertOccurrence(ctx, tx, ind.IndicatorID, in, now); err != nil {
			return err
		}
		res.IndicatorID = ind.IndicatorID
		res.Status = ind.Status
		return nil
	})
	if err != nil {
		return ioc.UpsertResult{}, err
	}
	return res, nil
}

// checkBlockInCase rejects an occurrence whose block is missing or belongs to
// another case, so an indicator can never collect occurrences across cases.
func checkBlockInCase(ctx context.Context, ex execer, blockID, caseID string) error {
	var blockCase string
	err := ex.QueryRowContext(ctx, `SELECT case_id FROM note_blocks WHERE block_id = ?`, blockID).Scan(&blockCase)
	if err != nil {
		return fmt.Errorf("look up block %s: %w", blockID, wrapError(err))
	}
	if blockCase != caseID {
		return fmt.Errorf("block %s does not belong to case %s: %w", blockID, caseID, models.ErrIntegrityViolation)
	}
	return nil
}

// createIndicator inserts a new indicator for in. Manual marks start
// confirmed; auto and propagated start detected.
func createIndicator(ctx context.Context, ex execer, in ioc.UpsertOccurrenceInput, key, now string) (*ioc.Indicator, error) {
	ind := &ioc.Indicator{
		IndicatorID:     uuid.New().String(),
		CaseID:          in.CaseID,
		Type:            in.Type,
		Value:           in.RawValue,
		MatchKey:        key,
		MatchKeyVersion: ioc.MatchKeyVersion,
		Status:          ioc.IOCStatusDetected,
		CreatedAt:       now,
		UpdatedAt:       now,
		UserID:          in.UserID,
	}
	manual := in.DetectionMethod == ioc.DetectionMethodManual
	if manual {
		ind.Status = ioc.IOCStatusConfirmed
		confirmed := now
		ind.ConfirmedAt = &confirmed
	}

	_, err := ex.ExecContext(ctx,
		`INSERT INTO indicators (`+indicatorColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ind.IndicatorID, ind.CaseID, string(ind.Type), ind.Value, ind.MatchKey, ind.MatchKeyVersion,
		string(ind.Status), ind.CreatedAt, ind.UpdatedAt, nullString(ind.ConfirmedAt), ind.UserID)
	if err != nil {
		return nil, fmt.Errorf("insert indicator: %w", wrapError(err))
	}

	if manual {
		if err := writeAudit(ctx, ex, ind.CaseID, in.UserID, models.AuditActionCreate, "indicator", ind.IndicatorID,
			map[string]string{
				"type":     string(ind.Type),
				"value":    ind.Value,
				"status":   string(ind.Status),
				"block_id": in.BlockID,
			}); err != nil {
			return nil, err
		}
	}
	return ind, nil
}

// confirmByManualMark raises a detected or false_positive indicator to
// confirmed because the analyst marked it by hand. Updates ind in place.
func confirmByManualMark(ctx context.Context, ex execer, ind *ioc.Indicator, in ioc.UpsertOccurrenceInput, now string) error {
	from := ind.Status
	if _, err := ex.ExecContext(ctx,
		`UPDATE indicators SET status = ?, confirmed_at = ?, updated_at = ? WHERE indicator_id = ?`,
		string(ioc.IOCStatusConfirmed), now, now, ind.IndicatorID); err != nil {
		return fmt.Errorf("confirm indicator %s: %w", ind.IndicatorID, wrapError(err))
	}
	ind.Status = ioc.IOCStatusConfirmed
	confirmed := now
	ind.ConfirmedAt = &confirmed
	ind.UpdatedAt = now

	return writeAudit(ctx, ex, ind.CaseID, in.UserID, models.AuditActionUpdate, "indicator", ind.IndicatorID,
		map[string]string{
			"from":     string(from),
			"to":       string(ioc.IOCStatusConfirmed),
			"block_id": in.BlockID,
			"reason":   "manual_mark",
		})
}

// insertOccurrence adds the occurrence. A repeat of the same (indicator,
// block, raw value) is skipped silently; manual marks never reach here in
// that state because indicatorInBlock already returned AlreadyTracked.
func insertOccurrence(ctx context.Context, ex execer, indicatorID string, in ioc.UpsertOccurrenceInput, now string) error {
	_, err := ex.ExecContext(ctx,
		`INSERT INTO ioc_occurrences (occurrence_id, indicator_id, block_id, raw_value,
			detection_method, created_at, user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(indicator_id, block_id, raw_value) DO NOTHING`,
		uuid.New().String(), indicatorID, in.BlockID, in.RawValue, in.DetectionMethod, now, in.UserID)
	if err != nil {
		return fmt.Errorf("insert occurrence: %w", wrapError(err))
	}
	return nil
}

func indicatorInBlock(ctx context.Context, ex execer, indicatorID, blockID string) (bool, error) {
	var exists bool
	err := ex.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM ioc_occurrences WHERE indicator_id = ? AND block_id = ?)`,
		indicatorID, blockID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check indicator %s in block %s: %w", indicatorID, blockID, wrapError(err))
	}
	return exists, nil
}

// findIndicatorByKey returns the indicator for (case, type, key), or nil if
// there is none.
func findIndicatorByKey(ctx context.Context, ex execer, caseID string, t ioc.IOCType, key string) (*ioc.Indicator, error) {
	row := ex.QueryRowContext(ctx,
		`SELECT `+indicatorColumns+` FROM indicators WHERE case_id = ? AND type = ? AND match_key = ?`,
		caseID, string(t), key)
	ind, err := scanIndicator(row)
	if errors.Is(err, models.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find indicator %s %q: %w", t, key, err)
	}
	return ind, nil
}

// loadIndicator returns the indicator row without occurrences.
func loadIndicator(ctx context.Context, ex execer, indicatorID string) (*ioc.Indicator, error) {
	row := ex.QueryRowContext(ctx,
		`SELECT `+indicatorColumns+` FROM indicators WHERE indicator_id = ?`, indicatorID)
	ind, err := scanIndicator(row)
	if errors.Is(err, models.ErrNotFound) {
		return nil, ioc.ErrIndicatorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load indicator %s: %w", indicatorID, err)
	}
	return ind, nil
}

func scanIndicator(s scanner) (*ioc.Indicator, error) {
	var ind ioc.Indicator
	var t, status string
	var confirmedAt sql.NullString
	err := s.Scan(&ind.IndicatorID, &ind.CaseID, &t, &ind.Value, &ind.MatchKey, &ind.MatchKeyVersion,
		&status, &ind.CreatedAt, &ind.UpdatedAt, &confirmedAt, &ind.UserID)
	if err != nil {
		return nil, wrapError(err)
	}
	ind.Type = ioc.IOCType(t)
	ind.Status = ioc.IOCStatus(status)
	if confirmedAt.Valid {
		ind.ConfirmedAt = &confirmedAt.String
	}
	ind.Occurrences = []ioc.Occurrence{}
	return &ind, nil
}

func scanOccurrences(rows *sql.Rows) ([]ioc.Occurrence, error) {
	out := []ioc.Occurrence{}
	for rows.Next() {
		var o ioc.Occurrence
		var evidenceItemID sql.NullString
		if err := rows.Scan(&o.OccurrenceID, &o.IndicatorID, &o.BlockID, &evidenceItemID, &o.CommittedAt,
			&o.RawValue, &o.DetectionMethod, &o.CreatedAt, &o.UserID); err != nil {
			return nil, wrapError(err)
		}
		if evidenceItemID.Valid {
			o.EvidenceItemID = &evidenceItemID.String
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func countOccurrences(ctx context.Context, ex execer, indicatorID string) (int, error) {
	var n int
	if err := ex.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ioc_occurrences WHERE indicator_id = ?`, indicatorID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count occurrences for %s: %w", indicatorID, wrapError(err))
	}
	return n, nil
}

func (r *IndicatorRepo) GetIndicator(ctx context.Context, indicatorID string) (*ioc.Indicator, error) {
	ind, err := loadIndicator(ctx, r.db, indicatorID)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, occurrenceSelect+` WHERE o.indicator_id = ?`+occurrenceOrder, indicatorID)
	if err != nil {
		return nil, fmt.Errorf("list occurrences for %s: %w", indicatorID, wrapError(err))
	}
	defer rows.Close()
	occ, err := scanOccurrences(rows)
	if err != nil {
		return nil, fmt.Errorf("scan occurrences for %s: %w", indicatorID, err)
	}
	ind.Occurrences = occ
	return ind, nil
}

func (r *IndicatorRepo) ListIndicatorsByCase(ctx context.Context, caseID string, includeFalsePositives bool) ([]ioc.Indicator, error) {
	filter := ""
	if !includeFalsePositives {
		filter = ` AND i.status != 'false_positive'`
	}

	// Indicator rows are read and closed before the occurrence query runs,
	// because the single pooled connection cannot serve two open result sets.
	indicators, err := r.listCaseIndicatorRows(ctx, caseID, filter)
	if err != nil {
		return nil, err
	}
	if len(indicators) == 0 {
		return indicators, nil
	}

	rows, err := r.db.QueryContext(ctx,
		occurrenceSelect+` JOIN indicators i ON i.indicator_id = o.indicator_id
		 WHERE i.case_id = ?`+filter+occurrenceOrder, caseID)
	if err != nil {
		return nil, fmt.Errorf("list case occurrences: %w", wrapError(err))
	}
	defer rows.Close()
	occ, err := scanOccurrences(rows)
	if err != nil {
		return nil, fmt.Errorf("scan case occurrences: %w", err)
	}

	index := make(map[string]int, len(indicators))
	for i := range indicators {
		index[indicators[i].IndicatorID] = i
	}
	for _, o := range occ {
		if i, ok := index[o.IndicatorID]; ok {
			indicators[i].Occurrences = append(indicators[i].Occurrences, o)
		}
	}
	return indicators, nil
}

func (r *IndicatorRepo) listCaseIndicatorRows(ctx context.Context, caseID, filter string) ([]ioc.Indicator, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+indicatorColumns+` FROM indicators i WHERE i.case_id = ?`+filter+`
		 ORDER BY CASE i.status WHEN 'confirmed' THEN 0 WHEN 'detected' THEN 1 ELSE 2 END,
		          i.created_at DESC`, caseID)
	if err != nil {
		return nil, fmt.Errorf("list case indicators: %w", wrapError(err))
	}
	defer rows.Close()

	out := []ioc.Indicator{}
	for rows.Next() {
		ind, err := scanIndicator(rows)
		if err != nil {
			return nil, fmt.Errorf("scan case indicator: %w", err)
		}
		out = append(out, *ind)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate case indicators: %w", err)
	}
	return out, nil
}

// ListManualIndicators is read by the commit-time scan, which only needs the
// indicator rows, so occurrences are left empty.
func (r *IndicatorRepo) ListManualIndicators(ctx context.Context, caseID string) ([]ioc.Indicator, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+indicatorColumns+` FROM indicators i
		 WHERE i.case_id = ?
		   AND EXISTS (
		       SELECT 1 FROM ioc_occurrences o
		       WHERE o.indicator_id = i.indicator_id AND o.detection_method = ?)
		 ORDER BY i.created_at ASC, i.rowid ASC`, caseID, ioc.DetectionMethodManual)
	if err != nil {
		return nil, fmt.Errorf("list manual indicators: %w", wrapError(err))
	}
	defer rows.Close()

	out := []ioc.Indicator{}
	for rows.Next() {
		ind, err := scanIndicator(rows)
		if err != nil {
			return nil, fmt.Errorf("scan manual indicator: %w", err)
		}
		out = append(out, *ind)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manual indicators: %w", err)
	}
	return out, nil
}

func (r *IndicatorRepo) ListBlockIOCs(ctx context.Context, blockID string) ([]ioc.BlockIOC, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT o.occurrence_id, o.indicator_id, o.block_id, i.type, o.raw_value, i.status, o.detection_method
		 FROM ioc_occurrences o
		 JOIN indicators i ON i.indicator_id = o.indicator_id
		 WHERE o.block_id = ?
		 ORDER BY o.rowid ASC`, blockID)
	if err != nil {
		return nil, fmt.Errorf("list block iocs: %w", wrapError(err))
	}
	defer rows.Close()

	out := []ioc.BlockIOC{}
	for rows.Next() {
		var b ioc.BlockIOC
		var t, status string
		if err := rows.Scan(&b.OccurrenceID, &b.IndicatorID, &b.BlockID, &t, &b.RawValue, &status,
			&b.DetectionMethod); err != nil {
			return nil, fmt.Errorf("scan block ioc: %w", wrapError(err))
		}
		b.Type = ioc.IOCType(t)
		b.Status = ioc.IOCStatus(status)
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate block iocs: %w", err)
	}
	return out, nil
}

func (r *IndicatorRepo) ListLinkedFacts(ctx context.Context, indicatorID string) ([]models.CaseFact, error) {
	return listLinkedFacts(ctx, r.db, indicatorID)
}

func listLinkedFacts(ctx context.Context, ex execer, indicatorID string) ([]models.CaseFact, error) {
	rows, err := ex.QueryContext(ctx,
		`SELECT fact_id, case_id, evidence_item_id, type, label, value,
		        source_ioc_id, source_block_id, notes, created_at, updated_at, user_id
		 FROM case_facts WHERE source_ioc_id = ? ORDER BY created_at ASC, fact_id ASC`, indicatorID)
	if err != nil {
		return nil, fmt.Errorf("list linked facts for %s: %w", indicatorID, wrapError(err))
	}
	defer rows.Close()

	facts, err := scanCaseFacts(rows)
	if err != nil {
		return nil, fmt.Errorf("scan linked facts for %s: %w", indicatorID, err)
	}
	return facts, nil
}

func (r *IndicatorRepo) UpdateIndicatorStatus(ctx context.Context, indicatorID string, status ioc.IOCStatus, userID string) error {
	switch status {
	case ioc.IOCStatusDetected, ioc.IOCStatusConfirmed, ioc.IOCStatusFalsePositive:
	default:
		return fmt.Errorf("%w: cannot set status %q", ioc.ErrInvalidStatusTransition, status)
	}

	return r.withTx(ctx, func(tx *sql.Tx) error {
		ind, err := loadIndicator(ctx, tx, indicatorID)
		if err != nil {
			return err
		}
		if ind.Status == ioc.IOCStatusPromoted {
			return ioc.ErrPromotedLocked
		}
		if ind.Status == status {
			return nil
		}

		now := FormatTime(time.Now())
		var confirmedAt any
		if status == ioc.IOCStatusConfirmed {
			confirmedAt = now
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE indicators SET status = ?, confirmed_at = ?, updated_at = ? WHERE indicator_id = ?`,
			string(status), confirmedAt, now, indicatorID); err != nil {
			return fmt.Errorf("update indicator status %s: %w", indicatorID, wrapError(err))
		}

		count, err := countOccurrences(ctx, tx, indicatorID)
		if err != nil {
			return err
		}
		return writeAudit(ctx, tx, ind.CaseID, userID, models.AuditActionUpdate, "indicator", indicatorID,
			map[string]any{
				"from":             string(ind.Status),
				"to":               string(status),
				"occurrence_count": count,
			})
	})
}

// statusPrecedence returns whichever status carries the stronger analyst
// decision, using the same ranking migration 016 applied.
func statusPrecedence(a, b ioc.IOCStatus) ioc.IOCStatus {
	if statusRank[string(b)] > statusRank[string(a)] {
		return b
	}
	return a
}

func (r *IndicatorRepo) UpdateIndicatorType(ctx context.Context, indicatorID string, newType ioc.IOCType, confirmMerge bool, userID string) (ioc.TypeChangeResult, error) {
	if !ioc.ValidType(newType) {
		return ioc.TypeChangeResult{}, fmt.Errorf("%w: %q", ioc.ErrInvalidType, newType)
	}

	var res ioc.TypeChangeResult
	err := r.withTx(ctx, func(tx *sql.Tx) error {
		src, err := loadIndicator(ctx, tx, indicatorID)
		if err != nil {
			return err
		}
		if src.Status == ioc.IOCStatusPromoted {
			return ioc.ErrPromotedLocked
		}
		if src.Type == newType {
			return nil
		}

		// The key comes from the indicator's display value. Occurrences stay
		// with the indicator even if their raw values would normalize
		// differently under the new type.
		newKey := ioc.MatchKey(newType, src.Value)
		target, err := findIndicatorByKey(ctx, tx, src.CaseID, newType, newKey)
		if err != nil {
			return err
		}

		if target == nil {
			if err := retypeIndicator(ctx, tx, src, newType, newKey, userID); err != nil {
				return err
			}
			res.Changed = true
			return nil
		}

		if !confirmMerge {
			count, err := countOccurrences(ctx, tx, src.IndicatorID)
			if err != nil {
				return err
			}
			res.Collision = &ioc.TypeCollision{
				TargetIndicatorID: target.IndicatorID,
				TargetStatus:      target.Status,
				OccurrencesToMove: count,
			}
			return nil
		}

		if err := mergeIndicator(ctx, tx, src, target, userID); err != nil {
			return err
		}
		res.Changed = true
		res.Merged = true
		return nil
	})
	if err != nil {
		return ioc.TypeChangeResult{}, err
	}
	return res, nil
}

// retypeIndicator changes type and key in place and resets the analyst
// decision, since a confirmation under the old type no longer applies.
func retypeIndicator(ctx context.Context, ex execer, src *ioc.Indicator, newType ioc.IOCType, newKey, userID string) error {
	now := FormatTime(time.Now())
	if _, err := ex.ExecContext(ctx,
		`UPDATE indicators SET type = ?, match_key = ?, match_key_version = ?, status = ?,
		     confirmed_at = NULL, updated_at = ?
		 WHERE indicator_id = ?`,
		string(newType), newKey, ioc.MatchKeyVersion, string(ioc.IOCStatusDetected), now, src.IndicatorID); err != nil {
		return fmt.Errorf("retype indicator %s: %w", src.IndicatorID, wrapError(err))
	}
	return writeAudit(ctx, ex, src.CaseID, userID, models.AuditActionUpdate, "indicator", src.IndicatorID,
		map[string]string{
			"from_type":   string(src.Type),
			"to_type":     string(newType),
			"from_status": string(src.Status),
		})
}

type mergedFromDetails struct {
	IndicatorID string `json:"indicator_id"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Status      string `json:"status"`
}

type mergeDetails struct {
	MergedFrom        mergedFromDetails `json:"merged_from"`
	OccurrencesMoved  int64             `json:"occurrences_moved"`
	DuplicatesDropped int64             `json:"duplicates_dropped"`
	FactsRepointed    int64             `json:"facts_repointed"`
}

// mergeIndicator folds src into target: moves occurrences (dropping any the
// target already has for the same block and raw value), repoints case facts,
// applies status precedence to the target and deletes src.
func mergeIndicator(ctx context.Context, ex execer, src, target *ioc.Indicator, userID string) error {
	moved, err := execCount(ctx, ex,
		`UPDATE ioc_occurrences SET indicator_id = ?
		 WHERE indicator_id = ?
		   AND NOT EXISTS (
		       SELECT 1 FROM ioc_occurrences t
		       WHERE t.indicator_id = ?
		         AND t.block_id = ioc_occurrences.block_id
		         AND t.raw_value = ioc_occurrences.raw_value)`,
		target.IndicatorID, src.IndicatorID, target.IndicatorID)
	if err != nil {
		return fmt.Errorf("move occurrences %s -> %s: %w", src.IndicatorID, target.IndicatorID, err)
	}

	// Whatever is left on src duplicates a (block, raw value) the target has.
	dropped, err := execCount(ctx, ex,
		`DELETE FROM ioc_occurrences WHERE indicator_id = ?`, src.IndicatorID)
	if err != nil {
		return fmt.Errorf("drop duplicate occurrences of %s: %w", src.IndicatorID, err)
	}

	repointed, err := execCount(ctx, ex,
		`UPDATE case_facts SET source_ioc_id = ? WHERE source_ioc_id = ?`,
		target.IndicatorID, src.IndicatorID)
	if err != nil {
		return fmt.Errorf("repoint facts %s -> %s: %w", src.IndicatorID, target.IndicatorID, err)
	}

	// The moved occurrences arrive as a fresh retype, so they contribute
	// detected. confirmed_at on the target is left as it was.
	status := statusPrecedence(target.Status, ioc.IOCStatusDetected)
	if _, err := ex.ExecContext(ctx,
		`UPDATE indicators SET status = ?, updated_at = ? WHERE indicator_id = ?`,
		string(status), FormatTime(time.Now()), target.IndicatorID); err != nil {
		return fmt.Errorf("update merge target %s: %w", target.IndicatorID, wrapError(err))
	}

	if _, err := ex.ExecContext(ctx,
		`DELETE FROM indicators WHERE indicator_id = ?`, src.IndicatorID); err != nil {
		return fmt.Errorf("delete merged indicator %s: %w", src.IndicatorID, wrapError(err))
	}

	return writeAudit(ctx, ex, target.CaseID, userID, models.AuditActionUpdate, "indicator", target.IndicatorID,
		mergeDetails{
			MergedFrom: mergedFromDetails{
				IndicatorID: src.IndicatorID,
				Type:        string(src.Type),
				Value:       src.Value,
				Status:      string(src.Status),
			},
			OccurrencesMoved:  moved,
			DuplicatesDropped: dropped,
			FactsRepointed:    repointed,
		})
}

// execCount runs a write and returns the number of rows it affected.
func execCount(ctx context.Context, ex execer, query string, args ...any) (int64, error) {
	result, err := ex.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, wrapError(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return n, nil
}

// validateFactRequest mirrors the request checks CaseFactService.CreateFact
// applies, so promoting cannot create a fact the normal path would reject.
// case_id is not checked because promote always takes it from the indicator.
func validateFactRequest(req models.CreateCaseFactRequest) error {
	if req.Type == "" {
		return errors.New("type is required")
	}
	if req.Label == "" {
		return errors.New("label is required")
	}
	if req.Value == "" {
		return errors.New("value is required")
	}
	return nil
}

func (r *IndicatorRepo) PromoteIndicator(ctx context.Context, indicatorID string, req models.CreateCaseFactRequest, userID string) (models.CaseFact, error) {
	if err := validateFactRequest(req); err != nil {
		return models.CaseFact{}, err
	}

	var fact models.CaseFact
	err := r.withTx(ctx, func(tx *sql.Tx) error {
		ind, err := loadIndicator(ctx, tx, indicatorID)
		if err != nil {
			return err
		}
		if ind.Status != ioc.IOCStatusDetected && ind.Status != ioc.IOCStatusConfirmed {
			return fmt.Errorf("%w: cannot promote from %q", ioc.ErrInvalidStatusTransition, ind.Status)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		sourceID := indicatorID
		fact = models.CaseFact{
			FactID:         uuid.New().String(),
			CaseID:         ind.CaseID,
			EvidenceItemID: req.EvidenceItemID,
			Type:           req.Type,
			Label:          req.Label,
			Value:          req.Value,
			SourceIOCID:    &sourceID,
			SourceBlockID:  req.SourceBlockID,
			Notes:          req.Notes,
			CreatedAt:      now,
			UpdatedAt:      now,
			UserID:         userID,
		}
		if err := insertCaseFact(ctx, tx, &fact); err != nil {
			return fmt.Errorf("insert promoted fact: %w", err)
		}

		// confirmed_at is kept: promotion builds on the confirmation.
		if _, err := tx.ExecContext(ctx,
			`UPDATE indicators SET status = ?, updated_at = ? WHERE indicator_id = ?`,
			string(ioc.IOCStatusPromoted), now, indicatorID); err != nil {
			return fmt.Errorf("promote indicator %s: %w", indicatorID, wrapError(err))
		}

		if err := writeAudit(ctx, tx, ind.CaseID, userID, models.AuditActionCreate, "case_fact", fact.FactID,
			caseFactAuditDetails(&fact)); err != nil {
			return err
		}
		return writeAudit(ctx, tx, ind.CaseID, userID, models.AuditActionUpdate, "indicator", indicatorID,
			map[string]string{
				"from":    string(ind.Status),
				"to":      string(ioc.IOCStatusPromoted),
				"fact_id": fact.FactID,
			})
	})
	if err != nil {
		return models.CaseFact{}, err
	}
	return fact, nil
}

func (r *IndicatorRepo) RestoreIndicator(ctx context.Context, indicatorID, userID string) ([]string, error) {
	deleted := []string{}
	err := r.withTx(ctx, func(tx *sql.Tx) error {
		ind, err := loadIndicator(ctx, tx, indicatorID)
		if err != nil {
			return err
		}
		if ind.Status != ioc.IOCStatusPromoted {
			return fmt.Errorf("%w: cannot restore from %q", ioc.ErrInvalidStatusTransition, ind.Status)
		}

		// Zero linked facts is valid: they may already have been deleted
		// from the Case Facts tab.
		facts, err := listLinkedFacts(ctx, tx, indicatorID)
		if err != nil {
			return err
		}
		for i := range facts {
			f := &facts[i]
			if err := deleteCaseFact(ctx, tx, f.FactID); err != nil {
				return fmt.Errorf("delete linked fact %s: %w", f.FactID, err)
			}
			if err := writeAudit(ctx, tx, f.CaseID, userID, models.AuditActionDelete, "case_fact", f.FactID,
				caseFactAuditDetails(f)); err != nil {
				return err
			}
			deleted = append(deleted, f.FactID)
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE indicators SET status = ?, confirmed_at = NULL, updated_at = ? WHERE indicator_id = ?`,
			string(ioc.IOCStatusDetected), FormatTime(time.Now()), indicatorID); err != nil {
			return fmt.Errorf("restore indicator %s: %w", indicatorID, wrapError(err))
		}

		return writeAudit(ctx, tx, ind.CaseID, userID, models.AuditActionUpdate, "indicator", indicatorID,
			map[string]any{
				"from":             string(ioc.IOCStatusPromoted),
				"to":               string(ioc.IOCStatusDetected),
				"deleted_fact_ids": deleted,
			})
	})
	if err != nil {
		return nil, err
	}
	return deleted, nil
}
