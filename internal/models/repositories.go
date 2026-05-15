package models

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *UserIdentity) error
	GetByID(ctx context.Context, userID string) (*UserIdentity, error)
	GetFirst(ctx context.Context) (*UserIdentity, error)
	Update(ctx context.Context, user *UserIdentity) error
}

type CaseRepository interface {
	Create(ctx context.Context, c *Case) error
	GetByID(ctx context.Context, caseID string) (*Case, error)
	List(ctx context.Context) ([]Case, error)
	Update(ctx context.Context, c *Case) error
	Delete(ctx context.Context, caseID string) error
}

type EvidenceRepository interface {
	Create(ctx context.Context, item *EvidenceItem) error
	GetByID(ctx context.Context, evidenceItemID string) (*EvidenceItem, error)
	ListByCase(ctx context.Context, caseID string) ([]EvidenceItem, error)
	Update(ctx context.Context, item *EvidenceItem) error
	Delete(ctx context.Context, evidenceItemID string) error
}

type NoteBlockRepository interface {
	Create(ctx context.Context, block *NoteBlock) error
	GetByID(ctx context.Context, blockID string) (*NoteBlock, error)
	ListByCase(ctx context.Context, caseID string) ([]NoteBlock, error)
	ListByEvidence(ctx context.Context, evidenceItemID string) ([]NoteBlock, error)
	GetLastBlock(ctx context.Context, caseID string) (*NoteBlock, error)
}

type TimelineRepository interface {
	Create(ctx context.Context, entry *TimelineEntry) error
	GetByID(ctx context.Context, entryID string) (*TimelineEntry, error)
	ListByCase(ctx context.Context, caseID string) ([]TimelineEntry, error)
	Update(ctx context.Context, entry *TimelineEntry) error
	Delete(ctx context.Context, entryID string) error
}

type TagRepository interface {
	Create(ctx context.Context, tag *Tag) error
	GetByID(ctx context.Context, tagID string) (*Tag, error)
	List(ctx context.Context) ([]Tag, error)
	Delete(ctx context.Context, tagID string) error
	AttachToBlock(ctx context.Context, blockID, tagID string) error
	DetachFromBlock(ctx context.Context, blockID, tagID string) error
	ListByBlock(ctx context.Context, blockID string) ([]Tag, error)
	AttachToEvidence(ctx context.Context, evidenceItemID, tagID string) error
	DetachFromEvidence(ctx context.Context, evidenceItemID, tagID string) error
	ListByEvidence(ctx context.Context, evidenceItemID string) ([]Tag, error)
}

type AttachmentRepository interface {
	Create(ctx context.Context, att *Attachment) error
	GetByID(ctx context.Context, attachmentID string) (*Attachment, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, entry *AuditLog) error
	ListByCase(ctx context.Context, caseID string) ([]AuditLog, error)
	ListByUser(ctx context.Context, userID string) ([]AuditLog, error)
}
