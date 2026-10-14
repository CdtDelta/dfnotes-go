package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
)

// legacyIOCRow is one ioc_entries row as it exists before migration 016.
type legacyIOCRow struct {
	iocID           string
	caseID          string
	blockID         string
	iocType         string
	value           string
	status          string
	detectionMethod string
	notes           string
	createdAt       string
	confirmedAt     sql.NullString
	userID          string
}

// indicatorGroup collects the legacy rows that collapse into one indicator.
// rows keeps the read order, so rows[0] is the earliest.
type indicatorGroup struct {
	caseID      string
	iocType     string
	matchKey    string
	indicatorID string
	rows        []legacyIOCRow
}

// statusRank orders statuses so the strongest analyst decision in a group
// wins: promoted > confirmed > false_positive > detected.
var statusRank = map[string]int{
	string(ioc.IOCStatusDetected):      0,
	string(ioc.IOCStatusFalsePositive): 1,
	string(ioc.IOCStatusConfirmed):     2,
	string(ioc.IOCStatusPromoted):      3,
}

// migrate016IOCIndicators moves per-occurrence ioc_entries rows into
// case-wide indicators and ioc_occurrences, repoints case_facts at
// indicators, and drops ioc_entries. Foreign keys stay on throughout; the
// step order guarantees nothing references a table when it is dropped.
func migrate016IOCIndicators(tx *sql.Tx) error {
	rows, err := readLegacyIOCRows(tx)
	if err != nil {
		return err
	}

	groups := groupLegacyRows(rows)
	now := FormatTime(time.Now())

	oldToNew, err := insertIndicators(tx, groups, now)
	if err != nil {
		return err
	}
	if err := writeMergeAudits(tx, groups, now); err != nil {
		return err
	}
	if err := rebuildCaseFacts(tx, oldToNew); err != nil {
		return err
	}
	if _, err := tx.Exec("DROP TABLE ioc_entries"); err != nil {
		return fmt.Errorf("drop ioc_entries: %w", err)
	}
	return checkForeignKeys(tx)
}

func readLegacyIOCRows(tx *sql.Tx) ([]legacyIOCRow, error) {
	rs, err := tx.Query(`SELECT ioc_id, case_id, block_id, type, value, status,
		detection_method, COALESCE(notes, ''), created_at, confirmed_at, user_id
		FROM ioc_entries ORDER BY created_at, ioc_id`)
	if err != nil {
		return nil, fmt.Errorf("read ioc_entries: %w", err)
	}
	defer rs.Close()

	var out []legacyIOCRow
	for rs.Next() {
		var r legacyIOCRow
		if err := rs.Scan(&r.iocID, &r.caseID, &r.blockID, &r.iocType, &r.value, &r.status,
			&r.detectionMethod, &r.notes, &r.createdAt, &r.confirmedAt, &r.userID); err != nil {
			return nil, fmt.Errorf("scan ioc_entries: %w", err)
		}
		out = append(out, r)
	}
	if err := rs.Err(); err != nil {
		return nil, fmt.Errorf("iterate ioc_entries: %w", err)
	}
	return out, nil
}

// groupLegacyRows groups rows by (case_id, type, match key), keeping groups in
// the order their first row appears so the output is deterministic.
func groupLegacyRows(rows []legacyIOCRow) []*indicatorGroup {
	type key struct{ caseID, iocType, matchKey string }
	index := make(map[key]*indicatorGroup)
	var groups []*indicatorGroup

	for _, r := range rows {
		k := key{r.caseID, r.iocType, ioc.MatchKey(ioc.IOCType(r.iocType), r.value)}
		g, ok := index[k]
		if !ok {
			g = &indicatorGroup{
				caseID:      k.caseID,
				iocType:     k.iocType,
				matchKey:    k.matchKey,
				indicatorID: uuid.New().String(),
			}
			index[k] = g
			groups = append(groups, g)
		}
		g.rows = append(g.rows, r)
	}
	return groups
}

// resolveStatus returns the winning status and, when it is confirmed or
// promoted, the earliest non-null confirmed_at among all rows in the group.
// Promotion implies the indicator was confirmed first, so its confirmation
// time is kept rather than lost.
func resolveStatus(rows []legacyIOCRow) (string, sql.NullString) {
	status := rows[0].status
	for _, r := range rows[1:] {
		if statusRank[r.status] > statusRank[status] {
			status = r.status
		}
	}

	var confirmedAt sql.NullString
	if status != string(ioc.IOCStatusConfirmed) && status != string(ioc.IOCStatusPromoted) {
		return status, confirmedAt
	}
	for _, r := range rows {
		if !r.confirmedAt.Valid {
			continue
		}
		if !confirmedAt.Valid || r.confirmedAt.String < confirmedAt.String {
			confirmedAt = r.confirmedAt
		}
	}
	return status, confirmedAt
}

// insertIndicators writes one indicator per group and one occurrence per
// legacy row. Returns a map from old ioc_id to new indicator_id.
func insertIndicators(tx *sql.Tx, groups []*indicatorGroup, now string) (map[string]string, error) {
	indStmt, err := tx.Prepare(`INSERT INTO indicators (indicator_id, case_id, type, value,
		match_key, match_key_version, status, created_at, updated_at, confirmed_at, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("prepare indicator insert: %w", err)
	}
	defer indStmt.Close()

	occStmt, err := tx.Prepare(`INSERT INTO ioc_occurrences (occurrence_id, indicator_id,
		block_id, raw_value, detection_method, created_at, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("prepare occurrence insert: %w", err)
	}
	defer occStmt.Close()

	oldToNew := make(map[string]string)
	for _, g := range groups {
		first := g.rows[0]
		status, confirmedAt := resolveStatus(g.rows)
		if _, err := indStmt.Exec(g.indicatorID, g.caseID, g.iocType, first.value,
			g.matchKey, ioc.MatchKeyVersion, status, first.createdAt, now, confirmedAt, first.userID); err != nil {
			return nil, fmt.Errorf("insert indicator for %s %q: %w", g.iocType, g.matchKey, err)
		}

		for _, r := range g.rows {
			// Reusing the old ioc_id keeps audit entries and old references traceable.
			if _, err := occStmt.Exec(r.iocID, g.indicatorID, r.blockID, r.value,
				r.detectionMethod, r.createdAt, r.userID); err != nil {
				return nil, fmt.Errorf("insert occurrence %s: %w", r.iocID, err)
			}
			oldToNew[r.iocID] = g.indicatorID
		}
	}
	return oldToNew, nil
}

type mergeSourceRow struct {
	IOCID       string `json:"ioc_id"`
	BlockID     string `json:"block_id"`
	Status      string `json:"status"`
	Value       string `json:"value"`
	Notes       string `json:"notes,omitempty"`
	ConfirmedAt string `json:"confirmed_at,omitempty"`
}

type mergeAuditDetails struct {
	Migration       int              `json:"migration"`
	Reason          string           `json:"reason"`
	Type            string           `json:"type"`
	MatchKey        string           `json:"match_key"`
	ResultingStatus string           `json:"resulting_status"`
	SourceRows      []mergeSourceRow `json:"source_rows"`
}

// writeMergeAudits records every group whose migration loses information at
// the row level: merged rows (individual statuses) or notes (no notes column
// on indicators). Plain INSERTs on tx, because AuditRepo uses the pool and
// would block on the single connection this transaction holds.
func writeMergeAudits(tx *sql.Tx, groups []*indicatorGroup, now string) error {
	stmt, err := tx.Prepare(`INSERT INTO audit_logs (log_id, case_id, user_id, action,
		entity_type, entity_id, details, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare audit insert: %w", err)
	}
	defer stmt.Close()

	for _, g := range groups {
		hasNotes := false
		sources := make([]mergeSourceRow, 0, len(g.rows))
		for _, r := range g.rows {
			if r.notes != "" {
				hasNotes = true
			}
			sources = append(sources, mergeSourceRow{
				IOCID: r.iocID, BlockID: r.blockID, Status: r.status, Value: r.value, Notes: r.notes,
				ConfirmedAt: r.confirmedAt.String, // empty when NULL, so omitted
			})
		}
		if len(g.rows) < 2 && !hasNotes {
			continue
		}

		status, _ := resolveStatus(g.rows)
		details, err := json.Marshal(mergeAuditDetails{
			Migration:       16,
			Reason:          "ioc_indicator_merge",
			Type:            g.iocType,
			MatchKey:        g.matchKey,
			ResultingStatus: status,
			SourceRows:      sources,
		})
		if err != nil {
			return fmt.Errorf("marshal audit details for indicator %s: %w", g.indicatorID, err)
		}

		if _, err := stmt.Exec(uuid.New().String(), g.caseID, g.rows[0].userID,
			string(models.AuditActionUpdate), "indicator", g.indicatorID, string(details), now); err != nil {
			return fmt.Errorf("insert audit for indicator %s: %w", g.indicatorID, err)
		}
	}
	return nil
}

// caseFactRow mirrors the case_facts columns from migration 012.
type caseFactRow struct {
	factID         string
	caseID         string
	evidenceItemID sql.NullString
	factType       string
	label          string
	value          string
	sourceIOCID    sql.NullString
	sourceBlockID  sql.NullString
	notes          string
	createdAt      string
	updatedAt      string
	userID         string
}

// rebuildCaseFacts recreates case_facts so source_ioc_id references
// indicators. SQLite cannot alter a foreign key in place.
func rebuildCaseFacts(tx *sql.Tx, oldToNew map[string]string) error {
	facts, err := readCaseFacts(tx)
	if err != nil {
		return err
	}

	// Same as 012 except the source_ioc_id reference.
	if _, err := tx.Exec(`CREATE TABLE case_facts_new (
    fact_id          TEXT PRIMARY KEY,
    case_id          TEXT NOT NULL REFERENCES cases(case_id),
    evidence_item_id TEXT REFERENCES evidence_items(evidence_item_id),
    type             TEXT NOT NULL,
    label            TEXT NOT NULL,
    value            TEXT NOT NULL,
    source_ioc_id    TEXT REFERENCES indicators(indicator_id),
    source_block_id  TEXT REFERENCES note_blocks(block_id),
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    user_id          TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("create case_facts_new: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO case_facts_new (fact_id, case_id, evidence_item_id,
		type, label, value, source_ioc_id, source_block_id, notes, created_at, updated_at, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare case_facts_new insert: %w", err)
	}
	defer stmt.Close()

	for _, f := range facts {
		source := sql.NullString{}
		if f.sourceIOCID.Valid {
			id, ok := oldToNew[f.sourceIOCID.String]
			if !ok {
				return fmt.Errorf("case fact %s references unknown ioc_id %s", f.factID, f.sourceIOCID.String)
			}
			source = sql.NullString{String: id, Valid: true}
		}
		if _, err := stmt.Exec(f.factID, f.caseID, f.evidenceItemID, f.factType, f.label, f.value,
			source, f.sourceBlockID, f.notes, f.createdAt, f.updatedAt, f.userID); err != nil {
			return fmt.Errorf("copy case fact %s: %w", f.factID, err)
		}
	}

	for _, q := range []string{
		"DROP TABLE case_facts",
		"ALTER TABLE case_facts_new RENAME TO case_facts",
		"CREATE INDEX idx_case_facts_case_id ON case_facts(case_id)",
		"CREATE INDEX idx_case_facts_source_ioc_id ON case_facts(source_ioc_id)",
	} {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("rebuild case_facts (%s): %w", q, err)
		}
	}
	return nil
}

func readCaseFacts(tx *sql.Tx) ([]caseFactRow, error) {
	rs, err := tx.Query(`SELECT fact_id, case_id, evidence_item_id, type, label, value,
		source_ioc_id, source_block_id, notes, created_at, updated_at, user_id FROM case_facts`)
	if err != nil {
		return nil, fmt.Errorf("read case_facts: %w", err)
	}
	defer rs.Close()

	var out []caseFactRow
	for rs.Next() {
		var f caseFactRow
		if err := rs.Scan(&f.factID, &f.caseID, &f.evidenceItemID, &f.factType, &f.label, &f.value,
			&f.sourceIOCID, &f.sourceBlockID, &f.notes, &f.createdAt, &f.updatedAt, &f.userID); err != nil {
			return nil, fmt.Errorf("scan case_facts: %w", err)
		}
		out = append(out, f)
	}
	if err := rs.Err(); err != nil {
		return nil, fmt.Errorf("iterate case_facts: %w", err)
	}
	return out, nil
}

// checkForeignKeys fails the migration if any row violates a foreign key.
func checkForeignKeys(tx *sql.Tx) error {
	rs, err := tx.Query("PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("foreign_key_check: %w", err)
	}
	defer rs.Close()

	var violations []string
	for rs.Next() {
		var table, parent string
		var rowid sql.NullInt64
		var fkid int
		if err := rs.Scan(&table, &rowid, &parent, &fkid); err != nil {
			return fmt.Errorf("scan foreign_key_check: %w", err)
		}
		violations = append(violations, fmt.Sprintf("%s rowid %d -> %s (fk %d)", table, rowid.Int64, parent, fkid))
	}
	if err := rs.Err(); err != nil {
		return fmt.Errorf("iterate foreign_key_check: %w", err)
	}
	if len(violations) > 0 {
		return fmt.Errorf("foreign key violations after migration 016: %s", strings.Join(violations, "; "))
	}
	return nil
}
