package services

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/database"
	"dfnotes-go/internal/models"
)

// setupCaseServiceTest wires CaseService and NoteService against the same real
// DB and authenticated session, so cases created through CaseService can be
// unlocked through NoteService exactly as the app does it.
func setupCaseServiceTest(t *testing.T) (*CaseService, *NoteService, models.CaseRepository, models.AuditLogRepository, *Session) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pub, priv, err := crypto.GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("GenerateSigningKeyPair: %v", err)
	}

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
	if err := database.NewUserRepo(db).Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	sess := NewSession()
	sess.SetAuthenticated(user, masterKey, priv)

	caseRepo := database.NewCaseRepo(db)
	auditRepo := database.NewAuditRepo(db)
	blockRepo := database.NewNoteBlockRepo(db)
	attachmentRepo := database.NewAttachmentRepo(db)

	caseSvc := NewCaseService(caseRepo, auditRepo, sess)
	noteSvc := NewNoteService(blockRepo, caseRepo, auditRepo, attachmentRepo, sess, noopTimer{})

	return caseSvc, noteSvc, caseRepo, auditRepo, sess
}

// unlockMethods returns the "method" detail of every UNLOCK audit entry for a case.
func unlockMethods(t *testing.T, auditRepo models.AuditLogRepository, caseID string) []string {
	t.Helper()
	entries, err := auditRepo.ListByCase(context.Background(), caseID)
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	var methods []string
	for _, e := range entries {
		if e.Action != models.AuditActionUnlock {
			continue
		}
		if e.EntityType != "case" || e.EntityID != caseID {
			t.Errorf("UNLOCK entry entity = %s/%s, want case/%s", e.EntityType, e.EntityID, caseID)
		}
		var d map[string]string
		if err := json.Unmarshal(e.Details, &d); err != nil {
			t.Fatalf("unmarshal UNLOCK details: %v", err)
		}
		methods = append(methods, d["method"])
	}
	return methods
}

func TestCreateCasePasswordlessRoundTrip(t *testing.T) {
	caseSvc, _, caseRepo, _, sess := setupCaseServiceTest(t)
	ctx := context.Background()

	resp, err := caseSvc.CreateCase(ctx, CreateCaseRequest{
		CaseNumber:     "CASE-NP-001",
		Title:          "Passwordless",
		NoCasePassword: true,
	})
	if err != nil {
		t.Fatalf("CreateCase: %v", err)
	}
	if !resp.NoCasePassword {
		t.Error("response NoCasePassword = false, want true")
	}

	c, err := caseRepo.GetByID(ctx, resp.CaseID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(c.Salt) != 0 {
		t.Errorf("salt = %x, want nil/empty", c.Salt)
	}
	if !c.NoCasePassword {
		t.Error("stored NoCasePassword = false, want true")
	}

	key, err := crypto.Decrypt(sess.DerivedKey(), c.EncryptedKey)
	if err != nil {
		t.Fatalf("unwrap case key: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("unwrapped key length = %d, want 32", len(key))
	}

	// List must carry the flag too, since the dashboard reads cases from it.
	cases, err := caseSvc.ListCases(ctx)
	if err != nil {
		t.Fatalf("ListCases: %v", err)
	}
	if len(cases) != 1 || !cases[0].NoCasePassword {
		t.Errorf("ListCases = %+v, want one case with NoCasePassword true", cases)
	}
}

func TestUnlockPasswordlessCase(t *testing.T) {
	caseSvc, noteSvc, _, auditRepo, _ := setupCaseServiceTest(t)
	ctx := context.Background()

	resp, err := caseSvc.CreateCase(ctx, CreateCaseRequest{
		CaseNumber:     "CASE-NP-002",
		Title:          "Passwordless",
		NoCasePassword: true,
	})
	if err != nil {
		t.Fatalf("CreateCase: %v", err)
	}

	if err := noteSvc.UnlockCase(ctx, UnlockCaseRequest{CaseID: resp.CaseID}); err != nil {
		t.Fatalf("UnlockCase with empty password: %v", err)
	}
	if _, err := noteSvc.getCaseKey(resp.CaseID); err != nil {
		t.Errorf("getCaseKey after unlock: %v", err)
	}

	methods := unlockMethods(t, auditRepo, resp.CaseID)
	if len(methods) != 1 || methods[0] != "no_password" {
		t.Errorf("UNLOCK methods = %v, want [no_password]", methods)
	}
}

func TestUnlockPasswordCaseCreatedThroughService(t *testing.T) {
	caseSvc, noteSvc, caseRepo, auditRepo, _ := setupCaseServiceTest(t)
	ctx := context.Background()

	resp, err := caseSvc.CreateCase(ctx, CreateCaseRequest{
		CaseNumber:   "CASE-PW-001",
		Title:        "With password",
		CasePassword: "case-password",
	})
	if err != nil {
		t.Fatalf("CreateCase: %v", err)
	}
	if resp.NoCasePassword {
		t.Error("response NoCasePassword = true, want false")
	}
	c, err := caseRepo.GetByID(ctx, resp.CaseID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if c.NoCasePassword || len(c.Salt) == 0 {
		t.Errorf("stored case: no_case_password=%v salt len=%d, want password required with salt", c.NoCasePassword, len(c.Salt))
	}

	if err := noteSvc.UnlockCase(ctx, UnlockCaseRequest{CaseID: resp.CaseID}); err == nil {
		t.Fatal("UnlockCase with empty password succeeded, want error")
	}
	if noteSvc.HasActiveCases() {
		t.Error("case unlocked after empty-password attempt")
	}
	if methods := unlockMethods(t, auditRepo, resp.CaseID); len(methods) != 0 {
		t.Errorf("UNLOCK entries after failed attempt = %v, want none", methods)
	}

	if err := noteSvc.UnlockCase(ctx, UnlockCaseRequest{
		CaseID:       resp.CaseID,
		CasePassword: "case-password",
	}); err != nil {
		t.Fatalf("UnlockCase with correct password: %v", err)
	}

	methods := unlockMethods(t, auditRepo, resp.CaseID)
	if len(methods) != 1 || methods[0] != "password" {
		t.Errorf("UNLOCK methods = %v, want [password]", methods)
	}
}

func TestCreateCaseNoPasswordWithPasswordRejected(t *testing.T) {
	caseSvc, _, _, _, _ := setupCaseServiceTest(t)

	_, err := caseSvc.CreateCase(context.Background(), CreateCaseRequest{
		CaseNumber:     "CASE-BAD-001",
		Title:          "Conflicting",
		CasePassword:   "should-not-be-here",
		NoCasePassword: true,
	})
	want := "case password must be empty when no case password is selected"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func TestCreateCaseOmittedFieldRequiresPassword(t *testing.T) {
	caseSvc, _, _, _, _ := setupCaseServiceTest(t)

	_, err := caseSvc.CreateCase(context.Background(), CreateCaseRequest{
		CaseNumber: "CASE-BAD-002",
		Title:      "No password, field omitted",
	})
	want := "case number, title, and case password are required"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

// TestRepoCreateOmittedFlagFailsClosed builds a Case directly, the way future
// code or a test fixture might, without setting NoCasePassword. The zero value
// must round-trip as "password required" and unlock must demand the password.
func TestRepoCreateOmittedFlagFailsClosed(t *testing.T) {
	_, noteSvc, caseRepo, _, sess := setupCaseServiceTest(t)
	ctx := context.Background()

	salt, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt: %v", err)
	}
	encKey, err := crypto.Encrypt(sess.DerivedKey(), crypto.DeriveKey("case-password", salt))
	if err != nil {
		t.Fatalf("wrap case key: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	c := &models.Case{
		CaseID:         "case-omitted-flag",
		CaseNumber:     "CASE-ZERO-001",
		Title:          "Flag omitted",
		Classification: models.ClassificationUnclassified,
		Salt:           salt,
		EncryptedKey:   encKey,
		CreatedBy:      sess.User().UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := caseRepo.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := caseRepo.GetByID(ctx, c.CaseID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.NoCasePassword {
		t.Fatal("NoCasePassword = true after creating with the field unset, want false")
	}

	if err := noteSvc.UnlockCase(ctx, UnlockCaseRequest{CaseID: c.CaseID}); err == nil {
		t.Fatal("UnlockCase with empty password succeeded, want error")
	}
	if err := noteSvc.UnlockCase(ctx, UnlockCaseRequest{CaseID: c.CaseID, CasePassword: "wrong"}); err == nil {
		t.Fatal("UnlockCase with wrong password succeeded, want error")
	}
	if noteSvc.HasActiveCases() {
		t.Fatal("case unlocked without the correct password")
	}
	if err := noteSvc.UnlockCase(ctx, UnlockCaseRequest{CaseID: c.CaseID, CasePassword: "case-password"}); err != nil {
		t.Fatalf("UnlockCase with correct password: %v", err)
	}
}
