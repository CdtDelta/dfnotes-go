package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dfnotes-go/internal/config"
	"dfnotes-go/internal/models"

	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo    models.TaskRepository
	noteService *NoteService
	session     *Session
}

func NewTaskService(taskRepo models.TaskRepository, noteService *NoteService, session *Session) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		noteService: noteService,
		session:     session,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, caseID string, title string, description string, evidenceItemID *string) (*models.Task, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	if caseID == "" {
		return nil, errors.New("case_id is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	task := models.Task{
		TaskID:         uuid.New().String(),
		CaseID:         caseID,
		EvidenceItemID: evidenceItemID,
		UserID:         s.session.User().UserID,
		Title:          title,
		Description:    description,
		Status:         models.TaskStatusOpen,
		CreatedAt:      now,
		UpdatedAt:      now,
		LinkedBlocks:   []models.LinkedBlock{},
	}

	if err := s.taskRepo.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, caseID string) ([]models.Task, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	tasks, err := s.taskRepo.ListTasks(ctx, caseID)
	if err != nil {
		return nil, err
	}
	s.populatePreviews(caseID, tasks)
	return tasks, nil
}

var validTaskStatuses = map[string]bool{
	"open": true, "in_progress": true, "blocked": true,
	"complete": true, "not_applicable": true,
}

func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	if !validTaskStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}
	task, err := s.taskRepo.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	task.Status = models.TaskStatus(status)
	task.UpdatedAt = now

	if task.Status == models.TaskStatusComplete {
		task.CompletedAt = &now
	} else {
		task.CompletedAt = nil
	}

	return s.taskRepo.UpdateTask(ctx, *task)
}

func (s *TaskService) UpdateTask(ctx context.Context, taskID string, title string, description string, evidenceItemID *string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	if title == "" {
		return errors.New("title is required")
	}
	task, err := s.taskRepo.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	task.Title = title
	task.Description = description
	task.EvidenceItemID = evidenceItemID
	task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return s.taskRepo.UpdateTask(ctx, *task)
}

func (s *TaskService) DeleteTask(ctx context.Context, taskID string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	return s.taskRepo.DeleteTask(ctx, taskID)
}

func (s *TaskService) LinkNoteToTask(ctx context.Context, taskID string, blockID string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	return s.taskRepo.LinkNoteBlock(ctx, taskID, blockID)
}

func (s *TaskService) UnlinkNoteFromTask(ctx context.Context, taskID string, blockID string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}
	return s.taskRepo.UnlinkNoteBlock(ctx, taskID, blockID)
}

func (s *TaskService) GetLinkedTasks(ctx context.Context, blockID string) ([]models.Task, error) {
	if !s.session.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}
	return s.taskRepo.GetLinkedTasks(ctx, blockID)
}

// ApplyTemplate reads the named template and creates all its tasks in the case.
func (s *TaskService) ApplyTemplate(ctx context.Context, caseID string, templateName string, evidenceItemID *string) error {
	if !s.session.IsAuthenticated() {
		return errors.New("not authenticated")
	}

	tf, err := config.LoadTemplates()
	if err != nil {
		return err
	}

	var found *config.TaskTemplate
	for i := range tf.Templates {
		if tf.Templates[i].Name == templateName {
			found = &tf.Templates[i]
			break
		}
	}
	if found == nil {
		return errors.New("template not found: " + templateName)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	userID := s.session.User().UserID
	for _, tt := range found.Tasks {
		tname := templateName
		task := models.Task{
			TaskID:         uuid.New().String(),
			CaseID:         caseID,
			EvidenceItemID: evidenceItemID,
			UserID:         userID,
			Title:          tt.Title,
			Description:    tt.Description,
			Status:         models.TaskStatusOpen,
			TemplateName:   &tname,
			CreatedAt:      now,
			UpdatedAt:      now,
			LinkedBlocks:   []models.LinkedBlock{},
		}
		if err := s.taskRepo.CreateTask(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// populatePreviews decrypts linked block content and fills in Preview fields.
// Failures are silently skipped (locked case, etc.) so the task list still renders.
func (s *TaskService) populatePreviews(caseID string, tasks []models.Task) {
	for i := range tasks {
		for j := range tasks[i].LinkedBlocks {
			if len(tasks[i].LinkedBlocks[j].EncryptedBody) == 0 {
				continue
			}
			content, err := s.noteService.DecryptBlockContent(caseID, tasks[i].LinkedBlocks[j].EncryptedBody)
			if err != nil {
				continue
			}
			tasks[i].LinkedBlocks[j].Preview = truncateRunes(content, 100)
			tasks[i].LinkedBlocks[j].EncryptedBody = nil
		}
	}
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
