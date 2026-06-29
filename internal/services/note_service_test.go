package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/database"
	"dfnotes-go/internal/models"
	"dfnotes-go/internal/verify"
)

// noopTimer satisfies timer.Service for tests without any goroutines.
type noopTimer struct{}

func (noopTimer) Start()          {}
func (noopTimer) Stop()           {}
func (noopTimer) ResetFull()      {}
func (noopTimer) ResetPartial()   {}
func (noopTimer) Snooze(int)      {}
func (noopTimer) Pause()          {}
func (noopTimer) Resume()         {}
func (noopTimer) IsPaused() bool  { return false }

// setupNoteServiceTest wires a real DB with real repos and a real session,
// using production crypto keys. Returns the service and the raw block repo so
// callers can read back committed blocks for verification.
func setupNoteServiceTest(t *testing.T) (*NoteService, models.NoteBlockRepository, models.CaseRepository, *Session, []byte) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Generate real ed25519 keys.
	pub, priv, err := crypto.GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("GenerateSigningKeyPair: %v", err)
	}

	// Create a user in the DB.
	salt, _ := crypto.GenerateSalt()
	masterKey := crypto.DeriveKey("master-password", salt)
	encPriv, err := crypto.EncryptPrivateKey(masterKey, priv)
	if err != nil {
		t.Fatalf("EncryptPrivateKey: %v", err)
	}
	user := &models.UserIdentity{
		UserID:              "test-user-1",
		Name:                "Test Examiner",
		Organization:        "Test Org",
		PublicKey:           pub,
		EncryptedPrivateKey: encPriv,
		Salt:                salt,
		CreatedAt:           time.Now().UTC().Truncate(time.Second),
	}
	userRepo := database.NewUserRepo(db)
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Establish a real session.
	sess := NewSession()
	sess.SetAuthenticated(user, masterKey, priv)

	// Build a case with a real case key.
	caseSalt, _ := crypto.GenerateSalt()
	caseKey := crypto.DeriveKey("case-password", caseSalt)
	encCaseKey, err := crypto.Encrypt(masterKey, caseKey)
	if err != nil {
		t.Fatalf("Encrypt case key: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	c := &models.Case{
		CaseID:         "test-case-1",
		CaseNumber:     "CASE-TEST-001",
		Title:          "Test Case",
		Classification: models.ClassificationUnclassified,
		Salt:           caseSalt,
		EncryptedKey:   encCaseKey,
		CreatedBy:      user.UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	caseRepo := database.NewCaseRepo(db)
	if err := caseRepo.Create(context.Background(), c); err != nil {
		t.Fatalf("create case: %v", err)
	}

	blockRepo := database.NewNoteBlockRepo(db)
	auditRepo := database.NewAuditRepo(db)
	attachmentRepo := database.NewAttachmentRepo(db)

	svc := NewNoteService(blockRepo, caseRepo, auditRepo, attachmentRepo, sess, noopTimer{})

	// Unlock the case so CommitNote can encrypt.
	if err := svc.UnlockCase(context.Background(), UnlockCaseRequest{
		CaseID:       c.CaseID,
		CasePassword: "case-password",
	}); err != nil {
		t.Fatalf("UnlockCase: %v", err)
	}

	return svc, blockRepo, caseRepo, sess, caseKey
}

// TestRoundTripSignatureValid is the canonical tripwire for timestamp
// canonicalization. It commits a block, reads it back from the DB via
// ListByCaseChainOrder, and asserts that verify.BlockIntegrity returns
// SignatureValid == true. If the stored created_at does not round-trip to the
// same truncated-second form used at commit time, SignatureValid will be false
// and this test will catch it before any real case data is affected.
func TestRoundTripSignatureValid(t *testing.T) {
	svc, blockRepo, _, sess, caseKey := setupNoteServiceTest(t)
	ctx := context.Background()
	caseID := "test-case-1"

	resp, err := svc.CommitNote(ctx, CommitNoteRequest{
		CaseID:  caseID,
		Content: "round-trip test note",
	})
	if err != nil {
		t.Fatalf("CommitNote: %v", err)
	}
	_ = resp

	blocks, err := blockRepo.ListByCaseChainOrder(ctx, caseID)
	if err != nil {
		t.Fatalf("ListByCaseChainOrder: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	user := sess.User()
	decrypt := func(ct []byte) ([]byte, error) { return crypto.Decrypt(caseKey, ct) }
	check, _ := verify.BlockIntegrity(b, user.PublicKey, decrypt)

	if !check.Decrypted {
		t.Error("block did not decrypt; check case key setup")
	}
	if !check.HashValid {
		t.Error("content hash mismatch after round-trip")
	}
	if !check.SignatureValid {
		t.Errorf("SignatureValid is false after round-trip; created_at stored=%v, payload ts=%v",
			b.CreatedAt,
			b.CreatedAt.UTC().Truncate(time.Second),
		)
	}
}
