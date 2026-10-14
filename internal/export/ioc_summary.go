package export

import (
	"time"

	"dfnotes-go/internal/ioc"
)

// iocSummaryFormatVersion is written into ioc_summary.json. Version 1 had no
// format_version field and held one flat row per occurrence.
const iocSummaryFormatVersion = 2

type iocSummaryV2 struct {
	FormatVersion int              `json:"format_version"`
	ExportedAt    string           `json:"exported_at"`
	Count         int              `json:"count"`
	Indicators    []iocIndicatorV2 `json:"indicators"`
}

type iocIndicatorV2 struct {
	IndicatorID     string            `json:"indicator_id"`
	Type            string            `json:"type"`
	Value           string            `json:"value"`
	Defanged        string            `json:"defanged"`
	MatchKey        string            `json:"match_key"`
	MatchKeyVersion int               `json:"match_key_version"`
	Status          string            `json:"status"`
	CreatedAt       string            `json:"created_at"`
	ConfirmedAt     *string           `json:"confirmed_at,omitempty"`
	Occurrences     []iocOccurrenceV2 `json:"occurrences"`
}

// EvidenceItemID and ItemNumber are pointers without omitempty so Master
// Notes occurrences carry explicit nulls rather than missing keys.
type iocOccurrenceV2 struct {
	OccurrenceID    string  `json:"occurrence_id"`
	BlockID         string  `json:"block_id"`
	BlockSequence   int     `json:"block_sequence"`
	EvidenceItemID  *string `json:"evidence_item_id"`
	ItemNumber      *string `json:"item_number"`
	RawValue        string  `json:"raw_value"`
	DetectionMethod string  `json:"detection_method"`
	CommittedAt     string  `json:"committed_at"`
}

// buildIOCSummaryV2 shapes indicators for ioc_summary.json. Indicator and
// occurrence order is kept as given. It also returns how many occurrences
// referenced a block missing from seqByBlock; those get block_sequence 0 and
// the caller decides whether to log.
func buildIOCSummaryV2(indicators []ioc.Indicator, seqByBlock map[string]int, itemNumberByEvidenceID map[string]string, exportedAt time.Time) (iocSummaryV2, int) {
	missing := 0
	out := make([]iocIndicatorV2, 0, len(indicators))
	for _, ind := range indicators {
		occs := make([]iocOccurrenceV2, 0, len(ind.Occurrences))
		for _, o := range ind.Occurrences {
			seq, ok := seqByBlock[o.BlockID]
			if !ok {
				missing++
			}
			var evidenceID, itemNumber *string
			if o.EvidenceItemID != nil {
				id := *o.EvidenceItemID
				evidenceID = &id
				if n, ok := itemNumberByEvidenceID[id]; ok {
					itemNumber = &n
				}
			}
			occs = append(occs, iocOccurrenceV2{
				OccurrenceID:    o.OccurrenceID,
				BlockID:         o.BlockID,
				BlockSequence:   seq,
				EvidenceItemID:  evidenceID,
				ItemNumber:      itemNumber,
				RawValue:        o.RawValue,
				DetectionMethod: o.DetectionMethod,
				CommittedAt:     o.CommittedAt,
			})
		}
		out = append(out, iocIndicatorV2{
			IndicatorID:     ind.IndicatorID,
			Type:            string(ind.Type),
			Value:           ind.Value,
			Defanged:        ioc.Defang(ind.Value, ind.Type),
			MatchKey:        ind.MatchKey,
			MatchKeyVersion: ind.MatchKeyVersion,
			Status:          string(ind.Status),
			CreatedAt:       ind.CreatedAt,
			ConfirmedAt:     ind.ConfirmedAt,
			Occurrences:     occs,
		})
	}
	return iocSummaryV2{
		FormatVersion: iocSummaryFormatVersion,
		ExportedAt:    exportedAt.UTC().Format(time.RFC3339),
		Count:         len(out),
		Indicators:    out,
	}, missing
}
