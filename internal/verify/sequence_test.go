package verify

import (
	"testing"

	"dfnotes-go/internal/models"
)

func TestSequenceByBlock(t *testing.T) {
	blocks := []models.NoteBlock{{BlockID: "a"}, {BlockID: "b"}, {BlockID: "c"}}
	got := SequenceByBlock(blocks)
	want := map[string]int{"a": 1, "b": 2, "c": 3}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for id, n := range want {
		if got[id] != n {
			t.Errorf("seq[%s] = %d, want %d", id, got[id], n)
		}
	}
}
