package models

import "time"

type TimelineEntry struct {
	EntryID       string    `json:"entry_id"`
	CaseID        string    `json:"case_id"`
	EventTime     time.Time `json:"event_time"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	SourceBlockID *string   `json:"source_block_id,omitempty"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}
