package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
)

// newIndicatorFixture returns a migrated database with user-1, case-1 (blocks
// b1 to b4, b1 on evidence ev-1) and case-2 (block c2b1). Block created_at
// runs backwards against insertion order, so a test that sees chain order
// proves the repository orders by rowid and not by timestamp.
func newIndicatorFixture(t *testing.T) (*DB, *IndicatorRepo) {
	t.Helper()
	db := setupTestDB(t)
	user := createTestUser(t, db)
	createTestCase(t, db, user.UserID)
	mustExec(t, db, `INSERT INTO cases (case_id, title, created_by, created_at, updated_at)
		VALUES ('case-2', 'Other', 'user-1', ?, ?)`, ts(0), ts(0))
	mustExec(t, db, `INSERT INTO evidence_items (evidence_item_id, case_id, name, collected_by, collected_at, created_at, updated_at)
		VALUES ('ev-1', 'case-1', 'Laptop', 'user-1', ?, ?, ?)`, ts(0), ts(0), ts(0))
	for _, b := range []struct {
		id, caseID string
		ev         any
		sec        int
	}{
		{"b1", "case-1", "ev-1", 50},
		{"b2", "case-1", nil, 40},
		{"b3", "case-1", nil, 30},
		{"b4", "case-1", nil, 20},
		{"c2b1", "case-2", nil, 10},
	} {
		mustExec(t, db, `INSERT INTO note_blocks (block_id, case_id, evidence_item_id, content_hash, prev_hash, signature, encrypted_body, author_id, created_at)
			VALUES (?, ?, ?, 'h', 'p', x'00', x'00', 'user-1', ?)`, b.id, b.caseID, b.ev, ts(b.sec))
	}
	return db, NewIndicatorRepo(db)
}

func upsert(t *testing.T, repo *IndicatorRepo, block string, typ ioc.IOCType, raw, method string) ioc.UpsertResult {
	t.Helper()
	res, err := repo.UpsertOccurrence(context.Background(), ioc.UpsertOccurrenceInput{
		CaseID: "case-1", BlockID: block, Type: typ, RawValue: raw, DetectionMethod: method, UserID: "user-1",
	})
	if err != nil {
		t.Fatalf("UpsertOccurrence(%s, %s, %q, %s): %v", block, typ, raw, method, err)
	}
	return res
}

func getInd(t *testing.T, repo *IndicatorRepo, id string) *ioc.Indicator {
	t.Helper()
	ind, err := repo.GetIndicator(context.Background(), id)
	if err != nil {
		t.Fatalf("GetIndicator(%s): %v", id, err)
	}
	return ind
}

func setStatus(t *testing.T, repo *IndicatorRepo, id string, status ioc.IOCStatus) {
	t.Helper()
	if err := repo.UpdateIndicatorStatus(context.Background(), id, status, "user-1"); err != nil {
		t.Fatalf("UpdateIndicatorStatus(%s, %s): %v", id, status, err)
	}
}

func promote(t *testing.T, repo *IndicatorRepo, id string) models.CaseFact {
	t.Helper()
	fact, err := repo.PromoteIndicator(context.Background(), id, models.CreateCaseFactRequest{
		Type: "domain", Label: "C2 domain", Value: "evil.com",
	}, "user-1")
	if err != nil {
		t.Fatalf("PromoteIndicator(%s): %v", id, err)
	}
	return fact
}

func addLinkedFact(t *testing.T, db *DB, factID, indicatorID string) {
	t.Helper()
	src := indicatorID
	if err := NewCaseFactRepo(db).Create(context.Background(), models.CaseFact{
		FactID: factID, CaseID: "case-1", Type: "domain", Label: "extra", Value: "evil.com",
		SourceIOCID: &src, CreatedAt: ts(59), UpdatedAt: ts(59), UserID: "user-1",
	}); err != nil {
		t.Fatalf("create fact %s: %v", factID, err)
	}
}

type auditRow struct {
	action, entityType, entityID string
	details                      map[string]any
}

// auditsSince returns audit entries inserted after the first `skip` rows.
func auditsSince(t *testing.T, db *DB, skip int) []auditRow {
	t.Helper()
	rows, err := db.Query(`SELECT action, entity_type, entity_id, details FROM audit_logs ORDER BY rowid LIMIT -1 OFFSET ?`, skip)
	if err != nil {
		t.Fatalf("query audits: %v", err)
	}
	defer rows.Close()
	var out []auditRow
	for rows.Next() {
		var a auditRow
		var details string
		if err := rows.Scan(&a.action, &a.entityType, &a.entityID, &details); err != nil {
			t.Fatalf("scan audit: %v", err)
		}
		if err := json.Unmarshal([]byte(details), &a.details); err != nil {
			t.Fatalf("unmarshal audit details %q: %v", details, err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate audits: %v", err)
	}
	return out
}

func auditCount(t *testing.T, db *DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_logs`).Scan(&n); err != nil {
		t.Fatalf("count audits: %v", err)
	}
	return n
}

func indicatorCount(t *testing.T, db *DB, caseID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM indicators WHERE case_id = ?`, caseID).Scan(&n); err != nil {
		t.Fatalf("count indicators: %v", err)
	}
	return n
}

func occBlocks(ind *ioc.Indicator) []string {
	out := []string{}
	for _, o := range ind.Occurrences {
		out = append(out, o.BlockID)
	}
	return out
}

// dumpTables captures every row of the tables a write method can touch.
func dumpTables(t *testing.T, db *DB) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, table := range []string{"indicators", "ioc_occurrences", "case_facts", "audit_logs"} {
		rows, err := db.Query(`SELECT * FROM ` + table + ` ORDER BY rowid`)
		if err != nil {
			t.Fatalf("dump %s: %v", table, err)
		}
		cols, err := rows.Columns()
		if err != nil {
			t.Fatalf("columns %s: %v", table, err)
		}
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatalf("scan %s: %v", table, err)
			}
			out[table] = append(out[table], fmt.Sprint(vals...))
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate %s: %v", table, err)
		}
		rows.Close()
	}
	return out
}

// failIndicatorAudits makes every indicator audit insert abort. Each write
// method writes its indicator audit last, so this forces a failure after all
// of the method's other writes have run. Lives only in the test database.
func failIndicatorAudits(t *testing.T, db *DB) {
	t.Helper()
	mustExec(t, db, `CREATE TRIGGER test_fail_indicator_audit BEFORE INSERT ON audit_logs
		WHEN NEW.entity_type = 'indicator'
		BEGIN SELECT RAISE(ABORT, 'forced test failure'); END`)
}

func requireForcedFailure(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "forced test failure") {
		t.Fatalf("expected forced trigger failure, got %v", err)
	}
}

func TestUpsertAutoOccurrences(t *testing.T) {
	db, repo := newIndicatorFixture(t)

	first := upsert(t, repo, "b1", ioc.IOCTypeDomain, "Evil.com", ioc.DetectionMethodAuto)
	if !first.Created || first.Status != ioc.IOCStatusDetected || first.StatusChanged || first.AlreadyTracked {
		t.Fatalf("first auto upsert = %+v, want created detected", first)
	}
	ind := getInd(t, repo, first.IndicatorID)
	if ind.Value != "Evil.com" || ind.MatchKey != "evil.com" || ind.MatchKeyVersion != ioc.MatchKeyVersion {
		t.Fatalf("indicator value/key = %q/%q v%d", ind.Value, ind.MatchKey, ind.MatchKeyVersion)
	}
	if ind.ConfirmedAt != nil {
		t.Fatal("auto indicator should have no confirmed_at")
	}
	if n := auditCount(t, db); n != 0 {
		t.Fatalf("auto upsert wrote %d audit entries, want 0", n)
	}

	setStatus(t, repo, first.IndicatorID, ioc.IOCStatusConfirmed)

	second := upsert(t, repo, "b2", ioc.IOCTypeDomain, "EVIL.COM.", ioc.DetectionMethodAuto)
	if second.IndicatorID != first.IndicatorID || second.Created || second.StatusChanged {
		t.Fatalf("case variant upsert = %+v, want join %s", second, first.IndicatorID)
	}
	if second.Status != ioc.IOCStatusConfirmed {
		t.Fatalf("occurrence should inherit confirmed, got %s", second.Status)
	}

	again := upsert(t, repo, "b2", ioc.IOCTypeDomain, "EVIL.COM.", ioc.DetectionMethodPropagated)
	if again.IndicatorID != first.IndicatorID {
		t.Fatalf("repeat upsert joined %s", again.IndicatorID)
	}
	ind = getInd(t, repo, first.IndicatorID)
	if len(ind.Occurrences) != 2 {
		t.Fatalf("expected 2 occurrences after repeat, got %d", len(ind.Occurrences))
	}
	if ind.Occurrences[1].RawValue != "EVIL.COM." || ind.Occurrences[1].DetectionMethod != "auto" {
		t.Fatalf("repeat should keep first occurrence, got %+v", ind.Occurrences[1])
	}
	// Only the status change above is audited.
	if n := auditCount(t, db); n != 1 {
		t.Fatalf("expected 1 audit entry, got %d", n)
	}
}

func TestUpsertManualNewIndicator(t *testing.T) {
	db, repo := newIndicatorFixture(t)

	res := upsert(t, repo, "b1", ioc.IOCTypeIPv4, "10.0.0.5", ioc.DetectionMethodManual)
	if !res.Created || res.Status != ioc.IOCStatusConfirmed {
		t.Fatalf("manual upsert = %+v, want created confirmed", res)
	}
	ind := getInd(t, repo, res.IndicatorID)
	if ind.Status != ioc.IOCStatusConfirmed || ind.ConfirmedAt == nil {
		t.Fatalf("manual indicator status=%s confirmed_at=%v", ind.Status, ind.ConfirmedAt)
	}
	if ind.Occurrences[0].DetectionMethod != "manual" {
		t.Fatalf("occurrence method = %s", ind.Occurrences[0].DetectionMethod)
	}
	audits := auditsSince(t, db, 0)
	if len(audits) != 1 || audits[0].action != "CREATE" || audits[0].entityType != "indicator" ||
		audits[0].entityID != res.IndicatorID {
		t.Fatalf("audits = %+v, want one indicator CREATE", audits)
	}
}

func TestUpsertManualRaisesStatus(t *testing.T) {
	for _, start := range []ioc.IOCStatus{ioc.IOCStatusDetected, ioc.IOCStatusFalsePositive} {
		t.Run(string(start), func(t *testing.T) {
			db, repo := newIndicatorFixture(t)
			auto := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodAuto)
			if start != ioc.IOCStatusDetected {
				setStatus(t, repo, auto.IndicatorID, start)
			}
			before := auditCount(t, db)

			res := upsert(t, repo, "b2", ioc.IOCTypeDomain, "Evil.com", ioc.DetectionMethodManual)
			if res.IndicatorID != auto.IndicatorID || res.Created || !res.StatusChanged ||
				res.Status != ioc.IOCStatusConfirmed {
				t.Fatalf("manual upsert = %+v", res)
			}
			ind := getInd(t, repo, auto.IndicatorID)
			if ind.Status != ioc.IOCStatusConfirmed || ind.ConfirmedAt == nil {
				t.Fatalf("status=%s confirmed_at=%v", ind.Status, ind.ConfirmedAt)
			}
			if len(ind.Occurrences) != 2 {
				t.Fatalf("expected 2 occurrences, got %d", len(ind.Occurrences))
			}
			audits := auditsSince(t, db, before)
			if len(audits) != 1 || audits[0].action != "UPDATE" ||
				audits[0].details["from"] != string(start) || audits[0].details["to"] != "confirmed" {
				t.Fatalf("audits = %+v", audits)
			}
		})
	}
}

func TestUpsertManualOnPromotedLeavesPromoted(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	auto := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodAuto)
	promote(t, repo, auto.IndicatorID)
	before := auditCount(t, db)

	res := upsert(t, repo, "b2", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodManual)
	if res.Status != ioc.IOCStatusPromoted || res.StatusChanged || res.AlreadyTracked {
		t.Fatalf("manual on promoted = %+v", res)
	}
	if got := len(getInd(t, repo, auto.IndicatorID).Occurrences); got != 2 {
		t.Fatalf("expected 2 occurrences, got %d", got)
	}
	if n := auditCount(t, db) - before; n != 0 {
		t.Fatalf("manual on promoted wrote %d audits", n)
	}
}

func TestUpsertManualAlreadyTracked(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	auto := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodAuto)
	before := dumpTables(t, db)

	// A different raw spelling still counts: the indicator is already in b1.
	res := upsert(t, repo, "b1", ioc.IOCTypeDomain, "EVIL.com", ioc.DetectionMethodManual)
	if !res.AlreadyTracked || res.IndicatorID != auto.IndicatorID || res.StatusChanged || res.Created ||
		res.Status != ioc.IOCStatusDetected {
		t.Fatalf("manual on tracked block = %+v", res)
	}
	if after := dumpTables(t, db); !reflect.DeepEqual(before, after) {
		t.Fatalf("AlreadyTracked changed the database:\nbefore %v\nafter  %v", before, after)
	}
}

func TestUpsertFileTypeIsExact(t *testing.T) {
	_, repo := newIndicatorFixture(t)
	a := upsert(t, repo, "b1", ioc.IOCTypeFile, "NTUSER.DAT", ioc.DetectionMethodAuto)
	b := upsert(t, repo, "b2", ioc.IOCTypeFile, "ntuser.dat", ioc.DetectionMethodAuto)
	if a.IndicatorID == b.IndicatorID || !b.Created {
		t.Fatalf("file case variants merged: %+v %+v", a, b)
	}
}

func TestUpsertValidation(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	ctx := context.Background()
	base := ioc.UpsertOccurrenceInput{CaseID: "case-1", BlockID: "b1", Type: ioc.IOCTypeDomain,
		RawValue: "evil.com", DetectionMethod: ioc.DetectionMethodAuto, UserID: "user-1"}

	bad := base
	bad.Type = "hostname"
	if _, err := repo.UpsertOccurrence(ctx, bad); !errors.Is(err, ioc.ErrInvalidType) {
		t.Fatalf("invalid type err = %v", err)
	}
	bad = base
	bad.DetectionMethod = "guessed"
	if _, err := repo.UpsertOccurrence(ctx, bad); !errors.Is(err, ioc.ErrInvalidDetectionMethod) {
		t.Fatalf("invalid method err = %v", err)
	}
	bad = base
	bad.RawValue = "   "
	if _, err := repo.UpsertOccurrence(ctx, bad); err == nil {
		t.Fatal("blank raw value accepted")
	}
	bad = base
	bad.BlockID = "c2b1"
	if _, err := repo.UpsertOccurrence(ctx, bad); !errors.Is(err, models.ErrIntegrityViolation) {
		t.Fatalf("cross-case block err = %v", err)
	}
	bad = base
	bad.BlockID = "missing"
	if _, err := repo.UpsertOccurrence(ctx, bad); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("missing block err = %v", err)
	}
	if n := indicatorCount(t, db, "case-1"); n != 0 {
		t.Fatalf("rejected upserts created %d indicators", n)
	}
}

func TestListIndicatorsByCase(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	ctx := context.Background()

	empty, err := repo.ListIndicatorsByCase(ctx, "case-2", true)
	if err != nil {
		t.Fatalf("ListIndicatorsByCase empty: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty case = %#v, want non-nil empty slice", empty)
	}

	// Occurrences for "older" are inserted b3 first, then b1. Chain order
	// (rowid) is b1 before b3 even though b1 has the later created_at.
	older := upsert(t, repo, "b3", ioc.IOCTypeIPv4, "10.0.0.1", ioc.DetectionMethodAuto).IndicatorID
	upsert(t, repo, "b1", ioc.IOCTypeIPv4, "10.0.0.1", ioc.DetectionMethodAuto)
	confirmed := upsert(t, repo, "b2", ioc.IOCTypeIPv4, "10.0.0.2", ioc.DetectionMethodManual).IndicatorID
	fp := upsert(t, repo, "b2", ioc.IOCTypeIPv4, "10.0.0.3", ioc.DetectionMethodAuto).IndicatorID
	promoted := upsert(t, repo, "b2", ioc.IOCTypeIPv4, "10.0.0.4", ioc.DetectionMethodAuto).IndicatorID
	newer := upsert(t, repo, "b4", ioc.IOCTypeIPv4, "10.0.0.5", ioc.DetectionMethodAuto).IndicatorID
	setStatus(t, repo, fp, ioc.IOCStatusFalsePositive)
	promote(t, repo, promoted)

	// Pin created_at so the DESC ordering does not depend on wall-clock ties.
	for id, sec := range map[string]int{older: 1, confirmed: 2, fp: 3, promoted: 4, newer: 5} {
		mustExec(t, db, `UPDATE indicators SET created_at = ? WHERE indicator_id = ?`, ts(sec), id)
	}

	ids := func(list []ioc.Indicator) []string {
		out := []string{}
		for _, i := range list {
			out = append(out, i.IndicatorID)
		}
		return out
	}

	list, err := repo.ListIndicatorsByCase(ctx, "case-1", false)
	if err != nil {
		t.Fatalf("ListIndicatorsByCase: %v", err)
	}
	if want := []string{confirmed, newer, older, promoted}; !reflect.DeepEqual(ids(list), want) {
		t.Fatalf("order without FP = %v, want %v", ids(list), want)
	}

	all, err := repo.ListIndicatorsByCase(ctx, "case-1", true)
	if err != nil {
		t.Fatalf("ListIndicatorsByCase all: %v", err)
	}
	if want := []string{confirmed, newer, older, promoted, fp}; !reflect.DeepEqual(ids(all), want) {
		t.Fatalf("order with FP = %v, want %v", ids(all), want)
	}

	olderInd := list[2]
	if got := occBlocks(&olderInd); !reflect.DeepEqual(got, []string{"b1", "b3"}) {
		t.Fatalf("occurrences not in chain order: %v", got)
	}
	o1, o3 := olderInd.Occurrences[0], olderInd.Occurrences[1]
	if o1.EvidenceItemID == nil || *o1.EvidenceItemID != "ev-1" || o3.EvidenceItemID != nil {
		t.Fatalf("evidence ids = %v, %v", o1.EvidenceItemID, o3.EvidenceItemID)
	}
	if o1.CommittedAt != ts(50) || o3.CommittedAt != ts(30) {
		t.Fatalf("committed_at = %s, %s", o1.CommittedAt, o3.CommittedAt)
	}
	for _, ind := range all {
		if len(ind.Occurrences) == 0 {
			t.Fatalf("indicator %s has no nested occurrences", ind.IndicatorID)
		}
	}

	// GetIndicator uses the same chain order.
	if got := occBlocks(getInd(t, repo, older)); !reflect.DeepEqual(got, []string{"b1", "b3"}) {
		t.Fatalf("GetIndicator occurrences = %v", got)
	}
}

func TestListBlockIOCsReflectsStatus(t *testing.T) {
	_, repo := newIndicatorFixture(t)
	ctx := context.Background()

	empty, err := repo.ListBlockIOCs(ctx, "b4")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty block = %#v, %v", empty, err)
	}

	id := upsert(t, repo, "b1", ioc.IOCTypeDomain, "Evil.com", ioc.DetectionMethodAuto).IndicatorID
	upsert(t, repo, "b2", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodAuto)
	setStatus(t, repo, id, ioc.IOCStatusFalsePositive)

	got, err := repo.ListBlockIOCs(ctx, "b1")
	if err != nil {
		t.Fatalf("ListBlockIOCs: %v", err)
	}
	if len(got) != 1 || got[0].Status != ioc.IOCStatusFalsePositive || got[0].RawValue != "Evil.com" ||
		got[0].Type != ioc.IOCTypeDomain || got[0].IndicatorID != id || got[0].DetectionMethod != "auto" {
		t.Fatalf("block iocs = %+v", got)
	}
}

func TestListManualIndicators(t *testing.T) {
	_, repo := newIndicatorFixture(t)
	ctx := context.Background()

	empty, err := repo.ListManualIndicators(ctx, "case-1")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("case with no indicators = %#v, %v", empty, err)
	}

	manual := upsert(t, repo, "b1", ioc.IOCTypeFile, "NTUSER.DAT", ioc.DetectionMethodManual).IndicatorID
	fp := upsert(t, repo, "b1", ioc.IOCTypeDomain, "fp.example.com", ioc.DetectionMethodManual).IndicatorID
	setStatus(t, repo, fp, ioc.IOCStatusFalsePositive)
	// Auto first, manual mark later in another block: still counts as manual.
	mixed := upsert(t, repo, "b2", ioc.IOCTypeDomain, "mixed.example.com", ioc.DetectionMethodAuto).IndicatorID
	upsert(t, repo, "b3", ioc.IOCTypeDomain, "mixed.example.com", ioc.DetectionMethodManual)
	upsert(t, repo, "b2", ioc.IOCTypeIPv4, "10.0.0.1", ioc.DetectionMethodAuto)
	upsert(t, repo, "b2", ioc.IOCTypeFile, "other.dat", ioc.DetectionMethodPropagated)
	if _, err := repo.UpsertOccurrence(ctx, ioc.UpsertOccurrenceInput{
		CaseID: "case-2", BlockID: "c2b1", Type: ioc.IOCTypeFile, RawValue: "NTUSER.DAT",
		DetectionMethod: ioc.DetectionMethodManual, UserID: "user-1",
	}); err != nil {
		t.Fatalf("manual mark in case-2: %v", err)
	}

	got, err := repo.ListManualIndicators(ctx, "case-1")
	if err != nil {
		t.Fatalf("ListManualIndicators: %v", err)
	}
	ids := map[string]ioc.Indicator{}
	for _, ind := range got {
		ids[ind.IndicatorID] = ind
		if ind.CaseID != "case-1" {
			t.Errorf("indicator %s from %s, want case-1", ind.IndicatorID, ind.CaseID)
		}
		if len(ind.Occurrences) != 0 {
			t.Errorf("indicator %s has %d occurrences loaded, want none", ind.IndicatorID, len(ind.Occurrences))
		}
	}
	if len(got) != 3 {
		t.Fatalf("got %d manual indicators, want 3: %+v", len(got), got)
	}
	for _, id := range []string{manual, fp, mixed} {
		if _, ok := ids[id]; !ok {
			t.Errorf("indicator %s missing from manual list", id)
		}
	}
	if ind := ids[fp]; ind.Status != ioc.IOCStatusFalsePositive {
		t.Errorf("false positive indicator status = %s", ind.Status)
	}
	if ind := ids[manual]; ind.Type != ioc.IOCTypeFile || ind.Value != "NTUSER.DAT" || ind.MatchKey != "NTUSER.DAT" {
		t.Errorf("manual indicator = %+v, want type, value and match key filled", ind)
	}

	other, err := repo.ListManualIndicators(ctx, "case-2")
	if err != nil || len(other) != 1 || other[0].CaseID != "case-2" {
		t.Errorf("case-2 manual indicators = %+v, %v; want only its own", other, err)
	}
}

func TestUpdateIndicatorStatus(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	ctx := context.Background()
	id := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodAuto).IndicatorID
	upsert(t, repo, "b2", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodAuto)

	steps := []struct {
		to            ioc.IOCStatus
		wantConfirmed bool
	}{
		{ioc.IOCStatusConfirmed, true},
		{ioc.IOCStatusFalsePositive, false},
		{ioc.IOCStatusConfirmed, true},
		{ioc.IOCStatusDetected, false},
		{ioc.IOCStatusFalsePositive, false},
		{ioc.IOCStatusDetected, false},
	}
	from := ioc.IOCStatusDetected
	for _, s := range steps {
		before := auditCount(t, db)
		setStatus(t, repo, id, s.to)
		ind := getInd(t, repo, id)
		if ind.Status != s.to || (ind.ConfirmedAt != nil) != s.wantConfirmed {
			t.Fatalf("%s -> %s: status=%s confirmed_at=%v", from, s.to, ind.Status, ind.ConfirmedAt)
		}
		audits := auditsSince(t, db, before)
		if len(audits) != 1 || audits[0].details["from"] != string(from) || audits[0].details["to"] != string(s.to) ||
			audits[0].details["occurrence_count"] != float64(2) {
			t.Fatalf("%s -> %s audits = %+v", from, s.to, audits)
		}
		from = s.to
	}

	before := auditCount(t, db)
	setStatus(t, repo, id, ioc.IOCStatusDetected)
	if n := auditCount(t, db) - before; n != 0 {
		t.Fatalf("same-status update wrote %d audits", n)
	}

	for _, bad := range []ioc.IOCStatus{ioc.IOCStatusPromoted, "bogus"} {
		if err := repo.UpdateIndicatorStatus(ctx, id, bad, "user-1"); !errors.Is(err, ioc.ErrInvalidStatusTransition) {
			t.Fatalf("target %q err = %v", bad, err)
		}
	}
	if err := repo.UpdateIndicatorStatus(ctx, "missing", ioc.IOCStatusConfirmed, "user-1"); !errors.Is(err, ioc.ErrIndicatorNotFound) {
		t.Fatalf("missing indicator err = %v", err)
	}

	promote(t, repo, id)
	if err := repo.UpdateIndicatorStatus(ctx, id, ioc.IOCStatusConfirmed, "user-1"); !errors.Is(err, ioc.ErrPromotedLocked) {
		t.Fatalf("promoted err = %v", err)
	}
}

func TestUpdateIndicatorTypePlain(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	ctx := context.Background()
	id := upsert(t, repo, "b1", ioc.IOCTypeDomain, "HTTP://Evil.COM/Path", ioc.DetectionMethodManual).IndicatorID
	before := auditCount(t, db)

	res, err := repo.UpdateIndicatorType(ctx, id, ioc.IOCTypeDomain, false, "user-1")
	if err != nil || res.Changed || auditCount(t, db) != before {
		t.Fatalf("same type = %+v, %v", res, err)
	}
	if _, err := repo.UpdateIndicatorType(ctx, id, "hostname", false, "user-1"); !errors.Is(err, ioc.ErrInvalidType) {
		t.Fatalf("invalid type err = %v", err)
	}

	res, err = repo.UpdateIndicatorType(ctx, id, ioc.IOCTypeURL, false, "user-1")
	if err != nil {
		t.Fatalf("UpdateIndicatorType: %v", err)
	}
	if !res.Changed || res.Merged || res.Collision != nil {
		t.Fatalf("plain retype = %+v", res)
	}
	ind := getInd(t, repo, id)
	if ind.Type != ioc.IOCTypeURL || ind.Status != ioc.IOCStatusDetected || ind.ConfirmedAt != nil ||
		ind.MatchKey != "http://evil.com/Path" || ind.Value != "HTTP://Evil.COM/Path" {
		t.Fatalf("retyped indicator = %+v", ind)
	}
	audits := auditsSince(t, db, before)
	if len(audits) != 1 || audits[0].details["from_type"] != "domain" || audits[0].details["to_type"] != "url" ||
		audits[0].details["from_status"] != "confirmed" {
		t.Fatalf("retype audits = %+v", audits)
	}

	promote(t, repo, id)
	if _, err := repo.UpdateIndicatorType(ctx, id, ioc.IOCTypeDomain, true, "user-1"); !errors.Is(err, ioc.ErrPromotedLocked) {
		t.Fatalf("promoted retype err = %v", err)
	}
}

// mergeFixture builds a source file indicator (b1, b2) and a target
// file_path indicator (b2 with the same raw value as the source, b3) that
// collide when the source is retyped to file_path. A fact points at the
// source.
func mergeFixture(t *testing.T, db *DB, repo *IndicatorRepo) (src, target string) {
	t.Helper()
	const upper, lower = `C:\Temp\x.exe`, `c:\temp\x.exe`
	src = upsert(t, repo, "b1", ioc.IOCTypeFile, upper, ioc.DetectionMethodAuto).IndicatorID
	upsert(t, repo, "b2", ioc.IOCTypeFile, upper, ioc.DetectionMethodAuto)
	target = upsert(t, repo, "b3", ioc.IOCTypeFilePath, lower, ioc.DetectionMethodManual).IndicatorID
	upsert(t, repo, "b2", ioc.IOCTypeFilePath, upper, ioc.DetectionMethodAuto)
	setStatus(t, repo, src, ioc.IOCStatusConfirmed)
	addLinkedFact(t, db, "fact-src", src)
	return src, target
}

func TestUpdateIndicatorTypeMerge(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	ctx := context.Background()
	src, target := mergeFixture(t, db, repo)
	targetBefore := getInd(t, repo, target)
	snapshot := dumpTables(t, db)

	res, err := repo.UpdateIndicatorType(ctx, src, ioc.IOCTypeFilePath, false, "user-1")
	if err != nil {
		t.Fatalf("collision check: %v", err)
	}
	want := &ioc.TypeCollision{TargetIndicatorID: target, TargetStatus: ioc.IOCStatusConfirmed, OccurrencesToMove: 2}
	if res.Changed || res.Merged || !reflect.DeepEqual(res.Collision, want) {
		t.Fatalf("unconfirmed collision = %+v (collision %+v)", res, res.Collision)
	}
	if after := dumpTables(t, db); !reflect.DeepEqual(snapshot, after) {
		t.Fatal("unconfirmed collision changed the database")
	}

	before := auditCount(t, db)
	res, err = repo.UpdateIndicatorType(ctx, src, ioc.IOCTypeFilePath, true, "user-1")
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if !res.Changed || !res.Merged || res.Collision != nil {
		t.Fatalf("merge result = %+v", res)
	}

	if _, err := repo.GetIndicator(ctx, src); !errors.Is(err, ioc.ErrIndicatorNotFound) {
		t.Fatalf("source still present: %v", err)
	}
	tgt := getInd(t, repo, target)
	if got := occBlocks(tgt); !reflect.DeepEqual(got, []string{"b1", "b2", "b3"}) {
		t.Fatalf("target occurrences = %v", got)
	}
	// Precedence(confirmed, detected) keeps confirmed, and confirmed_at is untouched.
	if tgt.Status != ioc.IOCStatusConfirmed || !reflect.DeepEqual(tgt.ConfirmedAt, targetBefore.ConfirmedAt) {
		t.Fatalf("target status=%s confirmed_at=%v, want confirmed %v", tgt.Status, tgt.ConfirmedAt, targetBefore.ConfirmedAt)
	}
	facts, err := repo.ListLinkedFacts(ctx, target)
	if err != nil || len(facts) != 1 || facts[0].FactID != "fact-src" {
		t.Fatalf("linked facts after merge = %+v, %v", facts, err)
	}

	audits := auditsSince(t, db, before)
	if len(audits) != 1 || audits[0].entityID != target {
		t.Fatalf("merge audits = %+v", audits)
	}
	d := audits[0].details
	mf, _ := d["merged_from"].(map[string]any)
	if mf["indicator_id"] != src || mf["type"] != "file" || mf["value"] != `C:\Temp\x.exe` || mf["status"] != "confirmed" ||
		d["occurrences_moved"] != float64(1) || d["duplicates_dropped"] != float64(1) || d["facts_repointed"] != float64(1) {
		t.Fatalf("merge audit details = %+v", d)
	}
}

func TestMergePrecedenceKeepsFalsePositive(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	src, target := mergeFixture(t, db, repo)
	setStatus(t, repo, target, ioc.IOCStatusFalsePositive)

	if _, err := repo.UpdateIndicatorType(context.Background(), src, ioc.IOCTypeFilePath, true, "user-1"); err != nil {
		t.Fatalf("merge: %v", err)
	}
	tgt := getInd(t, repo, target)
	if tgt.Status != ioc.IOCStatusFalsePositive || tgt.ConfirmedAt != nil {
		t.Fatalf("target status=%s confirmed_at=%v, want false_positive", tgt.Status, tgt.ConfirmedAt)
	}
}

func TestPromoteIndicator(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	ctx := context.Background()
	id := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodManual).IndicatorID
	confirmedAt := getInd(t, repo, id).ConfirmedAt
	before := auditCount(t, db)

	if _, err := repo.PromoteIndicator(ctx, id, models.CreateCaseFactRequest{Type: "domain", Value: "x"}, "user-1"); err == nil {
		t.Fatal("promote without label accepted")
	}

	bogusCase, bogusSource, block := "case-2", "not-this", "b1"
	fact, err := repo.PromoteIndicator(ctx, id, models.CreateCaseFactRequest{
		CaseID: bogusCase, SourceIOCID: &bogusSource, SourceBlockID: &block,
		Type: "domain", Label: "C2 domain", Value: "evil.com", Notes: "beacon",
	}, "user-1")
	if err != nil {
		t.Fatalf("PromoteIndicator: %v", err)
	}
	if fact.CaseID != "case-1" || fact.SourceIOCID == nil || *fact.SourceIOCID != id || fact.FactID == "" {
		t.Fatalf("fact = %+v", fact)
	}
	stored, err := NewCaseFactRepo(db).GetByID(ctx, fact.FactID)
	if err != nil || !reflect.DeepEqual(stored, fact) {
		t.Fatalf("stored fact = %+v, %v; want %+v", stored, err, fact)
	}

	ind := getInd(t, repo, id)
	if ind.Status != ioc.IOCStatusPromoted || !reflect.DeepEqual(ind.ConfirmedAt, confirmedAt) {
		t.Fatalf("promoted status=%s confirmed_at=%v, want %v", ind.Status, ind.ConfirmedAt, confirmedAt)
	}

	audits := auditsSince(t, db, before)
	if len(audits) != 2 {
		t.Fatalf("expected 2 audits, got %+v", audits)
	}
	wantFact := map[string]any{"fact_id": fact.FactID, "type": "domain", "label": "C2 domain"}
	if audits[0].action != "CREATE" || audits[0].entityType != "case_fact" || !reflect.DeepEqual(audits[0].details, wantFact) {
		t.Fatalf("fact audit = %+v", audits[0])
	}
	wantInd := map[string]any{"from": "confirmed", "to": "promoted", "fact_id": fact.FactID}
	if audits[1].action != "UPDATE" || audits[1].entityType != "indicator" || !reflect.DeepEqual(audits[1].details, wantInd) {
		t.Fatalf("indicator audit = %+v", audits[1])
	}

	if _, err := repo.PromoteIndicator(ctx, id, models.CreateCaseFactRequest{Type: "d", Label: "l", Value: "v"}, "user-1"); !errors.Is(err, ioc.ErrInvalidStatusTransition) {
		t.Fatalf("promote from promoted err = %v", err)
	}
	fp := upsert(t, repo, "b2", ioc.IOCTypeDomain, "fp.example", ioc.DetectionMethodAuto).IndicatorID
	setStatus(t, repo, fp, ioc.IOCStatusFalsePositive)
	if _, err := repo.PromoteIndicator(ctx, fp, models.CreateCaseFactRequest{Type: "d", Label: "l", Value: "v"}, "user-1"); !errors.Is(err, ioc.ErrInvalidStatusTransition) {
		t.Fatalf("promote from false_positive err = %v", err)
	}
}

func TestRestoreIndicator(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	ctx := context.Background()

	id := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodManual).IndicatorID
	if _, err := repo.RestoreIndicator(ctx, id, "user-1"); !errors.Is(err, ioc.ErrInvalidStatusTransition) {
		t.Fatalf("restore non-promoted err = %v", err)
	}

	first := promote(t, repo, id)
	addLinkedFact(t, db, "fact-2", id)
	before := auditCount(t, db)

	deleted, err := repo.RestoreIndicator(ctx, id, "user-1")
	if err != nil {
		t.Fatalf("RestoreIndicator: %v", err)
	}
	got := slices.Sorted(slices.Values(deleted))
	if want := slices.Sorted(slices.Values([]string{first.FactID, "fact-2"})); !reflect.DeepEqual(got, want) {
		t.Fatalf("deleted = %v, want %v", deleted, want)
	}
	if facts, err := repo.ListLinkedFacts(ctx, id); err != nil || len(facts) != 0 {
		t.Fatalf("facts after restore = %+v, %v", facts, err)
	}
	ind := getInd(t, repo, id)
	if ind.Status != ioc.IOCStatusDetected || ind.ConfirmedAt != nil {
		t.Fatalf("restored status=%s confirmed_at=%v", ind.Status, ind.ConfirmedAt)
	}

	audits := auditsSince(t, db, before)
	if len(audits) != 3 {
		t.Fatalf("expected 3 audits, got %+v", audits)
	}
	for i, factID := range deleted {
		if audits[i].action != "DELETE" || audits[i].entityType != "case_fact" || audits[i].entityID != factID ||
			audits[i].details["fact_id"] != factID || audits[i].details["label"] == nil || audits[i].details["type"] == nil {
			t.Fatalf("fact delete audit %d = %+v", i, audits[i])
		}
	}
	ids, _ := audits[2].details["deleted_fact_ids"].([]any)
	if audits[2].entityType != "indicator" || audits[2].details["from"] != "promoted" ||
		audits[2].details["to"] != "detected" || len(ids) != 2 {
		t.Fatalf("indicator restore audit = %+v", audits[2])
	}
}

func TestRestoreIndicatorWithoutFacts(t *testing.T) {
	db, repo := newIndicatorFixture(t)
	ctx := context.Background()
	id := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodAuto).IndicatorID
	fact := promote(t, repo, id)
	if err := NewCaseFactRepo(db).Delete(ctx, fact.FactID); err != nil {
		t.Fatalf("delete fact: %v", err)
	}

	deleted, err := repo.RestoreIndicator(ctx, id, "user-1")
	if err != nil {
		t.Fatalf("RestoreIndicator: %v", err)
	}
	if deleted == nil || len(deleted) != 0 {
		t.Fatalf("deleted = %#v, want empty slice", deleted)
	}
	if got := getInd(t, repo, id).Status; got != ioc.IOCStatusDetected {
		t.Fatalf("status = %s", got)
	}
}

func TestIndicatorWritesAreAtomic(t *testing.T) {
	ctx := context.Background()

	t.Run("promote", func(t *testing.T) {
		db, repo := newIndicatorFixture(t)
		id := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodManual).IndicatorID
		failIndicatorAudits(t, db)
		before := dumpTables(t, db)

		_, err := repo.PromoteIndicator(ctx, id, models.CreateCaseFactRequest{Type: "domain", Label: "l", Value: "v"}, "user-1")
		requireForcedFailure(t, err)
		if after := dumpTables(t, db); !reflect.DeepEqual(before, after) {
			t.Fatalf("failed promote left changes:\nbefore %v\nafter  %v", before, after)
		}
	})

	t.Run("restore", func(t *testing.T) {
		db, repo := newIndicatorFixture(t)
		id := upsert(t, repo, "b1", ioc.IOCTypeDomain, "evil.com", ioc.DetectionMethodManual).IndicatorID
		promote(t, repo, id)
		addLinkedFact(t, db, "fact-2", id)
		failIndicatorAudits(t, db)
		before := dumpTables(t, db)

		_, err := repo.RestoreIndicator(ctx, id, "user-1")
		requireForcedFailure(t, err)
		if after := dumpTables(t, db); !reflect.DeepEqual(before, after) {
			t.Fatalf("failed restore left changes:\nbefore %v\nafter  %v", before, after)
		}
	})

	t.Run("merge", func(t *testing.T) {
		db, repo := newIndicatorFixture(t)
		src, _ := mergeFixture(t, db, repo)
		failIndicatorAudits(t, db)
		before := dumpTables(t, db)

		_, err := repo.UpdateIndicatorType(ctx, src, ioc.IOCTypeFilePath, true, "user-1")
		requireForcedFailure(t, err)
		if after := dumpTables(t, db); !reflect.DeepEqual(before, after) {
			t.Fatalf("failed merge left changes:\nbefore %v\nafter  %v", before, after)
		}
	})
}
