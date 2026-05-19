package models

type TaskStatus string

const (
	TaskStatusOpen          TaskStatus = "open"
	TaskStatusInProgress    TaskStatus = "in_progress"
	TaskStatusBlocked       TaskStatus = "blocked"
	TaskStatusComplete      TaskStatus = "complete"
	TaskStatusNotApplicable TaskStatus = "not_applicable"
)

type Task struct {
	TaskID         string      `json:"task_id"`
	CaseID         string      `json:"case_id"`
	EvidenceItemID *string     `json:"evidence_item_id"`
	UserID         string      `json:"user_id"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Status         TaskStatus  `json:"status"`
	TemplateName   *string     `json:"template_name"`
	CreatedAt      string      `json:"created_at"`
	UpdatedAt      string      `json:"updated_at"`
	CompletedAt    *string     `json:"completed_at"`
	LinkedBlocks   []LinkedBlock `json:"linked_blocks"`
}

type LinkedBlock struct {
	BlockID       string `json:"block_id"`
	CommittedAt   string `json:"committed_at"`
	ContentHash   string `json:"content_hash"`
	Preview       string `json:"preview"`
	Source        string `json:"source"` // "" for master notes, evidence_item_id for evidence blocks
	LinkedAt      string `json:"linked_at"`
	EncryptedBody []byte `json:"-"` // populated by repo, consumed by service for preview
}
