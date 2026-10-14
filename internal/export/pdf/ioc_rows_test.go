package pdf

import (
	"testing"

	"github.com/go-pdf/fpdf"

	"dfnotes-go/internal/ioc"
)

func ptr(s string) *string { return &s }

func occ(blockID string, evidenceID *string) ioc.Occurrence {
	return ioc.Occurrence{BlockID: blockID, EvidenceItemID: evidenceID}
}

func TestBuildIOCTableRowsGrouping(t *testing.T) {
	inds := []ioc.Indicator{
		{IndicatorID: "d-late", Type: ioc.IOCTypeIPv4, Value: "1.2.3.4", Status: ioc.IOCStatusDetected, CreatedAt: "2026-10-05T00:00:00Z"},
		{IndicatorID: "p", Type: ioc.IOCTypeDomain, Value: "a.example.com", Status: ioc.IOCStatusPromoted, CreatedAt: "2026-10-01T00:00:00Z"},
		{IndicatorID: "c-late", Type: ioc.IOCTypeMD5, Value: "aa", Status: ioc.IOCStatusConfirmed, CreatedAt: "2026-10-04T00:00:00Z"},
		{IndicatorID: "fp", Type: ioc.IOCTypeCVE, Value: "CVE-2024-1", Status: ioc.IOCStatusFalsePositive, CreatedAt: "2026-10-01T00:00:00Z"},
		{IndicatorID: "d-early", Type: ioc.IOCTypeMD5, Value: "bb", Status: ioc.IOCStatusDetected, CreatedAt: "2026-10-01T00:00:00Z"},
		{IndicatorID: "c-early", Type: ioc.IOCTypeMD5, Value: "cc", Status: ioc.IOCStatusConfirmed, CreatedAt: "2026-10-02T00:00:00Z"},
	}
	g := buildIOCTableRows(inds, map[string]int{}, map[string]string{})

	// Active: confirmed (by created_at) then detected (by created_at).
	wantActive := []string{"cc", "aa", "bb", "1[.]2[.]3[.]4"}
	if len(g.Active) != len(wantActive) {
		t.Fatalf("active rows = %d, want %d", len(g.Active), len(wantActive))
	}
	for i, v := range wantActive {
		if g.Active[i][1] != v {
			t.Errorf("active[%d] value = %q, want %q", i, g.Active[i][1], v)
		}
	}
	if len(g.Promoted) != 1 || g.Promoted[0][2] != "promoted" || g.Promoted[0][1] != "a.example[.]com" {
		t.Errorf("promoted rows = %v, want the one promoted indicator, defanged", g.Promoted)
	}
	if len(g.Dismissed) != 1 || g.Dismissed[0][2] != "false_positive" {
		t.Errorf("dismissed rows = %v", g.Dismissed)
	}
	for _, r := range g.Active {
		if r[2] == "promoted" || r[2] == "false_positive" {
			t.Errorf("active table holds a %s row", r[2])
		}
	}
	if len(g.Active[0]) != len(iocTableHeaders) {
		t.Errorf("row has %d cells, want %d", len(g.Active[0]), len(iocTableHeaders))
	}
}

func TestBuildIOCTableRowsPlaces(t *testing.T) {
	seq := map[string]int{"m1": 12, "e1": 3, "e2": 7, "m2": 1, "e3": 9}
	evidence := map[string]string{"ev-a": "E002", "ev-b": "E001"}
	ind := ioc.Indicator{
		Type:   ioc.IOCTypeSHA256,
		Value:  "ABC",
		Status: ioc.IOCStatusConfirmed,
		Occurrences: []ioc.Occurrence{
			occ("m1", nil),
			occ("e1", ptr("ev-a")),
			occ("e2", ptr("ev-a")),
			occ("e1", ptr("ev-a")), // same block, second raw form
			occ("m2", nil),
			occ("e3", ptr("ev-b")),
			occ("gone", ptr("ev-unknown")),
		},
	}
	g := buildIOCTableRows([]ioc.Indicator{ind}, seq, evidence)
	row := g.Active[0]

	if row[3] != "Master, E002, E001, ev-unknown" {
		t.Errorf("Evidence = %q, want first-appearance order, de-duplicated", row[3])
	}
	if row[4] != "1, 3, 7, 9, 12" {
		t.Errorf("Blocks = %q, want ascending, de-duplicated", row[4])
	}
	if g.MissingBlocks != 1 {
		t.Errorf("MissingBlocks = %d, want 1", g.MissingBlocks)
	}
}

func TestIOCTableWidths(t *testing.T) {
	sum := 0.0
	for _, w := range iocTableWidths {
		sum += w
	}
	if sum != bodyWidth {
		t.Errorf("IOC table widths sum to %v, want bodyWidth %v", sum, bodyWidth)
	}
	if len(iocTableWidths) != len(iocTableHeaders) {
		t.Errorf("%d widths for %d headers", len(iocTableWidths), len(iocTableHeaders))
	}
}

// Long Blocks lists are not capped: the cell keeps every number and
// DrawTableMonospace wraps it over several lines.
func TestBuildIOCTableRowsLongBlocksNotCapped(t *testing.T) {
	seq := make(map[string]int)
	var occs []ioc.Occurrence
	for i := 1; i <= 60; i++ {
		id := string(rune('A'+i%26)) + string(rune('a'+i/26))
		seq[id] = i
		occs = append(occs, occ(id, nil))
	}
	g := buildIOCTableRows([]ioc.Indicator{{Type: ioc.IOCTypeMD5, Status: ioc.IOCStatusDetected, Occurrences: occs}}, seq, nil)
	blocks := g.Active[0][4]
	if blocks[:8] != "1, 2, 3," || blocks[len(blocks)-6:] != "59, 60" {
		t.Errorf("Blocks = %q, want the full list 1 through 60", blocks)
	}

	p := fpdf.New("P", "mm", "A4", "")
	p.AddUTF8FontFromBytes("DejaVu", "", dejaVuSansBytes)
	p.SetFont("DejaVu", "", 8)
	if n := countLines(p, blocks, iocTableWidths[4]); n < 2 {
		t.Errorf("Blocks cell measures %d line(s) at width %v, want it to wrap", n, iocTableWidths[4])
	}
}
