package database

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	iocpkg "dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func createTestUser(t *testing.T, db *DB) *models.UserIdentity {
	t.Helper()
	user := &models.UserIdentity{
		UserID:              "user-1",
		Name:                "Test User",
		Organization:        "Test Org",
		PublicKey:            []byte("pub-key-bytes"),
		EncryptedPrivateKey: []byte("enc-priv-key"),
		Salt:                []byte("salt-bytes"),
		TOTPEnabled:         false,
		CreatedAt:           time.Now().UTC().Truncate(time.Second),
	}
	repo := NewUserRepo(db)
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func createTestCase(t *testing.T, db *DB, userID string) *models.Case {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	c := &models.Case{
		CaseID:         "case-1",
		CaseNumber:     "CASE-2024-001",
		Title:          "Test Case",
		Description:    "A test case",
		Classification: models.ClassificationUnclassified,
		TicketNumber:   "TICKET-123",
		ExaminerName:   "Test User",
		Organization:   "Test Org",
		Salt:           []byte("case-salt-bytes"),
		EncryptedKey:   []byte("case-enc-key"),
		CreatedBy:      userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo := NewCaseRepo(db)
	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("create case: %v", err)
	}
	return c
}

func TestOpenAndMigrate(t *testing.T) {
	db := setupTestDB(t)

	var version int
	err := db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("query schema version: %v", err)
	}
	if version != 6 {
		t.Fatalf("expected version 6, got %d", version)
	}
}

func TestMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db1, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.Close()

	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	db2.Close()
}

func TestSeedTags(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTagRepo(db)
	tags, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 28 {
		t.Fatalf("expected 28 seed tags, got %d", len(tags))
	}
}

func TestDefaultDBPath(t *testing.T) {
	path := DefaultDBPath()
	if path == "" {
		t.Fatal("DefaultDBPath should not be empty")
	}
	if filepath.Base(path) != "dfnotes.db" {
		t.Fatalf("expected dfnotes.db, got %s", filepath.Base(path))
	}
}

func TestUserRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	got, err := repo.GetByID(ctx, user.UserID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != user.Name {
		t.Fatalf("expected name %q, got %q", user.Name, got.Name)
	}

	user.Name = "Updated Name"
	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ = repo.GetByID(ctx, user.UserID)
	if got.Name != "Updated Name" {
		t.Fatalf("expected updated name, got %q", got.Name)
	}
}

func TestUserRepoDuplicate(t *testing.T) {
	db := setupTestDB(t)
	createTestUser(t, db)

	repo := NewUserRepo(db)
	err := repo.Create(context.Background(), &models.UserIdentity{
		UserID:              "user-1",
		Name:                "Duplicate",
		PublicKey:           []byte("key"),
		EncryptedPrivateKey: []byte("key"),
		Salt:                []byte("salt"),
		CreatedAt:           time.Now(),
	})
	if err != models.ErrDuplicateKey {
		t.Fatalf("expected ErrDuplicateKey, got %v", err)
	}
}

func TestUserRepoNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	_, err := repo.GetByID(context.Background(), "nonexistent")
	if err != models.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCaseRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)

	caseRepo := NewCaseRepo(db)
	c := createTestCase(t, db, user.UserID)

	got, err := caseRepo.GetByID(ctx, c.CaseID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != c.Title {
		t.Fatalf("expected title %q, got %q", c.Title, got.Title)
	}
	if got.CaseNumber != "CASE-2024-001" {
		t.Fatalf("expected case number 'CASE-2024-001', got %q", got.CaseNumber)
	}
	if got.TicketNumber != "TICKET-123" {
		t.Fatalf("expected ticket number 'TICKET-123', got %q", got.TicketNumber)
	}
	if got.ExaminerName != "Test User" {
		t.Fatalf("expected examiner 'Test User', got %q", got.ExaminerName)
	}
	if got.Organization != "Test Org" {
		t.Fatalf("expected org 'Test Org', got %q", got.Organization)
	}
	if string(got.Salt) != "case-salt-bytes" {
		t.Fatalf("expected salt, got %q", got.Salt)
	}
	if string(got.EncryptedKey) != "case-enc-key" {
		t.Fatalf("expected encrypted key, got %q", got.EncryptedKey)
	}

	cases, err := caseRepo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(cases))
	}
	if cases[0].CaseNumber != "CASE-2024-001" {
		t.Fatalf("List: expected case number, got %q", cases[0].CaseNumber)
	}

	c.Title = "Updated Case"
	c.CaseNumber = "CASE-2024-002"
	c.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := caseRepo.Update(ctx, c); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ = caseRepo.GetByID(ctx, c.CaseID)
	if got.CaseNumber != "CASE-2024-002" {
		t.Fatalf("expected updated case number, got %q", got.CaseNumber)
	}

	if err := caseRepo.Delete(ctx, c.CaseID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = caseRepo.GetByID(ctx, c.CaseID)
	if err != models.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestUserRepoGetFirst(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	got, err := repo.GetFirst(ctx)
	if err != nil {
		t.Fatalf("GetFirst: %v", err)
	}
	if got.UserID != user.UserID {
		t.Fatalf("expected user %q, got %q", user.UserID, got.UserID)
	}
	if got.Name != user.Name {
		t.Fatalf("expected name %q, got %q", user.Name, got.Name)
	}
}

func TestUserRepoGetFirstEmpty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)

	_, err := repo.GetFirst(context.Background())
	if err != models.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestEvidenceRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)
	c := createTestCase(t, db, user.UserID)

	repo := NewEvidenceRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	item := &models.EvidenceItem{
		EvidenceItemID: "ev-1",
		CaseID:         c.CaseID,
		Name:           "Disk Image",
		Description:    "Server disk image",
		EvidenceType:   models.EvidenceTypeDisk,
		Status:         models.EvidenceStatusCollected,
		ContentHash:    "abc123",
		CustodyLog: []models.CustodyEntry{
			{Timestamp: now, Handler: user.UserID, Action: "collected", Description: "Initial collection"},
		},
		CollectedBy: user.UserID,
		CollectedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, item.EvidenceItemID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Disk Image" {
		t.Fatalf("expected name 'Disk Image', got %q", got.Name)
	}
	if len(got.CustodyLog) != 1 {
		t.Fatalf("expected 1 custody entry, got %d", len(got.CustodyLog))
	}

	items, err := repo.ListByCase(ctx, c.CaseID)
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item.Status = models.EvidenceStatusAnalyzing
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := repo.Update(ctx, item); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := repo.Delete(ctx, item.EvidenceItemID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestNoteBlockRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)
	c := createTestCase(t, db, user.UserID)

	repo := NewNoteBlockRepo(db)
	now := time.Now().UTC().Truncate(time.Second)

	block1 := &models.NoteBlock{
		BlockID:       "block-1",
		CaseID:        c.CaseID,
		ContentHash:   "hash1",
		PrevHash:      "genesis",
		Signature:     []byte("sig1"),
		EncryptedBody: []byte("body1"),
		AuthorID:      user.UserID,
		CreatedAt:     now,
	}
	if err := repo.Create(ctx, block1); err != nil {
		t.Fatalf("Create block1: %v", err)
	}

	block2 := &models.NoteBlock{
		BlockID:       "block-2",
		CaseID:        c.CaseID,
		ContentHash:   "hash2",
		PrevHash:      "hash1",
		Signature:     []byte("sig2"),
		EncryptedBody: []byte("body2"),
		AuthorID:      user.UserID,
		CreatedAt:     now.Add(time.Second),
	}
	if err := repo.Create(ctx, block2); err != nil {
		t.Fatalf("Create block2: %v", err)
	}

	got, err := repo.GetByID(ctx, "block-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ContentHash != "hash1" {
		t.Fatalf("expected hash1, got %s", got.ContentHash)
	}

	blocks, err := repo.ListByCase(ctx, c.CaseID)
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	last, err := repo.GetLastBlock(ctx, c.CaseID)
	if err != nil {
		t.Fatalf("GetLastBlock: %v", err)
	}
	if last.BlockID != "block-2" {
		t.Fatalf("expected block-2 as last, got %s", last.BlockID)
	}
}

func TestNoteBlockWithTags(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)
	c := createTestCase(t, db, user.UserID)

	blockRepo := NewNoteBlockRepo(db)
	tagRepo := NewTagRepo(db)
	now := time.Now().UTC().Truncate(time.Second)

	block := &models.NoteBlock{
		BlockID:       "block-tag-1",
		CaseID:        c.CaseID,
		ContentHash:   "hash",
		PrevHash:      "genesis",
		Signature:     []byte("sig"),
		EncryptedBody: []byte("body"),
		AuthorID:      user.UserID,
		CreatedAt:     now,
	}
	if err := blockRepo.Create(ctx, block); err != nil {
		t.Fatalf("Create block: %v", err)
	}

	if err := tagRepo.AttachToBlock(ctx, block.BlockID, "tag-finding"); err != nil {
		t.Fatalf("AttachToBlock: %v", err)
	}

	got, err := blockRepo.GetByID(ctx, block.BlockID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0].Name != "Finding" {
		t.Fatalf("expected 1 tag 'Finding', got %v", got.Tags)
	}

	if err := tagRepo.DetachFromBlock(ctx, block.BlockID, "tag-finding"); err != nil {
		t.Fatalf("DetachFromBlock: %v", err)
	}

	tags, err := tagRepo.ListByBlock(ctx, block.BlockID)
	if err != nil {
		t.Fatalf("ListByBlock: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("expected 0 tags after detach, got %d", len(tags))
	}
}

func TestNoteBlockWithEvidence(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)
	c := createTestCase(t, db, user.UserID)
	now := time.Now().UTC().Truncate(time.Second)

	evRepo := NewEvidenceRepo(db)
	ev := &models.EvidenceItem{
		EvidenceItemID: "ev-for-block",
		CaseID:         c.CaseID,
		Name:           "Evidence",
		EvidenceType:   models.EvidenceTypeLogs,
		Status:         models.EvidenceStatusCollected,
		CustodyLog:     []models.CustodyEntry{},
		CollectedBy:    user.UserID,
		CollectedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := evRepo.Create(ctx, ev); err != nil {
		t.Fatalf("Create evidence: %v", err)
	}

	blockRepo := NewNoteBlockRepo(db)
	evID := ev.EvidenceItemID
	block := &models.NoteBlock{
		BlockID:        "block-ev-1",
		CaseID:         c.CaseID,
		EvidenceItemID: &evID,
		ContentHash:    "hash",
		PrevHash:       "genesis",
		Signature:      []byte("sig"),
		EncryptedBody:  []byte("body"),
		AuthorID:       user.UserID,
		CreatedAt:      now,
	}
	if err := blockRepo.Create(ctx, block); err != nil {
		t.Fatalf("Create block: %v", err)
	}

	blocks, err := blockRepo.ListByEvidence(ctx, ev.EvidenceItemID)
	if err != nil {
		t.Fatalf("ListByEvidence: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestIOCRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)
	c := createTestCase(t, db, user.UserID)

	blockRepo := NewNoteBlockRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	block := &models.NoteBlock{
		BlockID:       "block-ioc-1",
		CaseID:        c.CaseID,
		ContentHash:   "hash-ioc",
		PrevHash:      "genesis",
		Signature:     []byte("sig"),
		EncryptedBody: []byte("body"),
		AuthorID:      user.UserID,
		CreatedAt:     now,
	}
	if err := blockRepo.Create(ctx, block); err != nil {
		t.Fatalf("Create block: %v", err)
	}

	repo := NewIOCRepo(db)
	createdAt := now.UTC().Format("2006-01-02T15:04:05Z")

	entry := &iocpkg.IOCEntry{
		IOCID:           "ioc-1",
		CaseID:          c.CaseID,
		BlockID:         block.BlockID,
		Type:            iocpkg.IOCTypeIPv4,
		Value:           "192.168.1.100",
		Status:          iocpkg.IOCStatusDetected,
		DetectionMethod: "auto",
		CreatedAt:       createdAt,
		UserID:          user.UserID,
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, "ioc-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Value != "192.168.1.100" {
		t.Fatalf("expected IP, got %q", got.Value)
	}
	if got.Status != iocpkg.IOCStatusDetected {
		t.Fatalf("expected detected status, got %q", got.Status)
	}

	entries, err := repo.ListByCase(ctx, c.CaseID, false)
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if err := repo.UpdateStatus(ctx, "ioc-1", iocpkg.IOCStatusConfirmed, func() *string { s := createdAt; return &s }()); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, _ = repo.GetByID(ctx, "ioc-1")
	if got.Status != iocpkg.IOCStatusConfirmed {
		t.Fatalf("expected confirmed, got %q", got.Status)
	}
	if got.ConfirmedAt == nil {
		t.Fatal("expected confirmed_at to be set")
	}

	byBlock, err := repo.GetByBlock(ctx, block.BlockID)
	if err != nil {
		t.Fatalf("GetByBlock: %v", err)
	}
	if len(byBlock) != 1 {
		t.Fatalf("expected 1 block IOC, got %d", len(byBlock))
	}

	existing, err := repo.GetExistingByBlock(ctx, block.BlockID)
	if err != nil {
		t.Fatalf("GetExistingByBlock: %v", err)
	}
	if _, ok := existing["ipv4:192.168.1.100"]; !ok {
		t.Fatal("expected ipv4:192.168.1.100 in existing set")
	}
}

func TestTimelineRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)
	c := createTestCase(t, db, user.UserID)

	repo := NewTimelineRepo(db)
	now := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	eventTime := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second).Format(time.RFC3339)

	entry := &models.TimelineEntry{
		EntryID:          "tl-1",
		CaseID:           c.CaseID,
		Timestamp:        eventTime,
		EventDescription: "Phishing email opened",
		CreatedAt:        now,
		UpdatedAt:        now,
		UserID:           user.UserID,
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, "tl-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.EventDescription != "Phishing email opened" {
		t.Fatalf("expected event description, got %q", got.EventDescription)
	}

	entries, err := repo.ListByCase(ctx, c.CaseID)
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got.EventDescription = "Lateral movement detected"
	got.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ = repo.GetByID(ctx, "tl-1")
	if got.EventDescription != "Lateral movement detected" {
		t.Fatalf("expected updated description, got %q", got.EventDescription)
	}

	if err := repo.Delete(ctx, "tl-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.GetByID(ctx, "tl-1")
	if err != models.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestAuditRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)
	c := createTestCase(t, db, user.UserID)

	repo := NewAuditRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	caseID := c.CaseID

	entry := &models.AuditLog{
		LogID:      "audit-1",
		CaseID:     &caseID,
		UserID:     user.UserID,
		Action:     models.AuditActionCreate,
		EntityType: "case",
		EntityID:   c.CaseID,
		Details:    json.RawMessage(`{"title":"Test Case"}`),
		CreatedAt:  now,
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	byCase, err := repo.ListByCase(ctx, c.CaseID)
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(byCase) != 1 {
		t.Fatalf("expected 1 entry by case, got %d", len(byCase))
	}
	if string(byCase[0].Details) != `{"title":"Test Case"}` {
		t.Fatalf("unexpected details: %s", byCase[0].Details)
	}

	byUser, err := repo.ListByUser(ctx, user.UserID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(byUser) != 1 {
		t.Fatalf("expected 1 entry by user, got %d", len(byUser))
	}
}

func TestTagRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := NewTagRepo(db)

	tag := &models.Tag{
		TagID: "tag-custom",
		Name:  "Custom Tag",
		Color: "#FF0000",
	}
	if err := repo.Create(ctx, tag); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, "tag-custom")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Custom Tag" {
		t.Fatalf("expected 'Custom Tag', got %q", got.Name)
	}

	if err := repo.Delete(ctx, "tag-custom"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, "tag-custom")
	if err != models.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestForeignKeyEnforcement(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	caseRepo := NewCaseRepo(db)
	now := time.Now().UTC().Truncate(time.Second)

	err := caseRepo.Create(ctx, &models.Case{
		CaseID:         "case-orphan",
		Title:          "Orphan Case",
		Classification: models.ClassificationUnclassified,
		CreatedBy:      "nonexistent-user",
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != models.ErrIntegrityViolation {
		t.Fatalf("expected ErrIntegrityViolation, got %v", err)
	}
}

func TestFormatParseTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	s := FormatTime(now)
	parsed, err := ParseTime(s)
	if err != nil {
		t.Fatalf("ParseTime: %v", err)
	}
	if !parsed.Equal(now) {
		t.Fatalf("expected %v, got %v", now, parsed)
	}
}

func TestOpenCreatesDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	path := filepath.Join(nested, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	db.Close()

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Fatal("expected directory to be created")
	}
}
