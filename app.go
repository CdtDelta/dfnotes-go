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

	"dfnotes-go/internal/backup"
	"dfnotes-go/internal/config"
	"dfnotes-go/internal/database"
	"dfnotes-go/internal/export"
	"dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
	"dfnotes-go/internal/services"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx             context.Context
	cfg             *config.Config
	db              *database.DB
	dbMissing       bool // cfg.DatabasePath set but file absent
	session         *services.Session
	identityService *services.IdentityService
	caseService     *services.CaseService
	noteService     *services.NoteService
	evidenceService *services.EvidenceService
	tagService      *services.TagService
	iocService      *ioc.IOCService
	timelineService *services.TimelineService
	taskService     *services.TaskService
	backupScheduler *backup.Scheduler
	noteBlockRepo   models.NoteBlockRepository
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	a.cfg = cfg

	if cfg.DatabasePath == "" {
		// First launch: no path configured yet. Wizard calls InitializeDatabase.
		return
	}

	if _, statErr := os.Stat(cfg.DatabasePath); os.IsNotExist(statErr) {
		// Path is configured but file is gone -- not a first launch, not openable.
		// Frontend detects this via CheckFirstLaunch returning an error.
		a.dbMissing = true
		return
	}

	if err := a.openDatabase(cfg.DatabasePath); err != nil {
		log.Fatalf("failed to open database at %s: %v", cfg.DatabasePath, err)
	}
}

func (a *App) openDatabase(path string) error {
	db, err := database.Open(path)
	if err != nil {
		return err
	}
	a.db = db
	log.Printf("Database initialized at %s", path)

	userRepo := database.NewUserRepo(db)
	caseRepo := database.NewCaseRepo(db)
	auditRepo := database.NewAuditRepo(db)
	noteBlockRepo := database.NewNoteBlockRepo(db)
	evidenceRepo := database.NewEvidenceRepo(db)
	tagRepo := database.NewTagRepo(db)
	attachmentRepo := database.NewAttachmentRepo(db)
	iocRepo := database.NewIOCRepo(db)
	timelineRepo := database.NewTimelineRepo(db)
	taskRepo := database.NewTaskRepo(db)

	a.session = services.NewSession()
	a.identityService = services.NewIdentityService(userRepo, auditRepo, a.session)
	a.caseService = services.NewCaseService(caseRepo, auditRepo, a.session)
	a.noteService = services.NewNoteService(noteBlockRepo, caseRepo, auditRepo, attachmentRepo, a.session)
	a.evidenceService = services.NewEvidenceService(evidenceRepo, auditRepo, a.session)
	a.tagService = services.NewTagService(tagRepo, a.session)
	a.iocService = ioc.NewIOCService(iocRepo)
	a.timelineService = services.NewTimelineService(timelineRepo, a.session)
	a.taskService = services.NewTaskService(taskRepo, a.noteService, a.session)
	a.noteBlockRepo = noteBlockRepo
	return nil
}

func (a *App) shutdown(ctx context.Context) {
	if a.backupScheduler != nil {
		a.backupScheduler.Stop()
	}
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

// CheckFirstLaunch returns true if the DB is not yet initialized (no path set),
// or (false, error) if the configured DB file is missing, or false for normal auth flow.
func (a *App) CheckFirstLaunch() (bool, error) {
	if a.dbMissing {
		return false, fmt.Errorf("database file not found at %s -- the file may have been moved or deleted", a.cfg.DatabasePath)
	}
	if a.db == nil {
		return true, nil
	}
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

// SetupIdentity creates the initial user identity and starts the backup scheduler.
func (a *App) SetupIdentity(req services.SetupRequest) (*services.SetupResponse, error) {
	resp, err := a.identityService.Setup(a.ctx, req)
	if err != nil {
		return nil, err
	}
	a.startBackupScheduler()
	return resp, nil
}

// ConfirmTOTPSetup verifies a TOTP code during setup.
func (a *App) ConfirmTOTPSetup(code string) (bool, error) {
	return a.identityService.ConfirmTOTPSetup(a.ctx, code)
}

// Login authenticates the user and starts the backup scheduler.
func (a *App) Login(req services.LoginRequest) (*services.LoginResponse, error) {
	resp, err := a.identityService.LoginFirstUser(a.ctx, req)
	if err != nil {
		return nil, err
	}
	a.startBackupScheduler()
	return resp, nil
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

func (a *App) startBackupScheduler() {
	if a.backupScheduler != nil {
		a.backupScheduler.Stop()
	}
	dbPath := a.cfg.DatabasePath
	if dbPath == "" {
		dbPath = database.DefaultDBPath()
	}
	a.backupScheduler = backup.NewScheduler(a.cfg, dbPath, func(err error) {
		wailsruntime.EventsEmit(a.ctx, "backup:failed", err.Error())
	})
	if a.cfg.BackupEnabled {
		a.backupScheduler.Start()
	}
}

// CommitNote creates a new immutable note block in the chain.
func (a *App) CommitNote(req services.CommitNoteRequest) (*services.NoteBlockResponse, error) {
	resp, err := a.noteService.CommitNote(a.ctx, req)
	if err != nil {
		return nil, err
	}
	// Best-effort IOC detection -- failure does not fail the commit.
	var evID *string
	if req.EvidenceItemID != "" {
		evID = &req.EvidenceItemID
	}
	if detectErr := a.iocService.DetectAndStore(
		a.ctx, req.CaseID, resp.BlockID, evID, req.Content, resp.AuthorID,
	); detectErr != nil {
		log.Printf("ioc detection failed for block %s: %v", resp.BlockID, detectErr)
	}
	return resp, nil
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

// GetCaseIOCs returns IOC entries for a case.
// When includeAll is false, false_positive records are excluded (default for summary page).
func (a *App) GetCaseIOCs(caseID string, includeAll bool) ([]ioc.IOCEntry, error) {
	return a.iocService.GetCaseIOCs(a.ctx, caseID, includeAll)
}

// UpdateIOCStatus changes the status of an IOC (detected, confirmed, false_positive).
func (a *App) UpdateIOCStatus(iocID string, status string) error {
	return a.iocService.UpdateIOCStatus(a.ctx, iocID, status)
}

// GetBlockIOCs returns all IOC entries for a specific committed block.
func (a *App) GetBlockIOCs(blockID string) ([]ioc.IOCEntry, error) {
	return a.iocService.GetBlockIOCs(a.ctx, blockID)
}

// GetTimelineEntries returns all timeline entries for a case, sorted by timestamp ASC.
func (a *App) GetTimelineEntries(caseID string) ([]models.TimelineEntry, error) {
	return a.timelineService.GetTimelineEntries(a.ctx, caseID)
}

// CreateTimelineEntry adds a new timeline entry to a case.
func (a *App) CreateTimelineEntry(req models.CreateTimelineEntryRequest) (*models.TimelineEntry, error) {
	return a.timelineService.CreateTimelineEntry(a.ctx, req)
}

// UpdateTimelineEntry updates timestamp, timezone, description, and notes on an existing entry.
func (a *App) UpdateTimelineEntry(req models.UpdateTimelineEntryRequest) (*models.TimelineEntry, error) {
	return a.timelineService.UpdateTimelineEntry(a.ctx, req)
}

// DeleteTimelineEntry hard-deletes a timeline entry. Timeline entries are analyst
// working notes and are not part of the hash chain.
func (a *App) DeleteTimelineEntry(entryID string) error {
	return a.timelineService.DeleteTimelineEntry(a.ctx, entryID)
}

// GetDefaultDBPath returns the OS-appropriate default database path.
func (a *App) GetDefaultDBPath() string {
	return database.DefaultDBPath()
}

// ChooseDBSavePath opens a native save-file dialog filtered to .db files.
func (a *App) ChooseDBSavePath() (string, error) {
	return wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Choose Database Location",
		DefaultFilename: "dfnotes.db",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "SQLite Database", Pattern: "*.db"},
		},
	})
}

// InitializeDatabase creates a new database at path, saves the path to config,
// and initializes all services. Returns a "FILE_EXISTS:..." error if a file already
// exists at path so the wizard can offer to open it instead.
func (a *App) InitializeDatabase(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("FILE_EXISTS: a database file already exists at %s", path)
	}
	if err := a.openDatabase(path); err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	a.cfg.DatabasePath = path
	return config.Save(a.cfg)
}

// MoveDatabase copies the current database to newPath, verifies integrity via
// SHA-256, updates config, and removes the original. The case must be locked.
func (a *App) MoveDatabase(newPath string) error {
	if a.noteService != nil && a.noteService.HasActiveCases() {
		return fmt.Errorf("lock the active case before moving the database")
	}
	if a.db == nil {
		return fmt.Errorf("no database is open")
	}
	currentPath := a.cfg.DatabasePath
	if currentPath == "" {
		currentPath = database.DefaultDBPath()
	}

	// Close before copy: SQLite checkpoints the WAL into the main file on close.
	a.db.Close()
	a.db = nil

	if err := database.CopyAndVerify(currentPath, newPath); err != nil {
		log.Printf("Recovery: re-opening original database at %s", currentPath)
		if reopenErr := a.openDatabase(currentPath); reopenErr != nil {
			log.Printf("critical: failed to reopen original database after failed move: %v", reopenErr)
		}
		return fmt.Errorf("copy database: %w", err)
	}

	if err := os.Remove(currentPath); err != nil {
		log.Printf("warning: could not remove original database at %s: %v", currentPath, err)
	}

	a.cfg.DatabasePath = newPath
	if err := config.Save(a.cfg); err != nil {
		log.Printf("critical: config save failed after move to %s: %v", newPath, err)
	}

	if err := a.openDatabase(newPath); err != nil {
		return fmt.Errorf("open database at new path: %w", err)
	}
	a.dbMissing = false
	return nil
}

// PointDatabase validates that newPath is a dfnotes-go database and updates
// config to use it. The case must be locked.
func (a *App) PointDatabase(newPath string) error {
	if a.noteService != nil && a.noteService.HasActiveCases() {
		return fmt.Errorf("lock the active case before changing the database")
	}
	if err := database.ValidateSchema(newPath); err != nil {
		return fmt.Errorf("invalid database: %w", err)
	}
	if a.db != nil {
		a.db.Close()
		a.db = nil
	}
	if err := a.openDatabase(newPath); err != nil {
		return fmt.Errorf("open database at new path: %w", err)
	}
	a.dbMissing = false
	a.cfg.DatabasePath = newPath
	return config.Save(a.cfg)
}

// handleExportCaseMenu is called from the native menu callback. It shows a
// native info dialog when no case is unlocked, otherwise signals the frontend.
func (a *App) handleExportCaseMenu() {
	if a.noteService == nil || !a.noteService.HasActiveCases() {
		wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
			Type:    wailsruntime.InfoDialog,
			Title:   "Export Case",
			Message: "No case is currently open. Open and unlock a case before exporting.",
		})
		return
	}
	wailsruntime.EventsEmit(a.ctx, "menu:export-case")
}

// ChooseExportSavePath opens a native save-file dialog for the export archive.
func (a *App) ChooseExportSavePath(defaultName string) (string, error) {
	defaultDir := filepath.Dir(a.cfg.DatabasePath)
	return wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:            "Save Export Archive",
		DefaultDirectory: defaultDir,
		DefaultFilename:  defaultName,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "7z Archive", Pattern: "*.7z"},
		},
	})
}

// ChooseDBOpenPath opens a native file picker filtered to .db files.
func (a *App) ChooseDBOpenPath() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Database File",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "SQLite Database", Pattern: "*.db"},
		},
	})
}

// GetBackupStatus returns the current backup status, reading last-run info
// from config (persisted) so the values survive app restarts.
func (a *App) GetBackupStatus() backup.Status {
	st := backup.Status{LastBackupStatus: "never"}
	if a.cfg != nil && a.cfg.LastBackupStatus != "" {
		st.LastBackupTime = a.cfg.LastBackupAt
		st.LastBackupStatus = a.cfg.LastBackupStatus
	}
	if a.backupScheduler != nil {
		st.IsRunning = a.backupScheduler.Status().IsRunning
	}
	return st
}

// TriggerBackupNow runs a backup immediately.
func (a *App) TriggerBackupNow() error {
	log.Printf("TriggerBackupNow called")
	if a.backupScheduler == nil {
		return fmt.Errorf("backup scheduler not initialized")
	}
	return a.backupScheduler.RunNow()
}

// GetConfig returns the current application configuration.
func (a *App) GetConfig() config.Config {
	return *a.cfg
}

// GetDBPath returns the path of the currently open database file.
func (a *App) GetDBPath() string {
	if a.cfg.DatabasePath != "" {
		return a.cfg.DatabasePath
	}
	return database.DefaultDBPath()
}

// ChooseDirectory opens a native directory picker and returns the selected path.
func (a *App) ChooseDirectory() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Directory",
	})
}

// SaveConfig persists the updated configuration to disk.
func (a *App) SaveConfig(cfg config.Config) error {
	log.Printf("Config saved: backup_dest_path=%q", cfg.BackupDestPath)
	// Preserve scheduler-owned fields; the frontend form does not manage them.
	cfg.LastBackupAt = a.cfg.LastBackupAt
	cfg.LastBackupStatus = a.cfg.LastBackupStatus
	if err := config.Save(&cfg); err != nil {
		return err
	}
	*a.cfg = cfg
	return nil
}

// ExportCase exports the full case to a password-protected 7z archive at archivePath.
// Runs in a background goroutine; progress is reported via Wails events:
//
//	export:progress {"stage": "...", "percent": N}
//	export:complete {"path": "..."}
//	export:error    {"message": "..."}
func (a *App) ExportCase(caseID string, archivePassword string, archivePath string) error {
	if len(archivePassword) < 8 {
		return fmt.Errorf("archive password must be at least 8 characters")
	}
	if archivePath == "" {
		return fmt.Errorf("archive path must not be empty")
	}
	if a.noteBlockRepo == nil || a.noteService == nil {
		return fmt.Errorf("database not initialized")
	}

	// Snapshot all fields accessed inside the goroutine to avoid data races
	// with concurrent MoveDatabase/PointDatabase calls on the main goroutine.
	ctx             := a.ctx
	caseService     := a.caseService
	evidenceService := a.evidenceService
	noteService     := a.noteService
	noteBlockRepo   := a.noteBlockRepo
	iocService      := a.iocService
	timelineService := a.timelineService
	taskService     := a.taskService
	identityService := a.identityService
	dbPath          := a.cfg.DatabasePath

	go func() {
		emitErr := func(msg string) {
			wailsruntime.EventsEmit(ctx, "export:error", map[string]string{"message": msg})
		}

		caseData, err := caseService.GetCase(ctx, caseID)
		if err != nil {
			emitErr(fmt.Sprintf("get case: %v", err))
			return
		}

		evidenceItems, err := evidenceService.ListEvidence(ctx, caseID)
		if err != nil {
			emitErr(fmt.Sprintf("list evidence: %v", err))
			return
		}

		masterBlocks, err := noteService.ListNotes(ctx, caseID)
		if err != nil {
			emitErr(fmt.Sprintf("list notes: %v", err))
			return
		}

		evidenceBlockMap := make(map[string][]services.NoteBlockResponse, len(evidenceItems))
		for _, item := range evidenceItems {
			blocks, err := noteService.ListEvidenceNotes(ctx, caseID, item.EvidenceItemID)
			if err != nil {
				emitErr(fmt.Sprintf("list evidence notes for %s: %v", item.EvidenceItemID, err))
				return
			}
			evidenceBlockMap[item.EvidenceItemID] = blocks
		}

		rawBlocks, err := noteBlockRepo.ListByCase(ctx, caseID)
		if err != nil {
			emitErr(fmt.Sprintf("list raw blocks: %v", err))
			return
		}

		iocEntries, err := iocService.GetCaseIOCs(ctx, caseID, true)
		if err != nil {
			emitErr(fmt.Sprintf("get IOCs: %v", err))
			return
		}

		timelineEntries, err := timelineService.GetTimelineEntries(ctx, caseID)
		if err != nil {
			emitErr(fmt.Sprintf("get timeline: %v", err))
			return
		}

		taskList, err := taskService.ListTasks(ctx, caseID)
		if err != nil {
			emitErr(fmt.Sprintf("get tasks: %v", err))
			return
		}

		user, err := identityService.GetFirstUser(ctx)
		if err != nil {
			emitErr(fmt.Sprintf("get user: %v", err))
			return
		}

		req := export.ExportRequest{
			CaseID:          caseID,
			ArchivePassword: archivePassword,
			DBPath:          dbPath,
			ArchivePath:     archivePath,
			ExaminerName:    user.Name,
			ExaminerPubKey:  user.PublicKey,
		}

		archivePath, err := export.ExportCase(
			ctx, req, caseData, evidenceItems, masterBlocks, evidenceBlockMap,
			rawBlocks, iocEntries, timelineEntries, taskList,
			func(stage string, percent int) {
				wailsruntime.EventsEmit(ctx, "export:progress", map[string]any{
					"stage":   stage,
					"percent": percent,
				})
			},
		)
		if err != nil {
			emitErr(err.Error())
			return
		}

		wailsruntime.EventsEmit(ctx, "export:complete", map[string]string{"path": archivePath})
	}()

	return nil
}

// CreateTask creates a new task in a case.
func (a *App) CreateTask(caseID string, title string, description string, evidenceItemID *string) (*models.Task, error) {
	return a.taskService.CreateTask(a.ctx, caseID, title, description, evidenceItemID)
}

// ListTasks returns all tasks for a case with linked block previews.
func (a *App) ListTasks(caseID string) ([]models.Task, error) {
	return a.taskService.ListTasks(a.ctx, caseID)
}

// UpdateTaskStatus changes the status of a task and manages completed_at accordingly.
func (a *App) UpdateTaskStatus(taskID string, status string) error {
	return a.taskService.UpdateTaskStatus(a.ctx, taskID, status)
}

// UpdateTask updates the title, description, and evidence item assignment of a task.
func (a *App) UpdateTask(taskID string, title string, description string, evidenceItemID *string) error {
	return a.taskService.UpdateTask(a.ctx, taskID, title, description, evidenceItemID)
}

// DeleteTask permanently removes a task and its note links.
func (a *App) DeleteTask(taskID string) error {
	return a.taskService.DeleteTask(a.ctx, taskID)
}

// LinkNoteToTask creates a link between a task and a committed note block.
func (a *App) LinkNoteToTask(taskID string, blockID string) error {
	return a.taskService.LinkNoteToTask(a.ctx, taskID, blockID)
}

// UnlinkNoteFromTask removes a link between a task and a note block.
func (a *App) UnlinkNoteFromTask(taskID string, blockID string) error {
	return a.taskService.UnlinkNoteFromTask(a.ctx, taskID, blockID)
}

// GetLinkedTasks returns all tasks linked to a given note block.
func (a *App) GetLinkedTasks(blockID string) ([]models.Task, error) {
	return a.taskService.GetLinkedTasks(a.ctx, blockID)
}

// LoadTemplates returns all task templates from the templates config file.
func (a *App) LoadTemplates() ([]config.TaskTemplate, error) {
	tf, err := config.LoadTemplates()
	if err != nil {
		return nil, err
	}
	return tf.Templates, nil
}

// SaveTemplates overwrites the task templates config file with the provided list.
func (a *App) SaveTemplates(templates []config.TaskTemplate) error {
	return config.SaveTemplates(&config.TemplatesFile{Templates: templates})
}

// ApplyTemplate instantiates all tasks from the named template into the case.
func (a *App) ApplyTemplate(caseID string, templateName string, evidenceItemID *string) error {
	return a.taskService.ApplyTemplate(a.ctx, caseID, templateName, evidenceItemID)
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

