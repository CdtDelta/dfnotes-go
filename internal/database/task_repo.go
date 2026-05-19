package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"dfnotes-go/internal/models"
)

type TaskRepo struct {
	db *DB
}

func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) CreateTask(ctx context.Context, task models.Task) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks
		    (task_id, case_id, evidence_item_id, user_id, title, description,
		     status, template_name, created_at, updated_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.TaskID,
		task.CaseID,
		nullString(task.EvidenceItemID),
		task.UserID,
		task.Title,
		task.Description,
		string(task.Status),
		nullString(task.TemplateName),
		task.CreatedAt,
		task.UpdatedAt,
		nullString(task.CompletedAt),
	)
	return wrapError(err)
}

func (r *TaskRepo) GetTask(ctx context.Context, taskID string) (*models.Task, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT task_id, case_id, evidence_item_id, user_id, title, description,
		        status, template_name, created_at, updated_at, completed_at
		 FROM tasks WHERE task_id = ?`, taskID)
	task, err := r.scanTask(row)
	if err != nil {
		return nil, err
	}
	blocks, err := r.GetLinkedBlocks(ctx, taskID)
	if err != nil {
		return nil, err
	}
	task.LinkedBlocks = blocks
	return task, nil
}

func (r *TaskRepo) ListTasks(ctx context.Context, caseID string) ([]models.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT task_id, case_id, evidence_item_id, user_id, title, description,
		        status, template_name, created_at, updated_at, completed_at
		 FROM tasks WHERE case_id = ? ORDER BY created_at ASC`, caseID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapError(err)
	}
	if len(tasks) == 0 {
		return []models.Task{}, nil
	}

	// Batch fetch all linked blocks in one query instead of one per task.
	taskIDs := make([]string, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.TaskID
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(taskIDs)), ",")
	blockArgs := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		blockArgs[i] = id
	}
	blockRows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT tnl.task_id, tnl.block_id, nb.created_at, nb.content_hash,
		                    nb.evidence_item_id, nb.encrypted_body, tnl.linked_at
		             FROM task_note_links tnl
		             JOIN note_blocks nb ON nb.block_id = tnl.block_id
		             WHERE tnl.task_id IN (%s)
		             ORDER BY tnl.task_id, tnl.linked_at ASC`, placeholders),
		blockArgs...)
	if err != nil {
		return nil, wrapError(err)
	}
	defer blockRows.Close()
	blocksByTask := make(map[string][]models.LinkedBlock, len(tasks))
	for blockRows.Next() {
		var taskID string
		var b models.LinkedBlock
		var evidenceItemID sql.NullString
		if err := blockRows.Scan(&taskID, &b.BlockID, &b.CommittedAt, &b.ContentHash,
			&evidenceItemID, &b.EncryptedBody, &b.LinkedAt); err != nil {
			return nil, wrapError(err)
		}
		if evidenceItemID.Valid {
			b.Source = evidenceItemID.String
		}
		blocksByTask[taskID] = append(blocksByTask[taskID], b)
	}
	if err := blockRows.Err(); err != nil {
		return nil, wrapError(err)
	}
	for i := range tasks {
		if blocks, ok := blocksByTask[tasks[i].TaskID]; ok {
			tasks[i].LinkedBlocks = blocks
		}
	}

	return tasks, nil
}

func (r *TaskRepo) UpdateTask(ctx context.Context, task models.Task) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE tasks
		 SET title = ?, description = ?, evidence_item_id = ?, status = ?,
		     template_name = ?, updated_at = ?, completed_at = ?
		 WHERE task_id = ?`,
		task.Title,
		task.Description,
		nullString(task.EvidenceItemID),
		string(task.Status),
		nullString(task.TemplateName),
		task.UpdatedAt,
		nullString(task.CompletedAt),
		task.TaskID,
	)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *TaskRepo) DeleteTask(ctx context.Context, taskID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE task_id = ?`, taskID)
	if err != nil {
		return wrapError(err)
	}
	return checkRowsAffected(result)
}

func (r *TaskRepo) LinkNoteBlock(ctx context.Context, taskID string, blockID string) error {
	linkedAt := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO task_note_links (task_id, block_id, linked_at) VALUES (?, ?, ?)`,
		taskID, blockID, linkedAt,
	)
	return wrapError(err)
}

func (r *TaskRepo) UnlinkNoteBlock(ctx context.Context, taskID string, blockID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM task_note_links WHERE task_id = ? AND block_id = ?`,
		taskID, blockID,
	)
	return wrapError(err)
}

func (r *TaskRepo) GetLinkedBlocks(ctx context.Context, taskID string) ([]models.LinkedBlock, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT tnl.block_id, nb.created_at, nb.content_hash, nb.evidence_item_id,
		        nb.encrypted_body, tnl.linked_at
		 FROM task_note_links tnl
		 JOIN note_blocks nb ON nb.block_id = tnl.block_id
		 WHERE tnl.task_id = ?
		 ORDER BY tnl.linked_at ASC`, taskID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var blocks []models.LinkedBlock
	for rows.Next() {
		var b models.LinkedBlock
		var evidenceItemID sql.NullString
		if err := rows.Scan(
			&b.BlockID, &b.CommittedAt, &b.ContentHash,
			&evidenceItemID, &b.EncryptedBody, &b.LinkedAt,
		); err != nil {
			return nil, wrapError(err)
		}
		if evidenceItemID.Valid {
			b.Source = evidenceItemID.String
		}
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapError(err)
	}
	if blocks == nil {
		blocks = []models.LinkedBlock{}
	}
	return blocks, nil
}

func (r *TaskRepo) GetLinkedTasks(ctx context.Context, blockID string) ([]models.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.task_id, t.case_id, t.evidence_item_id, t.user_id, t.title,
		        t.description, t.status, t.template_name, t.created_at, t.updated_at, t.completed_at
		 FROM tasks t
		 JOIN task_note_links tnl ON tnl.task_id = t.task_id
		 WHERE tnl.block_id = ?
		 ORDER BY t.created_at ASC`, blockID)
	if err != nil {
		return nil, wrapError(err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, err
		}
		task.LinkedBlocks = []models.LinkedBlock{}
		tasks = append(tasks, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapError(err)
	}
	if tasks == nil {
		tasks = []models.Task{}
	}
	return tasks, nil
}

func (r *TaskRepo) scanTask(s scanner) (*models.Task, error) {
	var task models.Task
	var evidenceItemID, templateName, completedAt sql.NullString

	err := s.Scan(
		&task.TaskID, &task.CaseID, &evidenceItemID, &task.UserID,
		&task.Title, &task.Description, &task.Status, &templateName,
		&task.CreatedAt, &task.UpdatedAt, &completedAt,
	)
	if err != nil {
		return nil, wrapError(err)
	}

	if evidenceItemID.Valid {
		task.EvidenceItemID = &evidenceItemID.String
	}
	if templateName.Valid {
		task.TemplateName = &templateName.String
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.String
	}
	task.LinkedBlocks = []models.LinkedBlock{}
	return &task, nil
}
