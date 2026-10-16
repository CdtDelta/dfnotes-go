package services

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dfnotes-go/internal/config"
	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/database"
	"dfnotes-go/internal/models"
)

const taskTestCaseID = "task-case-1"

// setupTaskServiceTest wires a real DB with one user and one case, and points
// the templates file at a temp config dir.
func setupTaskServiceTest(t *testing.T) (*TaskService, *database.DB) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pub, priv, err := crypto.GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("GenerateSigningKeyPair: %v", err)
	}
	salt, _ := crypto.GenerateSalt()
	now := time.Now().UTC().Truncate(time.Second)
	user := &models.UserIdentity{
		UserID:              "task-user-1",
		Name:                "Test Examiner",
		Organization:        "Test Org",
		PublicKey:           pub,
		EncryptedPrivateKey: []byte("unused"),
		Salt:                salt,
		CreatedAt:           now,
	}
	if err := database.NewUserRepo(db).Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	c := &models.Case{
		CaseID:         taskTestCaseID,
		CaseNumber:     "CASE-TASK-001",
		Title:          "Task Case",
		Classification: models.ClassificationUnclassified,
		Salt:           salt,
		EncryptedKey:   []byte("unused"),
		CreatedBy:      user.UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := database.NewCaseRepo(db).Create(context.Background(), c); err != nil {
		t.Fatalf("create case: %v", err)
	}

	sess := NewSession()
	sess.SetAuthenticated(user, make([]byte, 32), priv)
	return NewTaskService(database.NewTaskRepo(db), nil, sess), db
}

func saveTestTemplates(t *testing.T, templates ...config.TaskTemplate) {
	t.Helper()
	if err := config.SaveTemplates(&config.TemplatesFile{Templates: templates}); err != nil {
		t.Fatalf("SaveTemplates: %v", err)
	}
}

func countTasks(t *testing.T, db *database.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&n); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	return n
}

func TestApplyTemplateSubstitutesPlaceholders(t *testing.T) {
	svc, _ := setupTaskServiceTest(t)
	ctx := context.Background()
	saveTestTemplates(t, config.TaskTemplate{Name: "Evidence {{HOST}}", Tasks: []config.TemplateTask{
		{Title: "[{{HOST}}] Acquire", Description: "Image {{ HOST }} for {{TICKET}}"},
		{Title: "[{{HOST}}] Hash {X} [Y]", Description: ""},
	}})

	err := svc.ApplyTemplate(ctx, taskTestCaseID, "Evidence {{HOST}}", nil,
		map[string]string{"HOST": "  WS-042 ", "TICKET": "{{HOST}}"})
	if err != nil {
		t.Fatalf("ApplyTemplate: %v", err)
	}

	tasks, err := svc.taskRepo.ListTasks(ctx, taskTestCaseID)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(tasks))
	}
	byTitle := map[string]models.Task{}
	for _, task := range tasks {
		byTitle[task.Title] = task
	}
	acquire, ok := byTitle["[WS-042] Acquire"]
	if !ok {
		t.Fatalf("missing substituted title; got %v", byTitle)
	}
	if acquire.Description != "Image WS-042 for {{HOST}}" {
		t.Errorf("description = %q", acquire.Description)
	}
	if _, ok := byTitle["[WS-042] Hash {X} [Y]"]; !ok {
		t.Errorf("missing second task; got %v", byTitle)
	}
	for _, task := range tasks {
		if task.TemplateName == nil || *task.TemplateName != "Evidence {{HOST}}" {
			t.Errorf("template_name should be stored verbatim, got %v", task.TemplateName)
		}
		if task.CreatedAt != tasks[0].CreatedAt {
			t.Errorf("tasks should share a timestamp: %s vs %s", task.CreatedAt, tasks[0].CreatedAt)
		}
	}
}

func TestApplyTemplateNoPlaceholdersNilValues(t *testing.T) {
	svc, db := setupTaskServiceTest(t)
	saveTestTemplates(t, config.TaskTemplate{Name: "Triage", Tasks: []config.TemplateTask{
		{Title: "Interview"}, {Title: "Collect"},
	}})
	if err := svc.ApplyTemplate(context.Background(), taskTestCaseID, "Triage", nil, nil); err != nil {
		t.Fatalf("ApplyTemplate: %v", err)
	}
	if n := countTasks(t, db); n != 2 {
		t.Errorf("got %d tasks, want 2", n)
	}
}

func TestApplyTemplateRejectsBadInputWithoutWriting(t *testing.T) {
	svc, db := setupTaskServiceTest(t)
	saveTestTemplates(t, config.TaskTemplate{Name: "Evidence", Tasks: []config.TemplateTask{
		{Title: "[{{HOST}}] Acquire"},
	}})
	ctx := context.Background()

	cases := []struct {
		name, caseID, template string
		values                 map[string]string
	}{
		{"empty case id", "", "Evidence", map[string]string{"HOST": "a"}},
		{"unknown template", taskTestCaseID, "evidence", map[string]string{"HOST": "a"}},
		{"missing value", taskTestCaseID, "Evidence", nil},
		{"blank value", taskTestCaseID, "Evidence", map[string]string{"HOST": "  "}},
		{"multiline value", taskTestCaseID, "Evidence", map[string]string{"HOST": "a\nb"}},
		{"extra key", taskTestCaseID, "Evidence", map[string]string{"HOST": "a", "OTHER": "b"}},
	}
	for _, c := range cases {
		if err := svc.ApplyTemplate(ctx, c.caseID, c.template, nil, c.values); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
	if n := countTasks(t, db); n != 0 {
		t.Errorf("got %d tasks after rejected applies, want 0", n)
	}
}

// TestApplyTemplateMidBatchFailureLeavesNoTasks forces the second of three
// inserts to fail and proves the first one is rolled back.
func TestApplyTemplateMidBatchFailureLeavesNoTasks(t *testing.T) {
	svc, db := setupTaskServiceTest(t)
	saveTestTemplates(t, config.TaskTemplate{Name: "Evidence", Tasks: []config.TemplateTask{
		{Title: "[{{HOST}}] first"},
		{Title: "[{{HOST}}] boom"},
		{Title: "[{{HOST}}] third"},
	}})
	if _, err := db.Exec(`CREATE TRIGGER test_fail_boom BEFORE INSERT ON tasks
		WHEN NEW.title LIKE '%boom' BEGIN SELECT RAISE(ABORT, 'forced failure'); END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	err := svc.ApplyTemplate(context.Background(), taskTestCaseID, "Evidence", nil,
		map[string]string{"HOST": "WS-042"})
	if err == nil || !strings.Contains(err.Error(), "forced failure") || !strings.Contains(err.Error(), "task 2 of 3") {
		t.Fatalf("expected forced failure on task 2 of 3, got %v", err)
	}
	if n := countTasks(t, db); n != 0 {
		t.Errorf("got %d tasks after mid-batch failure, want 0", n)
	}
}
