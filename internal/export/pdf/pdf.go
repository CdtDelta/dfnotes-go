package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"

	"dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
	"dfnotes-go/internal/services"
)

// AttachmentInfo holds decrypted image data for PDF embedding.
type AttachmentInfo struct {
	Data        []byte
	ContentType string
	Filename    string
}

// PDFRequest contains all data needed to generate a case PDF.
type PDFRequest struct {
	CaseData         *services.CaseResponse
	ExaminerName     string
	EvidenceItems    []services.EvidenceResponse
	MasterBlocks     []services.NoteBlockResponse
	EvidenceBlockMap map[string][]services.NoteBlockResponse
	RawBlocks        []models.NoteBlock
	IOCEntries       []ioc.IOCEntry
	TimelineEntries  []models.TimelineEntry
	Tasks            []models.Task
	Attachments      map[string]*AttachmentInfo
	AppVersion       string
	ExportedAt       time.Time
}

// GenerateCasePDF generates a full-case PDF and returns its bytes.
func GenerateCasePDF(req PDFRequest) ([]byte, error) {
	if req.ExportedAt.IsZero() {
		req.ExportedAt = time.Now().UTC()
	}
	if req.AppVersion == "" {
		req.AppVersion = appVersion
	}

	p := fpdf.New("P", "mm", "A4", "")
	p.AliasNbPages("{nb}")
	p.SetMargins(marginLeft, marginTop, marginRight)
	p.SetAutoPageBreak(true, marginBottom)

	cc := classColorFor(req.CaseData.Classification)
	caseNum := req.CaseData.CaseNumber

	// Header: full-width colored bar with classification text
	p.SetHeaderFunc(func() {
		p.SetFont("Helvetica", "B", 12)
		p.SetFillColor(cc.BgR, cc.BgG, cc.BgB)
		p.SetTextColor(cc.TextR, cc.TextG, cc.TextB)
		p.SetXY(0, 5)
		p.CellFormat(pageWidth, 10, req.CaseData.Classification, "", 1, "C", true, 0, "")
	})

	// Footer: classification left | case number center | page N of M right
	p.SetFooterFunc(func() {
		p.SetY(-12)
		p.SetFont("Helvetica", "", fontSizeSm)
		p.SetTextColor(180, 180, 180)
		p.CellFormat(bodyWidth/3, lineHeight, req.CaseData.Classification, "", 0, "L", false, 0, "")
		p.CellFormat(bodyWidth/3, lineHeight, "Case: "+caseNum, "", 0, "C", false, 0, "")
		p.CellFormat(bodyWidth/3, lineHeight, fmt.Sprintf("Page %d of {nb}", p.PageNo()), "", 0, "R", false, 0, "")
	})

	// Build evidence index (itemID -> "E001" etc.)
	evidenceIndex := buildEvidenceIndex(req.EvidenceItems)

	// Build raw block map (blockID -> NoteBlock) for amendment lookup
	rawBlockMap := make(map[string]models.NoteBlock, len(req.RawBlocks))
	for _, b := range req.RawBlocks {
		rawBlockMap[b.BlockID] = b
	}

	// All blocks in chain order for appendix image scan
	allBlocks := make([]services.NoteBlockResponse, 0, len(req.MasterBlocks))
	allBlocks = append(allBlocks, req.MasterBlocks...)
	for _, evid := range req.EvidenceItems {
		allBlocks = append(allBlocks, req.EvidenceBlockMap[evid.EvidenceItemID]...)
	}

	// Page 1: Cover
	BuildCoverPage(p, req)

	// Page 2: TOC placeholder
	tocPage, tocY := BuildTOCPlaceholder(p)

	// Sections -- track TOC entries
	var tocEntries []TOCEntry

	addSection := func(name string, link, page int) {
		tocEntries = append(tocEntries, TOCEntry{
			Name: name,
			Link: link,
			Page: page,
		})
	}

	link, pg := BuildMasterNotesSection(p, req.MasterBlocks, rawBlockMap)
	addSection("Master Notes", link, pg)

	link, pg = BuildEvidenceSection(p, req.EvidenceItems, req.EvidenceBlockMap, rawBlockMap, evidenceIndex)
	addSection("Evidence Items", link, pg)

	link, pg = BuildIOCSection(p, req.IOCEntries)
	addSection("IOC Summary", link, pg)

	link, pg = BuildTimelineSection(p, req.TimelineEntries, evidenceIndex)
	addSection("Timeline", link, pg)

	link, pg = BuildTaskListSection(p, req.Tasks, req.EvidenceItems, evidenceIndex)
	addSection("Task List", link, pg)

	link, pg = BuildChainVerificationSection(p, req.RawBlocks)
	addSection("Chain Verification", link, pg)

	link, pg = BuildAppendixImages(p, allBlocks, req.Attachments)
	if link != 0 {
		addSection("Appendix A -- Images", link, pg)
	}

	lastPage := p.PageNo()

	// Fill TOC on page 2
	FillTOC(p, tocPage, tocY, tocEntries, lastPage)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildEvidenceIndex(items []services.EvidenceResponse) map[string]string {
	sorted := make([]services.EvidenceResponse, len(items))
	copy(sorted, items)
	// Sort by created_at for consistent numbering
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].CreatedAt < sorted[j-1].CreatedAt; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	index := make(map[string]string, len(sorted))
	for i, item := range sorted {
		index[item.EvidenceItemID] = fmt.Sprintf("E%03d", i+1)
	}
	return index
}

// newBytesReader returns an io.Reader over the provided bytes.
func newBytesReader(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}
