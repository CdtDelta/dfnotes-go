package models

import "time"

type NoteBlock struct {
	BlockID        string    `json:"block_id"`
	CaseID         string    `json:"case_id"`
	EvidenceItemID *string   `json:"evidence_item_id,omitempty"`
	AmendsBlockID  *string   `json:"amends_block_id,omitempty"`
	ContentHash    string    `json:"content_hash"`
	PrevHash       string    `json:"prev_hash"`
	Signature      []byte    `json:"signature"`
	EncryptedBody  []byte    `json:"encrypted_body"`
	AuthorID       string    `json:"author_id"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []Tag     `json:"tags,omitempty"`
}
