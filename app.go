package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"dfnotes-go/internal/database"
	"dfnotes-go/internal/services"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx              context.Context
	db               *database.DB
	session          *services.Session
	identityService  *services.IdentityService
	caseService      *services.CaseService
	noteService      *services.NoteService
	evidenceService  *services.EvidenceService
	tagService       *services.TagService
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	dbPath := database.DefaultDBPath()
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database at %s: %v", dbPath, err)
	}
	a.db = db
	log.Printf("Database initialized at %s", dbPath)

	userRepo := database.NewUserRepo(db)
	caseRepo := database.NewCaseRepo(db)
	auditRepo := database.NewAuditRepo(db)
	noteBlockRepo := database.NewNoteBlockRepo(db)
	evidenceRepo := database.NewEvidenceRepo(db)
	tagRepo := database.NewTagRepo(db)
	attachmentRepo := database.NewAttachmentRepo(db)

	a.session = services.NewSession()
	a.identityService = services.NewIdentityService(userRepo, auditRepo, a.session)
	a.caseService = services.NewCaseService(caseRepo, auditRepo, a.session)
	a.noteService = services.NewNoteService(noteBlockRepo, caseRepo, auditRepo, attachmentRepo, a.session)
	a.evidenceService = services.NewEvidenceService(evidenceRepo, auditRepo, a.session)
	a.tagService = services.NewTagService(tagRepo, a.session)
}

func (a *App) shutdown(ctx context.Context) {
	if a.noteService != nil {
		a.noteService.ClearAll()
	}
	if a.session != nil {
		a.session.Clear()
	}
	if a.db != nil {
		a.db.Close()
	}
}

// CheckFirstLaunch returns true if no user exists (first launch).
func (a *App) CheckFirstLaunch() (bool, error) {
	_, err := a.identityService.GetFirstUser(a.ctx)
	if err != nil {
		if err.Error() == "not found" {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// GetUserInfo returns login screen info for the existing user.
func (a *App) GetUserInfo() (*services.LoginScreenInfo, error) {
	return a.identityService.GetLoginScreenInfo(a.ctx)
}

// SetupIdentity creates the initial user identity.
func (a *App) SetupIdentity(req services.SetupRequest) (*services.SetupResponse, error) {
	return a.identityService.Setup(a.ctx, req)
}

// ConfirmTOTPSetup verifies a TOTP code during setup.
func (a *App) ConfirmTOTPSetup(code string) (bool, error) {
	return a.identityService.ConfirmTOTPSetup(a.ctx, code)
}

// Login authenticates the user.
func (a *App) Login(req services.LoginRequest) (*services.LoginResponse, error) {
	return a.identityService.LoginFirstUser(a.ctx, req)
}

// CreateCase creates a new forensic case.
func (a *App) CreateCase(req services.CreateCaseRequest) (*services.CaseResponse, error) {
	return a.caseService.CreateCase(a.ctx, req)
}

// ListCases returns all cases.
func (a *App) ListCases() ([]services.CaseResponse, error) {
	return a.caseService.ListCases(a.ctx)
}

// GetCase returns a single case by ID.
func (a *App) GetCase(id string) (*services.CaseResponse, error) {
	return a.caseService.GetCase(a.ctx, id)
}

// UnlockCase unlocks a case with the case-specific password.
func (a *App) UnlockCase(req services.UnlockCaseRequest) error {
	return a.noteService.UnlockCase(a.ctx, req)
}

// LockCase locks a case, zeroing its key material.
func (a *App) LockCase(caseID string) error {
	return a.noteService.LockCase(a.ctx, caseID)
}

// CommitNote creates a new immutable note block in the chain.
func (a *App) CommitNote(req services.CommitNoteRequest) (*services.NoteBlockResponse, error) {
	return a.noteService.CommitNote(a.ctx, req)
}

// ListNotes returns all note blocks for a case, decrypted and verified.
func (a *App) ListNotes(caseID string) ([]services.NoteBlockResponse, error) {
	return a.noteService.ListNotes(a.ctx, caseID)
}

// ListEvidenceNotes returns all note blocks for a specific evidence item, decrypted and verified.
func (a *App) ListEvidenceNotes(caseID, evidenceItemID string) ([]services.NoteBlockResponse, error) {
	return a.noteService.ListEvidenceNotes(a.ctx, caseID, evidenceItemID)
}

// AddEvidence adds a new evidence item to a case.
func (a *App) AddEvidence(req services.AddEvidenceRequest) (*services.EvidenceResponse, error) {
	return a.evidenceService.AddEvidence(a.ctx, req)
}

// ListEvidence returns all evidence items for a case.
func (a *App) ListEvidence(caseID string) ([]services.EvidenceResponse, error) {
	return a.evidenceService.ListEvidence(a.ctx, caseID)
}

// GetEvidence returns a single evidence item by ID.
func (a *App) GetEvidence(evidenceItemID string) (*services.EvidenceResponse, error) {
	return a.evidenceService.GetEvidence(a.ctx, evidenceItemID)
}

// UpdateEvidenceStatus updates the status of an evidence item.
func (a *App) UpdateEvidenceStatus(req services.UpdateEvidenceStatusRequest) (*services.EvidenceResponse, error) {
	return a.evidenceService.UpdateEvidenceStatus(a.ctx, req)
}

// AddCustodyEntry adds a custody chain entry to an evidence item.
func (a *App) AddCustodyEntry(req services.AddCustodyEntryRequest) (*services.EvidenceResponse, error) {
	return a.evidenceService.AddCustodyEntry(a.ctx, req)
}

// ListTags returns all available tags.
func (a *App) ListTags() ([]services.TagResponse, error) {
	return a.tagService.ListTags(a.ctx)
}

// CreateTag creates a new custom tag.
func (a *App) CreateTag(req services.CreateTagRequest) (*services.TagResponse, error) {
	return a.tagService.CreateTag(a.ctx, req)
}

// DeleteTag deletes a tag.
func (a *App) DeleteTag(tagID string) error {
	return a.tagService.DeleteTag(a.ctx, tagID)
}

// TagBlock attaches a tag to a note block.
func (a *App) TagBlock(req services.TagBlockRequest) error {
	return a.tagService.TagBlock(a.ctx, req)
}

// UntagBlock detaches a tag from a note block.
func (a *App) UntagBlock(req services.TagBlockRequest) error {
	return a.tagService.UntagBlock(a.ctx, req)
}

// TagEvidence attaches a tag to an evidence item.
func (a *App) TagEvidence(req services.TagEvidenceRequest) error {
	return a.tagService.TagEvidence(a.ctx, req)
}

// UntagEvidence detaches a tag from an evidence item.
func (a *App) UntagEvidence(req services.TagEvidenceRequest) error {
	return a.tagService.UntagEvidence(a.ctx, req)
}

// SaveAttachment saves an encrypted attachment.
func (a *App) SaveAttachment(req services.SaveAttachmentRequest) (*services.AttachmentResponse, error) {
	return a.noteService.SaveAttachment(a.ctx, req)
}

// GetAttachment retrieves and decrypts an attachment.
func (a *App) GetAttachment(caseID, attachmentID string) (*services.AttachmentResponse, error) {
	return a.noteService.GetAttachment(a.ctx, caseID, attachmentID)
}

// AttachImage opens a native file dialog for images, reads the selected file,
// encrypts it, and saves it as an attachment. Returns the attachment response.
// Returns nil (no error) if the user cancels the dialog.
func (a *App) AttachImage(caseID string) (*services.AttachmentResponse, error) {
	selectedFile, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Image",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp;*.svg"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("file dialog: %w", err)
	}
	if selectedFile == "" {
		// User cancelled
		return nil, nil
	}

	data, err := os.ReadFile(selectedFile)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	filename := filepath.Base(selectedFile)
	ext := strings.ToLower(filepath.Ext(selectedFile))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "image/png"
	}

	return a.noteService.SaveAttachment(a.ctx, services.SaveAttachmentRequest{
		CaseID:      caseID,
		Filename:    filename,
		ContentType: contentType,
		Data:        base64.StdEncoding.EncodeToString(data),
	})
}
