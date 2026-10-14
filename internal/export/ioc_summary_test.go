package export

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"dfnotes-go/internal/ioc"
)

func strPtr(s string) *string { return &s }

func summaryFixture() []ioc.Indicator {
	return []ioc.Indicator{
		{
			IndicatorID:     "ind-1",
			CaseID:          "case-1",
			Type:            ioc.IOCTypeDomain,
			Value:           "evil.example.com",
			MatchKey:        "evil.example.com",
			MatchKeyVersion: 1,
			Status:          ioc.IOCStatusConfirmed,
			CreatedAt:       "2026-10-01T10:00:00Z",
			ConfirmedAt:     strPtr("2026-10-02T10:00:00Z"),
			Occurrences: []ioc.Occurrence{
				{OccurrenceID: "occ-1", BlockID: "blk-master", RawValue: "evil.example.com", DetectionMethod: "auto", CommittedAt: "2026-10-01T10:00:00Z", CreatedAt: "2026-10-01T10:00:01Z"},
				{OccurrenceID: "occ-2", BlockID: "blk-ev", EvidenceItemID: strPtr("ev-1"), RawValue: "EVIL.example.com", DetectionMethod: "propagated", CommittedAt: "2026-10-03T10:00:00Z", CreatedAt: "2026-10-03T10:00:01Z"},
			},
		},
		{
			IndicatorID:     "ind-2",
			CaseID:          "case-1",
			Type:            ioc.IOCTypeSHA256,
			Value:           "E3B0C442",
			MatchKey:        "e3b0c442",
			MatchKeyVersion: 1,
			Status:          ioc.IOCStatusDetected,
			CreatedAt:       "2026-10-01T09:00:00Z",
			Occurrences: []ioc.Occurrence{
				{OccurrenceID: "occ-3", BlockID: "blk-gone", RawValue: "E3B0C442", DetectionMethod: "auto", CommittedAt: "2026-10-01T09:00:00Z"},
			},
		},
	}
}

func TestBuildIOCSummaryV2(t *testing.T) {
	seq := map[string]int{"blk-master": 3, "blk-ev": 7}
	items := map[string]string{"ev-1": "E001"}
	at := time.Date(2026, 10, 14, 21, 0, 0, 0, time.FixedZone("x", -4*3600))

	got, missing := buildIOCSummaryV2(summaryFixture(), seq, items, at)

	if got.FormatVersion != 2 {
		t.Errorf("format_version = %d, want 2", got.FormatVersion)
	}
	if got.ExportedAt != "2026-10-15T01:00:00Z" {
		t.Errorf("exported_at = %q, want UTC RFC3339", got.ExportedAt)
	}
	if got.Count != 2 || len(got.Indicators) != 2 {
		t.Fatalf("count = %d, len = %d, want 2 indicators", got.Count, len(got.Indicators))
	}
	if missing != 1 {
		t.Errorf("missing = %d, want 1", missing)
	}

	dom := got.Indicators[0]
	if dom.IndicatorID != "ind-1" || got.Indicators[1].IndicatorID != "ind-2" {
		t.Errorf("indicator order changed: %s, %s", dom.IndicatorID, got.Indicators[1].IndicatorID)
	}
	if dom.Defanged != ioc.Defang("evil.example.com", ioc.IOCTypeDomain) || dom.Defanged != "evil.example[.]com" {
		t.Errorf("defanged = %q", dom.Defanged)
	}
	if dom.ConfirmedAt == nil || *dom.ConfirmedAt != "2026-10-02T10:00:00Z" {
		t.Errorf("confirmed_at = %v", dom.ConfirmedAt)
	}

	master := dom.Occurrences[0]
	if master.EvidenceItemID != nil || master.ItemNumber != nil {
		t.Errorf("master occurrence evidence = %v / %v, want nil / nil", master.EvidenceItemID, master.ItemNumber)
	}
	if master.BlockSequence != 3 {
		t.Errorf("master block_sequence = %d, want 3", master.BlockSequence)
	}
	ev := dom.Occurrences[1]
	if ev.EvidenceItemID == nil || *ev.EvidenceItemID != "ev-1" || ev.ItemNumber == nil || *ev.ItemNumber != "E001" {
		t.Errorf("evidence occurrence = %v / %v, want ev-1 / E001", ev.EvidenceItemID, ev.ItemNumber)
	}
	if ev.BlockSequence != 7 || ev.RawValue != "EVIL.example.com" || ev.DetectionMethod != "propagated" || ev.CommittedAt != "2026-10-03T10:00:00Z" {
		t.Errorf("evidence occurrence fields = %+v", ev)
	}

	if got.Indicators[1].Occurrences[0].BlockSequence != 0 {
		t.Errorf("unknown block sequence = %d, want 0", got.Indicators[1].Occurrences[0].BlockSequence)
	}
}

func TestBuildIOCSummaryV2Empty(t *testing.T) {
	got, _ := buildIOCSummaryV2(nil, nil, nil, time.Now())
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if inds, ok := m["indicators"].([]any); !ok || len(inds) != 0 {
		t.Errorf("indicators = %v, want empty array (not null)", m["indicators"])
	}
	if m["count"].(float64) != 0 {
		t.Errorf("count = %v, want 0", m["count"])
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func assertKeys(t *testing.T, what string, m map[string]any, want []string) {
	t.Helper()
	sort.Strings(want)
	got := sortedKeys(m)
	if len(got) != len(want) {
		t.Errorf("%s keys = %v, want %v", what, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s keys = %v, want %v", what, got, want)
			return
		}
	}
}

func TestIOCSummaryV2JSONKeys(t *testing.T) {
	seq := map[string]int{"blk-master": 3, "blk-ev": 7, "blk-gone": 1}
	got, _ := buildIOCSummaryV2(summaryFixture(), seq, map[string]string{"ev-1": "E001"}, time.Now())
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	assertKeys(t, "top", top, []string{"format_version", "exported_at", "count", "indicators"})

	inds := top["indicators"].([]any)
	indicatorKeys := []string{"indicator_id", "type", "value", "defanged", "match_key", "match_key_version", "status", "created_at", "occurrences"}
	assertKeys(t, "confirmed indicator", inds[0].(map[string]any), append([]string{"confirmed_at"}, indicatorKeys...))
	// confirmed_at is omitted, not null, when the indicator was never confirmed.
	assertKeys(t, "detected indicator", inds[1].(map[string]any), indicatorKeys)

	occKeys := []string{"occurrence_id", "block_id", "block_sequence", "evidence_item_id", "item_number", "raw_value", "detection_method", "committed_at"}
	occs := inds[0].(map[string]any)["occurrences"].([]any)
	master := occs[0].(map[string]any)
	assertKeys(t, "master occurrence", master, occKeys)
	if master["evidence_item_id"] != nil || master["item_number"] != nil {
		t.Errorf("master occurrence nulls = %v / %v, want JSON null", master["evidence_item_id"], master["item_number"])
	}
	assertKeys(t, "evidence occurrence", occs[1].(map[string]any), occKeys)
}

func TestCaseFactsFormatVersion(t *testing.T) {
	data, err := json.Marshal(caseFactsSummary{FormatVersion: caseFactsFormatVersion, Facts: []caseFactExport{}})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["format_version"].(float64) != 2 {
		t.Errorf("case_facts format_version = %v, want 2", m["format_version"])
	}
}
