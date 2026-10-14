package verify

import "dfnotes-go/internal/models"

// SequenceByBlock maps each block id to its 1-based chain position, the same
// numbering Chain assigns to BlockResult.Sequence. blocks MUST be in chain
// order, as for Chain. Exports use it to label blocks without re-verifying.
func SequenceByBlock(blocks []models.NoteBlock) map[string]int {
	seq := make(map[string]int, len(blocks))
	for i, b := range blocks {
		seq[b.BlockID] = i + 1
	}
	return seq
}
