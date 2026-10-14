package ioc_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dfnotes-go/internal/database"
	"dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
	"dfnotes-go/internal/services"
)

const (
	testUserID = "user-1"
	testCaseID = "case-1"

	sha256Lower = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// Three detectable values: one IPv4, one domain, one SHA256.
	detectContent = "Beacon from 10.1.2.3 to evil-example.com, payload hash " + sha256Lower
)

// newService returns an IOCService on a migrated temporary database holding
// user-1 and case-1 with blocks b1 to b3 (b1 on evidence ev-1). The session
// is authenticated as user-1.
func newService(t *testing.T) (*ioc.IOCService, *database.DB) {
	t.Helper()
	db := newFixtureDB(t)
	session := services.NewSession()
	session.SetAuthenticated(&models.UserIdentity{UserID: testUserID}, []byte("test-derived-key"), nil)
	return ioc.NewIOCService(database.NewIndicatorRepo(db), session), db
}

func newFixtureDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if err := database.NewUserRepo(db).Create(ctx, &models.UserIdentity{
		UserID:              testUserID,
		Name:                "Test User",
		Organization:        "Test Org",
		PublicKey:           []byte("pub"),
		EncryptedPrivateKey: []byte("priv"),
		Salt:                []byte("salt"),
		CreatedAt:           now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ts := database.FormatTime(now)
	exec(t, db, `INSERT INTO cases (case_id, title, created_by, created_at, updated_at)
		VALUES (?, 'Test', ?, ?, ?)`, testCaseID, testUserID, ts, ts)
	exec(t, db, `INSERT INTO evidence_items (evidence_item_id, case_id, name, collected_by, collected_at, created_at, updated_at)
		VALUES ('ev-1', ?, 'Laptop', ?, ?, ?, ?)`, testCaseID, testUserID, ts, ts, ts)
	for _, b := range []struct {
		id string
		ev any
	}{{"b1", "ev-1"}, {"b2", nil}, {"b3", nil}} {
		exec(t, db, `INSERT INTO note_blocks (block_id, case_id, evidence_item_id, content_hash, prev_hash, signature, encrypted_body, author_id, created_at)
			VALUES (?, ?, ?, 'h', 'p', x'00', x'00', ?, ?)`, b.id, testCaseID, b.ev, testUserID, ts)
	}
	return db
}

func exec(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func detect(t *testing.T, svc *ioc.IOCService, block, content string) {
	t.Helper()
	if err := svc.DetectAndStore(context.Background(), testCaseID, block, content, testUserID); err != nil {
		t.Fatalf("DetectAndStore(%s): %v", block, err)
	}
}

func indicators(t *testing.T, svc *ioc.IOCService) []ioc.Indicator {
	t.Helper()
	inds, err := svc.GetCaseIndicators(context.Background(), testCaseID, true)
	if err != nil {
		t.Fatalf("GetCaseIndicators: %v", err)
	}
	return inds
}

func byType(t *testing.T, inds []ioc.Indicator, typ ioc.IOCType) ioc.Indicator {
	t.Helper()
	for _, ind := range inds {
		if ind.Type == typ {
			return ind
		}
	}
	t.Fatalf("no %s indicator in %d indicators", typ, len(inds))
	return ioc.Indicator{}
}

func byID(inds []ioc.Indicator, id string) (ioc.Indicator, bool) {
	for _, ind := range inds {
		if ind.IndicatorID == id {
			return ind, true
		}
	}
	return ioc.Indicator{}, false
}

func mark(t *testing.T, svc *ioc.IOCService, block, typ, value string) ioc.UpsertResult {
	t.Helper()
	res, err := svc.MarkManualIOC(context.Background(), testCaseID, block, typ, value)
	if err != nil {
		t.Fatalf("MarkManualIOC(%s, %s, %q): %v", block, typ, value, err)
	}
	return res
}

func factRequest() models.CreateCaseFactRequest {
	return models.CreateCaseFactRequest{Type: "network", Label: "C2 domain", Value: "evil-example.com"}
}

func TestDetectAndStore(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)

	detect(t, svc, "b1", detectContent)
	inds := indicators(t, svc)
	if len(inds) != 3 {
		t.Fatalf("got %d indicators, want 3: %+v", len(inds), inds)
	}
	for _, typ := range []ioc.IOCType{ioc.IOCTypeIPv4, ioc.IOCTypeDomain, ioc.IOCTypeSHA256} {
		ind := byType(t, inds, typ)
		if ind.Status != ioc.IOCStatusDetected {
			t.Errorf("%s status = %s, want detected", typ, ind.Status)
		}
		if len(ind.Occurrences) != 1 || ind.Occurrences[0].DetectionMethod != ioc.DetectionMethodAuto {
			t.Errorf("%s occurrences = %+v, want one auto occurrence", typ, ind.Occurrences)
		}
	}

	sha := byType(t, inds, ioc.IOCTypeSHA256)
	if err := svc.UpdateIndicatorStatus(ctx, sha.IndicatorID, "confirmed"); err != nil {
		t.Fatalf("confirm sha256: %v", err)
	}

	detect(t, svc, "b2", "Same file seen again: "+strings.ToUpper(sha256Lower))
	inds = indicators(t, svc)
	if len(inds) != 3 {
		t.Fatalf("after second block got %d indicators, want 3", len(inds))
	}
	sha, ok := byID(inds, sha.IndicatorID)
	if !ok {
		t.Fatal("sha256 indicator disappeared")
	}
	if sha.Status != ioc.IOCStatusConfirmed {
		t.Errorf("sha256 status = %s, want confirmed", sha.Status)
	}
	if len(sha.Occurrences) != 2 || sha.Occurrences[1].BlockID != "b2" ||
		sha.Occurrences[1].RawValue != strings.ToUpper(sha256Lower) {
		t.Errorf("sha256 occurrences = %+v, want b1 then b2 with the uppercase raw value", sha.Occurrences)
	}

	block, err := svc.GetBlockIndicators(ctx, "b2")
	if err != nil {
		t.Fatalf("GetBlockIndicators(b2): %v", err)
	}
	if len(block) != 1 || block[0].IndicatorID != sha.IndicatorID || block[0].Status != ioc.IOCStatusConfirmed {
		t.Errorf("b2 block iocs = %+v, want the confirmed sha256 indicator", block)
	}
}

func TestDetectAndStoreInvalidBlock(t *testing.T) {
	svc, _ := newService(t)
	err := svc.DetectAndStore(context.Background(), testCaseID, "no-such-block", detectContent, testUserID)
	if err == nil {
		t.Fatal("DetectAndStore on a missing block returned nil error")
	}
	if n := len(indicators(t, svc)); n != 0 {
		t.Errorf("got %d indicators after failed detection, want 0", n)
	}
}

func TestDetectAndStoreNoMatches(t *testing.T) {
	svc, _ := newService(t)
	detect(t, svc, "b1", "nothing interesting here")
	if n := len(indicators(t, svc)); n != 0 {
		t.Errorf("got %d indicators, want 0", n)
	}
}

// blockIOCsFor returns the block's occurrences that belong to indicatorID.
func blockIOCsFor(t *testing.T, svc *ioc.IOCService, block, indicatorID string) []ioc.BlockIOC {
	t.Helper()
	all, err := svc.GetBlockIndicators(context.Background(), block)
	if err != nil {
		t.Fatalf("GetBlockIndicators(%s): %v", block, err)
	}
	var out []ioc.BlockIOC
	for _, b := range all {
		if b.IndicatorID == indicatorID {
			out = append(out, b)
		}
	}
	return out
}

func occurrenceCount(t *testing.T, db *database.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM ioc_occurrences").Scan(&n); err != nil {
		t.Fatalf("count occurrences: %v", err)
	}
	return n
}

func TestDetectAndStorePropagatesManual(t *testing.T) {
	svc, _ := newService(t)
	res := mark(t, svc, "b1", "file", "NTUSER.DAT")

	detect(t, svc, "b2", `Loaded C:\Users\x\NTUSER.DAT into Registry Explorer`)

	got := blockIOCsFor(t, svc, "b2", res.IndicatorID)
	if len(got) != 1 {
		t.Fatalf("b2 occurrences of the manual indicator = %+v, want 1", got)
	}
	o := got[0]
	if o.DetectionMethod != ioc.DetectionMethodPropagated || o.Status != ioc.IOCStatusConfirmed ||
		o.RawValue != "NTUSER.DAT" || o.Type != ioc.IOCTypeFile {
		t.Errorf("b2 occurrence = %+v, want propagated confirmed file NTUSER.DAT", o)
	}

	ind, ok := byID(indicators(t, svc), res.IndicatorID)
	if !ok || len(ind.Occurrences) != 2 {
		t.Errorf("manual indicator = %+v, want occurrences in b1 and b2", ind)
	}
}

func TestDetectAndStoreNoRetroactivePropagation(t *testing.T) {
	svc, _ := newService(t)
	detect(t, svc, "b1", "Copied NTUSER.DAT off the image")
	res := mark(t, svc, "b2", "file", "NTUSER.DAT")

	if got := blockIOCsFor(t, svc, "b1", res.IndicatorID); len(got) != 0 {
		t.Errorf("block committed before the manual mark gained %+v", got)
	}
	ind, _ := byID(indicators(t, svc), res.IndicatorID)
	if len(ind.Occurrences) != 1 || ind.Occurrences[0].BlockID != "b2" {
		t.Errorf("manual indicator occurrences = %+v, want only b2", ind.Occurrences)
	}
}

func TestDetectAndStoreAutoWinsOverPropagated(t *testing.T) {
	svc, _ := newService(t)
	res := mark(t, svc, "b1", "domain", "evil-example.com")

	detect(t, svc, "b2", "Beacon to evil-example.com again")

	got := blockIOCsFor(t, svc, "b2", res.IndicatorID)
	if len(got) != 1 || got[0].DetectionMethod != ioc.DetectionMethodAuto {
		t.Fatalf("b2 occurrences = %+v, want exactly one auto occurrence", got)
	}
	if got[0].Status != ioc.IOCStatusConfirmed {
		t.Errorf("status = %s, want confirmed from the manual mark", got[0].Status)
	}
}

func TestDetectAndStorePropagatesFalsePositive(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)
	res := mark(t, svc, "b1", "file", "NTUSER.DAT")
	if err := svc.UpdateIndicatorStatus(ctx, res.IndicatorID, "false_positive"); err != nil {
		t.Fatalf("mark false positive: %v", err)
	}

	detect(t, svc, "b2", "Parsed NTUSER.DAT")

	got := blockIOCsFor(t, svc, "b2", res.IndicatorID)
	if len(got) != 1 || got[0].Status != ioc.IOCStatusFalsePositive ||
		got[0].DetectionMethod != ioc.DetectionMethodPropagated {
		t.Errorf("b2 occurrences = %+v, want one propagated false_positive", got)
	}
}

func TestDetectAndStorePropagationNoMatch(t *testing.T) {
	svc, db := newService(t)
	mark(t, svc, "b1", "file", "NTUSER.DAT")
	before := occurrenceCount(t, db)

	detect(t, svc, "b2", "nothing interesting here, not even ntuser.dat.log")

	if after := occurrenceCount(t, db); after != before {
		t.Errorf("occurrence count went from %d to %d, want unchanged", before, after)
	}
}

// failingManualList returns an error from ListManualIndicators and passes
// every other call through to the real repository.
type failingManualList struct {
	ioc.IndicatorRepository
}

func (failingManualList) ListManualIndicators(context.Context, string) ([]ioc.Indicator, error) {
	return nil, errors.New("forced list failure")
}

func TestDetectAndStoreManualListErrorKeepsRegexResults(t *testing.T) {
	db := newFixtureDB(t)
	session := services.NewSession()
	session.SetAuthenticated(&models.UserIdentity{UserID: testUserID}, []byte("k"), nil)
	svc := ioc.NewIOCService(failingManualList{database.NewIndicatorRepo(db)}, session)

	err := svc.DetectAndStore(context.Background(), testCaseID, "b1", detectContent, testUserID)
	if err == nil || !strings.Contains(err.Error(), "forced list failure") {
		t.Fatalf("DetectAndStore error = %v, want the list failure", err)
	}
	if n := len(indicators(t, svc)); n != 3 {
		t.Errorf("got %d indicators, want the 3 regex results stored", n)
	}
}

func TestMarkManualIOC(t *testing.T) {
	svc, _ := newService(t)

	res := mark(t, svc, "b1", "domain", "manual.example.org")
	if !res.Created || res.Status != ioc.IOCStatusConfirmed || res.AlreadyTracked {
		t.Errorf("new manual mark = %+v, want created and confirmed", res)
	}

	detect(t, svc, "b1", detectContent)
	var detected ioc.Indicator
	for _, ind := range indicators(t, svc) {
		if ind.Type == ioc.IOCTypeDomain && ind.Value == "evil-example.com" {
			detected = ind
		}
	}
	if detected.Status != ioc.IOCStatusDetected {
		t.Fatalf("evil-example.com status = %s, want detected", detected.Status)
	}

	raised := mark(t, svc, "b2", "domain", "EVIL-example.com")
	if raised.IndicatorID != detected.IndicatorID || raised.Created ||
		!raised.StatusChanged || raised.Status != ioc.IOCStatusConfirmed {
		t.Errorf("manual mark on detected value = %+v, want existing indicator raised to confirmed", raised)
	}

	again, err := svc.MarkManualIOC(context.Background(), testCaseID, "b2", "domain", "evil-example.com")
	if err != nil {
		t.Fatalf("marking again in the same block returned error: %v", err)
	}
	if !again.AlreadyTracked || again.IndicatorID != detected.IndicatorID {
		t.Errorf("repeat manual mark = %+v, want AlreadyTracked", again)
	}
}

// upsertSpy fails the test if the service reaches the repository. Embedding
// the interface leaves every other method nil, so any other call panics.
type upsertSpy struct {
	ioc.IndicatorRepository
	called bool
}

func (s *upsertSpy) UpsertOccurrence(context.Context, ioc.UpsertOccurrenceInput) (ioc.UpsertResult, error) {
	s.called = true
	return ioc.UpsertResult{}, nil
}

func TestMarkManualIOCInvalidType(t *testing.T) {
	spy := &upsertSpy{}
	session := services.NewSession()
	session.SetAuthenticated(&models.UserIdentity{UserID: testUserID}, []byte("k"), nil)
	svc := ioc.NewIOCService(spy, session)

	_, err := svc.MarkManualIOC(context.Background(), testCaseID, "b1", "hostname", "box01")
	if err == nil {
		t.Fatal("invalid type accepted")
	}
	if !errors.Is(err, ioc.ErrInvalidType) {
		t.Errorf("errors.Is(err, ErrInvalidType) = false for %v", err)
	}
	if err.Error() != "That is not a valid IOC type." {
		t.Errorf("message = %q", err.Error())
	}
	if spy.called {
		t.Error("invalid type reached the repository")
	}
}

func TestPromotedIndicatorIsLocked(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)
	res := mark(t, svc, "b1", "domain", "evil-example.com")
	if _, err := svc.PromoteIndicator(ctx, res.IndicatorID, factRequest()); err != nil {
		t.Fatalf("PromoteIndicator: %v", err)
	}

	err := svc.UpdateIndicatorStatus(ctx, res.IndicatorID, "false_positive")
	if !errors.Is(err, ioc.ErrPromotedLocked) {
		t.Fatalf("status change on promoted: errors.Is(ErrPromotedLocked) false for %v", err)
	}
	if want := "This indicator is promoted to a case fact. Restore it before changing its status."; err.Error() != want {
		t.Errorf("status message = %q, want %q", err.Error(), want)
	}

	_, err = svc.UpdateIndicatorType(ctx, res.IndicatorID, "url", false)
	if !errors.Is(err, ioc.ErrPromotedLocked) {
		t.Fatalf("type change on promoted: errors.Is(ErrPromotedLocked) false for %v", err)
	}
	if want := "This indicator is promoted to a case fact. Restore it before changing its type."; err.Error() != want {
		t.Errorf("type message = %q, want %q", err.Error(), want)
	}
}

func TestUpdateIndicatorStatusValidation(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)
	res := mark(t, svc, "b1", "domain", "evil-example.com")

	for _, status := range []string{"promoted", "bogus", ""} {
		err := svc.UpdateIndicatorStatus(ctx, res.IndicatorID, status)
		if !errors.Is(err, ioc.ErrInvalidStatusTransition) {
			t.Errorf("status %q: errors.Is(ErrInvalidStatusTransition) false for %v", status, err)
			continue
		}
		if err.Error() != "Status must be detected, confirmed, or false positive." {
			t.Errorf("status %q: message = %q", status, err.Error())
		}
	}

	err := svc.UpdateIndicatorStatus(ctx, "no-such-indicator", "confirmed")
	if !errors.Is(err, ioc.ErrIndicatorNotFound) || !errors.Is(err, models.ErrNotFound) {
		t.Errorf("missing indicator: want ErrIndicatorNotFound and models.ErrNotFound, got %v", err)
	}
	if err != nil && err.Error() != "This indicator no longer exists. Refresh the IOC list and try again." {
		t.Errorf("missing indicator message = %q", err.Error())
	}
}

func TestUpdateIndicatorTypeCollision(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)
	target := mark(t, svc, "b1", "domain", "collide.example.com")
	src := mark(t, svc, "b2", "file", "collide.example.com")

	res, err := svc.UpdateIndicatorType(ctx, src.IndicatorID, "domain", false)
	if err != nil {
		t.Fatalf("UpdateIndicatorType without confirm: %v", err)
	}
	if res.Changed || res.Merged || res.Collision == nil ||
		res.Collision.TargetIndicatorID != target.IndicatorID || res.Collision.OccurrencesToMove != 1 {
		t.Fatalf("first call = %+v, want an unconfirmed collision with the domain indicator", res)
	}
	inds := indicators(t, svc)
	if s, ok := byID(inds, src.IndicatorID); !ok || s.Type != ioc.IOCTypeFile {
		t.Errorf("source after unconfirmed collision = %+v, want unchanged file indicator", s)
	}
	if tg, _ := byID(inds, target.IndicatorID); len(tg.Occurrences) != 1 {
		t.Errorf("target occurrences after unconfirmed collision = %d, want 1", len(tg.Occurrences))
	}

	res, err = svc.UpdateIndicatorType(ctx, src.IndicatorID, "domain", true)
	if err != nil {
		t.Fatalf("UpdateIndicatorType with confirm: %v", err)
	}
	if !res.Changed || !res.Merged || res.Collision != nil {
		t.Fatalf("second call = %+v, want merged", res)
	}
	inds = indicators(t, svc)
	if _, ok := byID(inds, src.IndicatorID); ok {
		t.Error("source indicator still exists after merge")
	}
	tg, ok := byID(inds, target.IndicatorID)
	if !ok || len(tg.Occurrences) != 2 {
		t.Errorf("target after merge = %+v, want 2 occurrences", tg)
	}
}

func TestUpdateIndicatorTypeInvalid(t *testing.T) {
	svc, _ := newService(t)
	res := mark(t, svc, "b1", "domain", "evil-example.com")
	_, err := svc.UpdateIndicatorType(context.Background(), res.IndicatorID, "hostname", false)
	if !errors.Is(err, ioc.ErrInvalidType) {
		t.Errorf("errors.Is(ErrInvalidType) false for %v", err)
	}
}

func TestPromoteLinkedFactsRestore(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)
	res := mark(t, svc, "b1", "domain", "evil-example.com")

	fact, err := svc.PromoteIndicator(ctx, res.IndicatorID, factRequest())
	if err != nil {
		t.Fatalf("PromoteIndicator: %v", err)
	}
	if fact.SourceIOCID == nil || *fact.SourceIOCID != res.IndicatorID || fact.CaseID != testCaseID || fact.UserID != testUserID {
		t.Errorf("promoted fact = %+v, want source %s in %s by %s", fact, res.IndicatorID, testCaseID, testUserID)
	}
	if ind, _ := byID(indicators(t, svc), res.IndicatorID); ind.Status != ioc.IOCStatusPromoted {
		t.Errorf("status after promote = %s, want promoted", ind.Status)
	}

	_, err = svc.PromoteIndicator(ctx, res.IndicatorID, factRequest())
	if !errors.Is(err, ioc.ErrInvalidStatusTransition) {
		t.Errorf("second promote: errors.Is(ErrInvalidStatusTransition) false for %v", err)
	} else if err.Error() != "Only a detected or confirmed indicator can be promoted to a case fact." {
		t.Errorf("second promote message = %q", err.Error())
	}

	linked, err := svc.GetLinkedFacts(ctx, res.IndicatorID)
	if err != nil {
		t.Fatalf("GetLinkedFacts: %v", err)
	}
	if len(linked) != 1 || linked[0].FactID != fact.FactID {
		t.Fatalf("linked facts = %+v, want [%s]", linked, fact.FactID)
	}

	deleted, err := svc.RestoreIndicator(ctx, res.IndicatorID)
	if err != nil {
		t.Fatalf("RestoreIndicator: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != fact.FactID {
		t.Errorf("deleted facts = %v, want [%s]", deleted, fact.FactID)
	}
	if ind, _ := byID(indicators(t, svc), res.IndicatorID); ind.Status != ioc.IOCStatusDetected {
		t.Errorf("status after restore = %s, want detected", ind.Status)
	}
	linked, err = svc.GetLinkedFacts(ctx, res.IndicatorID)
	if err != nil || len(linked) != 0 {
		t.Errorf("linked facts after restore = %+v, %v; want none", linked, err)
	}

	_, err = svc.RestoreIndicator(ctx, res.IndicatorID)
	if !errors.Is(err, ioc.ErrInvalidStatusTransition) {
		t.Errorf("restore of detected: errors.Is(ErrInvalidStatusTransition) false for %v", err)
	} else if err.Error() != "Only a promoted indicator can be restored." {
		t.Errorf("restore of detected message = %q", err.Error())
	}
}

func TestListExportRows(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)
	detect(t, svc, "b1", detectContent)
	detect(t, svc, "b2", strings.ToUpper(sha256Lower))

	inds := indicators(t, svc)
	ip := byType(t, inds, ioc.IOCTypeIPv4)
	sha := byType(t, inds, ioc.IOCTypeSHA256)
	if err := svc.UpdateIndicatorStatus(ctx, ip.IndicatorID, "false_positive"); err != nil {
		t.Fatalf("mark ip false positive: %v", err)
	}
	if err := svc.UpdateIndicatorStatus(ctx, sha.IndicatorID, "confirmed"); err != nil {
		t.Fatalf("confirm sha256: %v", err)
	}

	rows, err := svc.ListExportRows(ctx, testCaseID)
	if err != nil {
		t.Fatalf("ListExportRows: %v", err)
	}

	// Expected rows follow repository indicator order, then chain order.
	inds = indicators(t, svc)
	var want []struct {
		ind ioc.Indicator
		occ ioc.Occurrence
	}
	for _, ind := range inds {
		for _, o := range ind.Occurrences {
			want = append(want, struct {
				ind ioc.Indicator
				occ ioc.Occurrence
			}{ind, o})
		}
	}
	if len(want) != 4 {
		t.Fatalf("fixture has %d occurrences, want 4", len(want))
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d export rows, want %d (one per occurrence)", len(rows), len(want))
	}

	sawFalsePositive := false
	for i, r := range rows {
		w := want[i]
		if r.IOCID != w.occ.OccurrenceID {
			t.Errorf("row %d IOCID = %s, want occurrence id %s", i, r.IOCID, w.occ.OccurrenceID)
		}
		if r.Status != w.ind.Status || r.Type != w.ind.Type {
			t.Errorf("row %d status/type = %s/%s, want indicator's %s/%s", i, r.Status, r.Type, w.ind.Status, w.ind.Type)
		}
		if r.CaseID != testCaseID || r.BlockID != w.occ.BlockID || r.Value != w.occ.RawValue ||
			r.DetectionMethod != w.occ.DetectionMethod || r.CreatedAt != w.occ.CreatedAt ||
			r.UserID != w.occ.UserID || r.Notes != nil {
			t.Errorf("row %d = %+v, does not match occurrence %+v", i, r, w.occ)
		}
		if (r.ConfirmedAt == nil) != (w.ind.ConfirmedAt == nil) {
			t.Errorf("row %d ConfirmedAt = %v, want indicator's %v", i, r.ConfirmedAt, w.ind.ConfirmedAt)
		}
		if r.BlockID == "b1" && (r.EvidenceItemID == nil || *r.EvidenceItemID != "ev-1") {
			t.Errorf("row %d on b1 EvidenceItemID = %v, want ev-1", i, r.EvidenceItemID)
		}
		if r.Status == ioc.IOCStatusFalsePositive {
			sawFalsePositive = true
		}
		if r.Type == ioc.IOCTypeSHA256 && r.Status != ioc.IOCStatusConfirmed {
			t.Errorf("row %d sha256 status = %s, want confirmed from the indicator", i, r.Status)
		}
	}
	if !sawFalsePositive {
		t.Error("export rows left out the false positive indicator")
	}
}

func TestUnauthenticatedSession(t *testing.T) {
	ctx := context.Background()
	db := newFixtureDB(t)
	svc := ioc.NewIOCService(database.NewIndicatorRepo(db), services.NewSession())

	checks := map[string]error{
		"DetectAndStore":        svc.DetectAndStore(ctx, testCaseID, "b1", detectContent, testUserID),
		"UpdateIndicatorStatus": svc.UpdateIndicatorStatus(ctx, "x", "confirmed"),
	}
	_, checks["UpdateIndicatorType"] = svc.UpdateIndicatorType(ctx, "x", "domain", false)
	_, checks["MarkManualIOC"] = svc.MarkManualIOC(ctx, testCaseID, "b1", "domain", "evil-example.com")
	_, checks["PromoteIndicator"] = svc.PromoteIndicator(ctx, "x", factRequest())
	_, checks["RestoreIndicator"] = svc.RestoreIndicator(ctx, "x")
	_, checks["GetCaseIndicators"] = svc.GetCaseIndicators(ctx, testCaseID, true)
	_, checks["GetBlockIndicators"] = svc.GetBlockIndicators(ctx, "b1")
	_, checks["GetLinkedFacts"] = svc.GetLinkedFacts(ctx, "x")
	_, checks["ListExportRows"] = svc.ListExportRows(ctx, testCaseID)

	for name, err := range checks {
		if !errors.Is(err, ioc.ErrNotAuthenticated) {
			t.Errorf("%s without a session: err = %v, want ErrNotAuthenticated", name, err)
		}
	}

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM indicators").Scan(&n); err != nil {
		t.Fatalf("count indicators: %v", err)
	}
	if n != 0 {
		t.Errorf("unauthenticated calls wrote %d indicators, want 0", n)
	}
}
