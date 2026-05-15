package models

type TimelineEntry struct {
	EntryID           string  `json:"entry_id"`
	CaseID            string  `json:"case_id"`
	EvidenceItemID    *string `json:"evidence_item_id,omitempty"`
	Timestamp         string  `json:"timestamp"`
	DisplayTimezone   *string `json:"display_timezone,omitempty"`
	EventDescription  string  `json:"event_description"`
	InvestigatorNotes string  `json:"investigator_notes"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	UserID            string  `json:"user_id"`
}

type CreateTimelineEntryRequest struct {
	CaseID            string  `json:"case_id"`
	EvidenceItemID    *string `json:"evidence_item_id"`
	Timestamp         string  `json:"timestamp"`
	DisplayTimezone   *string `json:"display_timezone"`
	EventDescription  string  `json:"event_description"`
	InvestigatorNotes string  `json:"investigator_notes"`
}

type UpdateTimelineEntryRequest struct {
	EntryID           string  `json:"entry_id"`
	Timestamp         string  `json:"timestamp"`
	DisplayTimezone   *string `json:"display_timezone"`
	EventDescription  string  `json:"event_description"`
	InvestigatorNotes string  `json:"investigator_notes"`
}
