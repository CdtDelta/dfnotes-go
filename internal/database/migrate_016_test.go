package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

const preIndicatorVersion = 15

// openAtVersion opens a fresh database file and migrates it only up to target.
func openAtVersion(t *testing.T, path string, target int) *DB {
	t.Helper()
	db, err := openConn(path)
	if err != nil {
		t.Fatalf("openConn: %v", err)
	}
	if err := db.migrateTo(path, target); err != nil {
		db.Close()
		t.Fatalf("migrateTo(%d): %v", target, err)
	}
	return db
}

func mustExec(t *testing.T, db *DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func schemaVersion(t *testing.T, db *sql.DB) int {
	t.Helper()
	var v int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&v); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	return v
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&n); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return n == 1
}

func ts(sec int) string {
	return "2026-10-01T12:00:" + twoDigits(sec) + "Z"
}

func twoDigits(n int) string {
	return string([]byte{byte('0' + n/10), byte('0' + n%10)})
}

const (
	shaLower = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	shaUpper = "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"
)

// seedFixtures inserts users, cases, blocks, ioc_entries and case_facts at
// schema version 15.
func seedFixtures(t *testing.T, db *DB) {
	t.Helper()
	mustExec(t, db, `INSERT INTO users (user_id, name, public_key, encrypted_private_key, salt, created_at)
		VALUES ('user-1', 'Examiner', x'00', x'00', x'00', ?)`, ts(0))
	for _, c := range []string{"case-1", "case-2"} {
		mustExec(t, db, `INSERT INTO cases (case_id, title, created_by, created_at, updated_at)
			VALUES (?, ?, 'user-1', ?, ?)`, c, c, ts(0), ts(0))
	}
	mustExec(t, db, `INSERT INTO evidence_items (evidence_item_id, case_id, name, collected_by, collected_at, created_at, updated_at)
		VALUES ('ev-1', 'case-1', 'Laptop', 'user-1', ?, ?, ?)`, ts(0), ts(0), ts(0))
	for _, b := range []struct{ id, caseID string }{
		{"b1", "case-1"}, {"b2", "case-1"}, {"b3", "case-1"}, {"b4", "case-2"},
	} {
		mustExec(t, db, `INSERT INTO note_blocks (block_id, case_id, content_hash, prev_hash, signature, encrypted_body, author_id, created_at)
			VALUES (?, ?, 'h', 'p', x'00', x'00', 'user-1', ?)`, b.id, b.caseID, ts(0))
	}

	type row struct {
		id, caseID, block, typ, value, status, method string
		notes, confirmedAt                            any
		sec                                           int
	}
	rows := []row{
		// sha256 in 3 blocks; earliest row is uppercase and only detected.
		{"s1", "case-1", "b1", "sha256", shaUpper, "detected", "auto", nil, nil, 1},
		{"s2", "case-1", "b2", "sha256", shaLower, "confirmed", "auto", nil, ts(30), 2},
		{"s3", "case-1", "b3", "sha256", shaLower, "false_positive", "auto", nil, nil, 3},
		// Case variants of a domain in the same block.
		{"d1", "case-1", "b1", "domain", "Evil.com", "detected", "auto", nil, nil, 4},
		{"d2", "case-1", "b1", "domain", "evil.com", "detected", "auto", nil, nil, 5},
		// Promoted domain in 2 blocks, each referenced by a case fact.
		{"p1", "case-1", "b1", "domain", "bad.example", "promoted", "auto", nil, ts(31), 6},
		{"p2", "case-1", "b2", "domain", "bad.example", "promoted", "auto", nil, nil, 7},
		// file type is exact match.
		{"fa", "case-1", "b1", "file", "NTUSER.DAT", "confirmed", "manual", nil, ts(32), 8},
		{"fb", "case-1", "b2", "file", "ntuser.dat", "confirmed", "manual", nil, ts(33), 9},
		// ipv6 spellings of one address.
		{"v1", "case-1", "b1", "ipv6", "2001:0db8::1", "detected", "auto", nil, nil, 10},
		{"v2", "case-1", "b2", "ipv6", "2001:DB8::1", "detected", "auto", nil, nil, 11},
		// Single rows: notes, empty notes, NULL notes.
		{"n1", "case-1", "b1", "ipv4", "10.1.1.1", "detected", "auto", "seen in proxy logs", nil, 12},
		{"e1", "case-1", "b1", "ipv4", "10.2.2.2", "detected", "auto", "", nil, 13},
		{"e2", "case-1", "b1", "ipv4", "10.3.3.3", "detected", "auto", nil, nil, 14},
		// Same value in two cases.
		{"x1", "case-1", "b1", "cve", "CVE-2024-1111", "detected", "auto", nil, nil, 15},
		{"x2", "case-2", "b4", "cve", "cve-2024-1111", "detected", "auto", nil, nil, 16},
	}
	for _, r := range rows {
		mustExec(t, db, `INSERT INTO ioc_entries (ioc_id, case_id, block_id, type, value, status,
			detection_method, notes, created_at, confirmed_at, user_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'user-1')`,
			r.id, r.caseID, r.block, r.typ, r.value, r.status, r.method, r.notes, ts(r.sec), r.confirmedAt)
	}

	facts := []struct {
		id      string
		ev      any
		source  any
		srcBlk  any
		notes   string
		created string
	}{
		{"f1", "ev-1", "p1", "b1", "first sighting", ts(20)},
		{"f2", nil, "p2", "b2", "", ts(21)},
		{"f3", nil, nil, nil, "manual fact", ts(22)},
	}
	for _, f := range facts {
		mustExec(t, db, `INSERT INTO case_facts (fact_id, case_id, evidence_item_id, type, label, value,
			source_ioc_id, source_block_id, notes, created_at, updated_at, user_id)
			VALUES (?, 'case-1', ?, 'domain', 'C2 domain', 'bad.example', ?, ?, ?, ?, ?, 'user-1')`,
			f.id, f.ev, f.source, f.srcBlk, f.notes, f.created, f.created)
	}
}

// factSnapshot returns every case_facts column except source_ioc_id, by fact_id.
func factSnapshot(t *testing.T, db *DB) map[string][]any {
	t.Helper()
	rs, err := db.Query(`SELECT fact_id, case_id, evidence_item_id, type, label, value,
		source_block_id, notes, created_at, updated_at, user_id FROM case_facts`)
	if err != nil {
		t.Fatalf("read case_facts: %v", err)
	}
	defer rs.Close()
	out := map[string][]any{}
	for rs.Next() {
		vals := make([]any, 11)
		ptrs := make([]any, 11)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rs.Scan(ptrs...); err != nil {
			t.Fatalf("scan case_facts: %v", err)
		}
		out[vals[0].(string)] = vals
	}
	return out
}

type indicatorRow struct {
	id, value, status, createdAt, userID string
	confirmedAt                          sql.NullString
	version                              int
}

func getIndicator(t *testing.T, db *DB, caseID, typ, matchKey string) indicatorRow {
	t.Helper()
	var r indicatorRow
	err := db.QueryRow(`SELECT indicator_id, value, status, created_at, user_id, confirmed_at, match_key_version
		FROM indicators WHERE case_id=? AND type=? AND match_key=?`, caseID, typ, matchKey).
		Scan(&r.id, &r.value, &r.status, &r.createdAt, &r.userID, &r.confirmedAt, &r.version)
	if err != nil {
		t.Fatalf("indicator %s/%s/%s: %v", caseID, typ, matchKey, err)
	}
	return r
}

func occurrenceBlocks(t *testing.T, db *DB, indicatorID string) []string {
	t.Helper()
	rs, err := db.Query("SELECT block_id FROM ioc_occurrences WHERE indicator_id=? ORDER BY created_at", indicatorID)
	if err != nil {
		t.Fatalf("occurrences: %v", err)
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var b string
		if err := rs.Scan(&b); err != nil {
			t.Fatalf("scan occurrence: %v", err)
		}
		out = append(out, b)
	}
	return out
}

func auditDetails(t *testing.T, db *DB, indicatorID string) []mergeAuditDetails {
	t.Helper()
	rs, err := db.Query(`SELECT action, details FROM audit_logs WHERE entity_type='indicator' AND entity_id=?`, indicatorID)
	if err != nil {
		t.Fatalf("audit query: %v", err)
	}
	defer rs.Close()
	var out []mergeAuditDetails
	for rs.Next() {
		var action, raw string
		if err := rs.Scan(&action, &raw); err != nil {
			t.Fatalf("scan audit: %v", err)
		}
		if action != "UPDATE" {
			t.Errorf("audit action = %q, want UPDATE", action)
		}
		var d mergeAuditDetails
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			t.Fatalf("audit details %q: %v", raw, err)
		}
		out = append(out, d)
	}
	return out
}

func TestMigrate016IOCIndicators(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db := openAtVersion(t, path, preIndicatorVersion)
	defer db.Close()
	seedFixtures(t, db)

	var oldCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM ioc_entries").Scan(&oldCount); err != nil {
		t.Fatal(err)
	}
	oldIDs := map[string]bool{}
	rs, err := db.Query("SELECT ioc_id FROM ioc_entries")
	if err != nil {
		t.Fatal(err)
	}
	for rs.Next() {
		var id string
		rs.Scan(&id)
		oldIDs[id] = true
	}
	rs.Close()
	factsBefore := factSnapshot(t, db)

	if err := db.migrateTo(path, 16); err != nil {
		t.Fatalf("migrate to 16: %v", err)
	}

	t.Run("sha256 merged across blocks", func(t *testing.T) {
		ind := getIndicator(t, db, "case-1", "sha256", shaLower)
		if ind.status != "confirmed" {
			t.Errorf("status = %q, want confirmed", ind.status)
		}
		if ind.value != shaUpper {
			t.Errorf("value = %q, want earliest raw %q", ind.value, shaUpper)
		}
		if ind.createdAt != ts(1) {
			t.Errorf("created_at = %q, want %q", ind.createdAt, ts(1))
		}
		if ind.confirmedAt.String != ts(30) {
			t.Errorf("confirmed_at = %v, want %q", ind.confirmedAt, ts(30))
		}
		if ind.version != 1 {
			t.Errorf("match_key_version = %d, want 1", ind.version)
		}
		if got := occurrenceBlocks(t, db, ind.id); !reflect.DeepEqual(got, []string{"b1", "b2", "b3"}) {
			t.Errorf("occurrence blocks = %v", got)
		}
		audits := auditDetails(t, db, ind.id)
		if len(audits) != 1 {
			t.Fatalf("audit entries = %d, want 1", len(audits))
		}
		a := audits[0]
		if a.Migration != 16 || a.Reason != "ioc_indicator_merge" || a.ResultingStatus != "confirmed" ||
			a.Type != "sha256" || a.MatchKey != shaLower || len(a.SourceRows) != 3 {
			t.Errorf("audit details = %+v", a)
		}
	})

	t.Run("domain case variants in one block", func(t *testing.T) {
		ind := getIndicator(t, db, "case-1", "domain", "evil.com")
		if ind.value != "Evil.com" {
			t.Errorf("value = %q, want Evil.com", ind.value)
		}
		if got := occurrenceBlocks(t, db, ind.id); !reflect.DeepEqual(got, []string{"b1", "b1"}) {
			t.Errorf("occurrence blocks = %v, want [b1 b1]", got)
		}
	})

	t.Run("promoted domain keeps facts", func(t *testing.T) {
		ind := getIndicator(t, db, "case-1", "domain", "bad.example")
		if ind.status != "promoted" {
			t.Errorf("status = %q, want promoted", ind.status)
		}
		if ind.confirmedAt.String != ts(31) {
			t.Errorf("confirmed_at = %v, want %q carried over from p1", ind.confirmedAt, ts(31))
		}

		var raw string
		if err := db.QueryRow(`SELECT details FROM audit_logs WHERE entity_type='indicator' AND entity_id=?`, ind.id).Scan(&raw); err != nil {
			t.Fatalf("promoted audit entry: %v", err)
		}
		var d struct {
			SourceRows []map[string]any `json:"source_rows"`
		}
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			t.Fatalf("audit details %q: %v", raw, err)
		}
		got := map[string]any{}
		for _, r := range d.SourceRows {
			got[r["ioc_id"].(string)] = r["confirmed_at"]
		}
		if got["p1"] != ts(31) {
			t.Errorf("audit p1 confirmed_at = %v, want %q", got["p1"], ts(31))
		}
		if v, present := got["p2"]; !present || v != nil {
			t.Errorf("audit p2 confirmed_at = %v (present=%v), want key omitted", v, present)
		}
		for _, f := range []string{"f1", "f2"} {
			var src string
			if err := db.QueryRow("SELECT source_ioc_id FROM case_facts WHERE fact_id=?", f).Scan(&src); err != nil {
				t.Fatal(err)
			}
			if src != ind.id {
				t.Errorf("%s source_ioc_id = %q, want %q", f, src, ind.id)
			}
		}
	})

	t.Run("file values stay separate", func(t *testing.T) {
		upper := getIndicator(t, db, "case-1", "file", "NTUSER.DAT")
		lower := getIndicator(t, db, "case-1", "file", "ntuser.dat")
		if upper.id == lower.id {
			t.Error("NTUSER.DAT and ntuser.dat merged")
		}
		var method string
		if err := db.QueryRow("SELECT detection_method FROM ioc_occurrences WHERE occurrence_id='fa'").Scan(&method); err != nil {
			t.Fatal(err)
		}
		if method != "manual" {
			t.Errorf("detection_method = %q, want manual", method)
		}
	})

	t.Run("ipv6 spellings merged", func(t *testing.T) {
		ind := getIndicator(t, db, "case-1", "ipv6", "2001:db8::1")
		if n := len(occurrenceBlocks(t, db, ind.id)); n != 2 {
			t.Errorf("occurrences = %d, want 2", n)
		}
	})

	t.Run("notes are audited", func(t *testing.T) {
		ind := getIndicator(t, db, "case-1", "ipv4", "10.1.1.1")
		audits := auditDetails(t, db, ind.id)
		if len(audits) != 1 {
			t.Fatalf("audit entries = %d, want 1", len(audits))
		}
		if audits[0].SourceRows[0].Notes != "seen in proxy logs" {
			t.Errorf("notes = %q", audits[0].SourceRows[0].Notes)
		}
	})

	t.Run("empty notes not audited", func(t *testing.T) {
		for _, v := range []string{"10.2.2.2", "10.3.3.3"} {
			ind := getIndicator(t, db, "case-1", "ipv4", v)
			if n := len(auditDetails(t, db, ind.id)); n != 0 {
				t.Errorf("%s: audit entries = %d, want 0", v, n)
			}
		}
	})

	t.Run("total audit entries", func(t *testing.T) {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE entity_type='indicator'").Scan(&n); err != nil {
			t.Fatal(err)
		}
		// sha256, Evil.com, bad.example, ipv6 merges, plus the notes row.
		if n != 5 {
			t.Errorf("indicator audit entries = %d, want 5", n)
		}
	})

	t.Run("null fact source preserved", func(t *testing.T) {
		var src sql.NullString
		if err := db.QueryRow("SELECT source_ioc_id FROM case_facts WHERE fact_id='f3'").Scan(&src); err != nil {
			t.Fatal(err)
		}
		if src.Valid {
			t.Errorf("f3 source_ioc_id = %q, want NULL", src.String)
		}
	})

	t.Run("same value in two cases", func(t *testing.T) {
		a := getIndicator(t, db, "case-1", "cve", "CVE-2024-1111")
		b := getIndicator(t, db, "case-2", "cve", "CVE-2024-1111")
		if a.id == b.id {
			t.Error("indicators merged across cases")
		}
	})

	t.Run("occurrence ids reuse ioc ids", func(t *testing.T) {
		rs, err := db.Query("SELECT occurrence_id FROM ioc_occurrences")
		if err != nil {
			t.Fatal(err)
		}
		defer rs.Close()
		n := 0
		for rs.Next() {
			var id string
			rs.Scan(&id)
			if !oldIDs[id] {
				t.Errorf("occurrence_id %q is not an old ioc_id", id)
			}
			n++
		}
		if n != oldCount {
			t.Errorf("occurrences = %d, want %d", n, oldCount)
		}
		var indicators int
		db.QueryRow("SELECT COUNT(*) FROM indicators").Scan(&indicators)
		if indicators != 11 {
			t.Errorf("indicators = %d, want 11", indicators)
		}
	})

	t.Run("schema end state", func(t *testing.T) {
		if tableExists(t, db.DB, "ioc_entries") {
			t.Error("ioc_entries still exists")
		}
		if v := schemaVersion(t, db.DB); v != 16 {
			t.Errorf("schema_version = %d, want 16", v)
		}

		rs, err := db.Query("PRAGMA foreign_key_list(case_facts)")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for rs.Next() {
			var id, seq int
			var table, from string
			var to, onUpdate, onDelete, match sql.NullString
			if err := rs.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				t.Fatal(err)
			}
			if from == "source_ioc_id" {
				found = true
				if table != "indicators" || to.String != "indicator_id" {
					t.Errorf("source_ioc_id references %s(%s), want indicators(indicator_id)", table, to.String)
				}
			}
		}
		rs.Close()
		if !found {
			t.Error("no foreign key on case_facts.source_ioc_id")
		}

		var violations int
		rs, err = db.Query("PRAGMA foreign_key_check")
		if err != nil {
			t.Fatal(err)
		}
		for rs.Next() {
			violations++
		}
		rs.Close()
		if violations != 0 {
			t.Errorf("foreign_key_check returned %d rows", violations)
		}

		for _, idx := range []string{"idx_case_facts_case_id", "idx_case_facts_source_ioc_id"} {
			var n int
			db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&n)
			if n != 1 {
				t.Errorf("index %s missing", idx)
			}
		}
	})

	t.Run("case facts otherwise unchanged", func(t *testing.T) {
		if after := factSnapshot(t, db); !reflect.DeepEqual(after, factsBefore) {
			t.Errorf("case_facts changed:\nbefore %v\nafter  %v", factsBefore, after)
		}
	})
}

func TestMigrate016UnmappedFactSourceFails(t *testing.T) {
	// case_facts.source_ioc_id has a foreign key, so an orphan cannot exist
	// with foreign keys on; this exercises the guard directly.
	path := filepath.Join(t.TempDir(), "test.db")
	db := openAtVersion(t, path, 16)
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DROP TABLE case_facts"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`CREATE TABLE case_facts (fact_id TEXT, case_id TEXT, evidence_item_id TEXT,
		type TEXT, label TEXT, value TEXT, source_ioc_id TEXT, source_block_id TEXT, notes TEXT,
		created_at TEXT, updated_at TEXT, user_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO case_facts VALUES ('f9', 'c', NULL, 't', 'l', 'v', 'ghost', NULL, '', 'x', 'x', 'u')`); err != nil {
		t.Fatal(err)
	}
	err = rebuildCaseFacts(tx, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("rebuildCaseFacts error = %v, want unknown ioc_id ghost", err)
	}
}

// The target version floats because Open migrates to the latest version.
var snapshotName = regexp.MustCompile(`^test\.pre-migration-v15-to-v\d+-\d{8}T\d{6}Z\.db$`)

func snapshotFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if strings.Contains(e.Name(), ".pre-migration-") {
			out = append(out, e.Name())
		}
	}
	return out
}

func TestMigrateSnapshotBeforePendingMigration(t *testing.T) {
	// Spaces and a quote prove the VACUUM INTO path is bound safely.
	dir := filepath.Join(t.TempDir(), "o'brien case dir")
	path := filepath.Join(dir, "test.db")
	db := openAtVersion(t, path, preIndicatorVersion)
	seedFixtures(t, db)
	db.Close()

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	db.Close()

	snaps := snapshotFiles(t, dir)
	if len(snaps) != 1 || !snapshotName.MatchString(snaps[0]) {
		t.Fatalf("snapshots = %v, want one matching %s", snaps, snapshotName)
	}

	snap, err := sql.Open("sqlite", filepath.Join(dir, snaps[0]))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	if v := schemaVersion(t, snap); v != preIndicatorVersion {
		t.Errorf("snapshot schema_version = %d, want %d", v, preIndicatorVersion)
	}
	if !tableExists(t, snap, "ioc_entries") {
		t.Error("snapshot is missing ioc_entries")
	}
	var n int
	snap.QueryRow("SELECT COUNT(*) FROM ioc_entries").Scan(&n)
	if n != 16 {
		t.Errorf("snapshot ioc_entries rows = %d, want 16", n)
	}
}

func TestMigrateNoSnapshotForNewDatabase(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	if snaps := snapshotFiles(t, dir); len(snaps) != 0 {
		t.Errorf("snapshots = %v, want none", snaps)
	}
}

func TestMigrateNoSnapshotWhenCurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	for i := 0; i < 2; i++ {
		db, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		db.Close()
	}
	if snaps := snapshotFiles(t, dir); len(snaps) != 0 {
		t.Errorf("snapshots = %v, want none", snaps)
	}
}

func TestMigrateHookFailureRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db := openAtVersion(t, path, preIndicatorVersion)
	defer db.Close()
	seedFixtures(t, db)

	// Run the real hook to completion, then fail, so the rollback has to undo
	// every table change the hook made.
	real := goMigrations[16]
	goMigrations[16] = func(tx *sql.Tx) error {
		if err := real(tx); err != nil {
			return err
		}
		return errors.New("injected failure")
	}
	t.Cleanup(func() { goMigrations[16] = real })

	err := db.migrateTo(path, 16)
	if err == nil || !strings.Contains(err.Error(), "16") {
		t.Fatalf("migrateTo error = %v, want failure naming version 16", err)
	}

	if v := schemaVersion(t, db.DB); v != preIndicatorVersion {
		t.Errorf("schema_version = %d, want %d", v, preIndicatorVersion)
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM ioc_entries").Scan(&n); err != nil {
		t.Fatalf("ioc_entries after rollback: %v", err)
	}
	if n != 16 {
		t.Errorf("ioc_entries rows = %d, want 16", n)
	}
	for _, tbl := range []string{"indicators", "ioc_occurrences", "case_facts_new"} {
		if tableExists(t, db.DB, tbl) {
			t.Errorf("%s exists after rollback", tbl)
		}
	}
	var refTable string
	if err := db.QueryRow(`SELECT "table" FROM pragma_foreign_key_list('case_facts') WHERE "from"='source_ioc_id'`).Scan(&refTable); err != nil {
		t.Fatal(err)
	}
	if refTable != "ioc_entries" {
		t.Errorf("case_facts.source_ioc_id references %s after rollback, want ioc_entries", refTable)
	}
}
