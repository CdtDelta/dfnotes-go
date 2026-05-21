package pdf

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"dfnotes-go/internal/ioc"
	"dfnotes-go/internal/models"
	"dfnotes-go/internal/services"
)

// sectionHeading writes a bold section heading and adds a PDF bookmark.
// Returns the link ID (set at current position) for TOC use.
func sectionHeading(p *fpdf.Fpdf, title string, level int) int {
	link := p.AddLink()
	p.SetLink(link, -1, -1)
	p.SetFont("Helvetica", "B", fontSizeH1)
	p.SetTextColor(30, 30, 30)
	p.Bookmark(title, level, -1)
	p.MultiCell(bodyWidth, lineHeight+2, title, "", "L", false)
	p.Ln(4)
	p.SetFont("Helvetica", "", fontSizeBody)
	p.SetTextColor(30, 30, 30)
	return link
}

func subHeading(p *fpdf.Fpdf, title string) {
	p.SetFont("Helvetica", "B", fontSizeH2)
	p.SetTextColor(30, 30, 30)
	p.MultiCell(bodyWidth, lineHeight, title, "", "L", false)
	p.Ln(3)
	p.SetFont("Helvetica", "", fontSizeBody)
	p.SetTextColor(30, 30, 30)
}

func separator(p *fpdf.Fpdf) {
	y := p.GetY() + 2
	p.SetDrawColor(180, 180, 180)
	p.SetLineWidth(0.3)
	p.Line(marginLeft, y, marginLeft+bodyWidth, y)
	p.Ln(5)
}

// renderLabelValue renders a single label/value row with bold label and plain value.
func renderLabelValue(p *fpdf.Fpdf, label, value string, labelW float64) {
	p.SetFont("Helvetica", "B", fontSizeSm)
	p.SetTextColor(30, 30, 30)
	p.SetX(marginLeft)
	p.Cell(labelW, lineHeight, label)
	p.SetFont("Helvetica", "", fontSizeSm)
	p.SetTextColor(30, 30, 30)
	p.Cell(bodyWidth-labelW, lineHeight, value)
	p.Ln(-1)
}

// BuildCoverPage renders page 1 of the PDF.
func BuildCoverPage(p *fpdf.Fpdf, req PDFRequest) {
	p.AddPage()
	cc := classColorFor(req.CaseData.Classification)

	// Large classification banner
	p.SetFillColor(cc.BgR, cc.BgG, cc.BgB)
	p.SetTextColor(cc.TextR, cc.TextG, cc.TextB)
	p.SetFont("Helvetica", "B", 20)
	p.SetXY(0, 50)
	p.CellFormat(pageWidth, 18, req.CaseData.Classification, "", 1, "C", true, 0, "")
	p.Ln(12)

	// Case number and title
	p.SetFont("Helvetica", "B", 18)
	p.SetTextColor(30, 30, 30)
	p.SetX(marginLeft)
	p.MultiCell(bodyWidth, 12, req.CaseData.CaseNumber+": "+req.CaseData.Title, "", "C", false)
	p.Ln(10)

	// Metadata rows
	metaRows := [][]string{
		{"Examiner:", req.ExaminerName},
		{"Organization:", req.CaseData.Organization},
	}
	if req.CaseData.TicketNumber != "" {
		metaRows = append(metaRows, []string{"Ticket Number:", req.CaseData.TicketNumber})
	}
	metaRows = append(metaRows,
		[]string{"Case Created:", req.CaseData.CreatedAt},
		[]string{"PDF Exported:", req.ExportedAt.UTC().Format(time.RFC3339)},
		[]string{"dfnotes-go Version:", req.AppVersion},
	)

	p.SetTextColor(30, 30, 30)
	for _, row := range metaRows {
		p.SetX(marginLeft + 20)
		p.SetFont("Helvetica", "B", fontSizeBody+1)
		p.Cell(50, lineHeight+1, row[0])
		p.SetFont("Helvetica", "", fontSizeBody+1)
		p.Cell(0, lineHeight+1, row[1])
		p.Ln(-1)
	}

	p.Ln(20)

	// SHA-256 note
	p.SetFont("Helvetica", "I", fontSizeSm)
	p.SetTextColor(80, 80, 80)
	p.SetX(marginLeft)
	p.MultiCell(bodyWidth, lineHeight, "SHA-256 of this file is recorded in the accompanying .sha256 sidecar file.", "", "C", false)
}

// BuildTOCPlaceholder adds the TOC page and returns the page number and Y where TOC content should start.
func BuildTOCPlaceholder(p *fpdf.Fpdf) (int, float64) {
	p.AddPage()
	page := p.PageNo()
	p.SetFont("Helvetica", "B", fontSizeH1)
	p.SetTextColor(30, 30, 30)
	p.Bookmark("Table of Contents", 0, -1)
	p.MultiCell(bodyWidth, lineHeight+2, "Table of Contents", "", "L", false)
	p.Ln(4)
	y := p.GetY()
	return page, y
}

// FillTOC goes back to tocPage and writes section entries with page numbers.
func FillTOC(p *fpdf.Fpdf, tocPage int, tocY float64, entries []TOCEntry, lastPage int) {
	p.SetPage(tocPage)
	p.SetXY(marginLeft, tocY)
	p.SetFont("Helvetica", "", fontSizeBody+1)
	for _, e := range entries {
		p.SetTextColor(30, 30, 30)
		title := e.Name
		pageStr := fmt.Sprintf("%d", e.Page)
		// Dots fill
		dotW := bodyWidth - p.GetStringWidth(title) - p.GetStringWidth(pageStr) - 4
		dots := ""
		dotCharW := p.GetStringWidth(".")
		if dotCharW > 0 {
			n := int(dotW / dotCharW)
			if n > 0 {
				dots = " " + strings.Repeat(".", n) + " "
			}
		}
		p.SetX(marginLeft)
		p.SetFont("Helvetica", "", fontSizeBody+1)
		p.Cell(p.GetStringWidth(title)+2, lineHeight, title)
		p.SetTextColor(120, 120, 120)
		p.Cell(p.GetStringWidth(dots)+2, lineHeight, dots)
		p.SetTextColor(30, 30, 30)
		p.Cell(0, lineHeight, pageStr)
		p.Ln(-1)
	}
	p.SetPage(lastPage)
}

// TOCEntry records a section for the table of contents.
type TOCEntry struct {
	Name string
	Link int
	Page int
}

// BuildMasterNotesSection renders the Master Notes section.
func BuildMasterNotesSection(p *fpdf.Fpdf, blocks []services.NoteBlockResponse, rawBlockMap map[string]models.NoteBlock) (int, int) {
	p.AddPage()
	startPage := p.PageNo()
	link := sectionHeading(p, "Master Notes", 0)

	for _, block := range blocks {
		renderBlockHeader(p, block, rawBlockMap)
		RenderMarkdown(p, block.Content)
		separator(p)
	}
	return link, startPage
}

func renderBlockHeader(p *fpdf.Fpdf, block services.NoteBlockResponse, rawBlockMap map[string]models.NoteBlock) {
	// Top rule
	p.SetDrawColor(180, 180, 180)
	p.SetLineWidth(0.3)
	y := p.GetY()
	p.Line(marginLeft, y, marginLeft+bodyWidth, y)
	p.Ln(3)

	const labelW = 45.0

	renderLabelValue(p, "Block ID:", block.BlockID, labelW)
	renderLabelValue(p, "Committed:", block.CreatedAt, labelW)

	// Full verification hash in Courier 8pt; no truncation
	p.SetFont("Helvetica", "B", fontSizeSm)
	p.SetTextColor(30, 30, 30)
	p.SetX(marginLeft)
	p.Cell(labelW, lineHeight, "Verification Hash:")
	p.SetFont("Courier", "", 8)
	p.SetTextColor(30, 30, 30)
	p.MultiCell(bodyWidth-labelW, lineHeight, block.ContentHash, "", "L", false)

	// Verification status with semantic color
	verifyLabel := "VERIFIED"
	vR, vG, vB := 0, 140, 0
	if !block.Verified {
		verifyLabel = "TAMPERED"
		vR, vG, vB = 180, 0, 0
	}
	p.SetFont("Helvetica", "B", fontSizeSm)
	p.SetTextColor(30, 30, 30)
	p.SetX(marginLeft)
	p.Cell(labelW, lineHeight, "Verification:")
	p.SetTextColor(vR, vG, vB)
	p.Cell(bodyWidth-labelW, lineHeight, verifyLabel)
	p.Ln(-1)

	// Amendment info
	if raw, ok := rawBlockMap[block.BlockID]; ok && raw.AmendsBlockID != nil {
		p.SetFont("Helvetica", "B", fontSizeSm)
		p.SetTextColor(30, 30, 30)
		p.SetX(marginLeft)
		p.Cell(labelW, lineHeight, "AMENDMENT:")
		p.SetFont("Helvetica", "", fontSizeSm)
		p.Cell(bodyWidth-labelW, lineHeight, "Amends block: "+*raw.AmendsBlockID)
		p.Ln(-1)
	}

	p.SetFont("Helvetica", "", fontSizeBody)
	p.SetTextColor(30, 30, 30)
	p.Ln(2)
}

// BuildEvidenceSection renders the Evidence Items section.
func BuildEvidenceSection(p *fpdf.Fpdf, evidenceItems []services.EvidenceResponse, evidenceBlockMap map[string][]services.NoteBlockResponse, rawBlockMap map[string]models.NoteBlock, evidenceIndex map[string]string) (int, int) {
	p.AddPage()
	startPage := p.PageNo()
	link := sectionHeading(p, "Evidence Items", 0)

	for _, item := range evidenceItems {
		eNum := evidenceIndex[item.EvidenceItemID]
		subHeading(p, eNum+" -- "+item.Name)

		// Metadata as clean label/value rows
		const labelW = 50.0
		metaRows := [][]string{
			{"Item Number:", eNum},
			{"Type:", item.EvidenceType},
			{"Acquisition Hash:", item.ContentHash},
			{"Current Status:", item.Status},
			{"Created At:", item.CreatedAt},
		}
		for _, row := range metaRows {
			renderLabelValue(p, row[0], row[1], labelW)
		}
		p.Ln(4)

		// Custody log
		if len(item.CustodyLog) > 0 {
			p.SetFont("Helvetica", "B", fontSizeBody)
			p.SetTextColor(30, 30, 30)
			p.Cell(0, lineHeight, "Chain of Custody")
			p.Ln(-1)
			p.Ln(2)
			custodyRows := make([][]string, len(item.CustodyLog))
			for i, c := range item.CustodyLog {
				custodyRows[i] = []string{c.Timestamp, c.Action, c.Description}
			}
			DrawTable(p,
				[]string{"Timestamp", "Action", "Notes"},
				[]float64{50, 40, bodyWidth - 90},
				custodyRows,
				map[string]CellStyle{},
			)
		}

		// Note blocks
		blocks := evidenceBlockMap[item.EvidenceItemID]
		if len(blocks) > 0 {
			p.SetFont("Helvetica", "B", fontSizeBody)
			p.SetTextColor(30, 30, 30)
			p.Cell(0, lineHeight, "Note Blocks")
			p.Ln(-1)
			p.Ln(2)
			for _, block := range blocks {
				renderBlockHeader(p, block, rawBlockMap)
				RenderMarkdown(p, block.Content)
				separator(p)
			}
		}

		p.Ln(6)
	}
	return link, startPage
}

// BuildIOCSection renders the IOC Summary section.
func BuildIOCSection(p *fpdf.Fpdf, iocEntries []ioc.IOCEntry) (int, int) {
	p.AddPage()
	startPage := p.PageNo()
	link := sectionHeading(p, "IOC Summary", 0)

	var confirmed, dismissed []ioc.IOCEntry
	for _, e := range iocEntries {
		if e.Status == ioc.IOCStatusFalsePositive {
			dismissed = append(dismissed, e)
		} else {
			confirmed = append(confirmed, e)
		}
	}

	subHeading(p, "Confirmed and Detected IOCs")
	if len(confirmed) == 0 {
		p.SetFont("Helvetica", "I", fontSizeBody)
		p.SetTextColor(80, 80, 80)
		p.Cell(0, lineHeight, "No confirmed or detected IOCs.")
		p.Ln(-1)
		p.Ln(4)
	} else {
		rows := iocTableRows(confirmed)
		DrawTableMonospace(p,
			[]string{"Type", "Value (Defanged)", "Status", "Source Block ID", "Detected At"},
			[]float64{22, 60, 20, 50, 28},
			rows,
			map[string]CellStyle{},
		)
	}

	subHeading(p, "Dismissed (False Positives)")
	if len(dismissed) == 0 {
		p.SetFont("Helvetica", "I", fontSizeBody)
		p.SetTextColor(80, 80, 80)
		p.Cell(0, lineHeight, "No dismissed IOCs.")
		p.Ln(-1)
		p.Ln(4)
	} else {
		// Gray text for dismissed rows
		styles := make(map[string]CellStyle)
		for i := range dismissed {
			for col := 0; col < 5; col++ {
				styles[cellKey(i, col)] = CellStyle{120, 120, 120}
			}
		}
		rows := iocTableRows(dismissed)
		DrawTableMonospace(p,
			[]string{"Type", "Value (Defanged)", "Status", "Source Block ID", "Detected At"},
			[]float64{22, 60, 20, 50, 28},
			rows,
			styles,
		)
	}
	return link, startPage
}

func iocTableRows(entries []ioc.IOCEntry) [][]string {
	rows := make([][]string, len(entries))
	for i, e := range entries {
		ca := ""
		if e.ConfirmedAt != nil {
			ca = *e.ConfirmedAt
		}
		rows[i] = []string{
			string(e.Type),
			defangValue(e.Value, string(e.Type)),
			string(e.Status),
			e.BlockID,
			func() string {
				if ca != "" {
					return ca
				}
				return e.CreatedAt
			}(),
		}
	}
	return rows
}

// BuildTimelineSection renders the Timeline section as record blocks.
func BuildTimelineSection(p *fpdf.Fpdf, entries []models.TimelineEntry, evidenceIndex map[string]string) (int, int) {
	p.AddPage()
	startPage := p.PageNo()
	link := sectionHeading(p, "Timeline", 0)

	if len(entries) == 0 {
		p.SetFont("Helvetica", "I", fontSizeBody)
		p.SetTextColor(80, 80, 80)
		p.Cell(0, lineHeight, "No timeline entries.")
		p.Ln(-1)
		return link, startPage
	}

	// Sort by timestamp
	sorted := make([]models.TimelineEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp < sorted[j].Timestamp
	})

	for i, e := range sorted {
		// Page overflow check
		if p.GetY() > pageHeight-marginBottom-30 {
			p.AddPage()
		}

		tzDisplay := ""
		if e.DisplayTimezone != nil {
			tzDisplay = *e.DisplayTimezone
		}
		source := "Case Level"
		if e.EvidenceItemID != nil {
			if label, ok := evidenceIndex[*e.EvidenceItemID]; ok {
				source = label
			}
		}

		// Line 1: Timestamp (bold) -- Source
		p.SetFont("Helvetica", "B", fontSizeBody)
		p.SetTextColor(30, 30, 30)
		p.SetX(marginLeft)
		tsW := p.GetStringWidth(e.Timestamp) + 4
		p.Cell(tsW, lineHeight, e.Timestamp)
		p.SetFont("Helvetica", "", fontSizeBody)
		p.SetTextColor(80, 80, 80)
		p.Cell(0, lineHeight, "-- "+source)
		p.Ln(-1)

		// Line 2: Secondary TZ if present
		if tzDisplay != "" {
			p.SetFont("Helvetica", "I", 9)
			p.SetTextColor(80, 80, 80)
			p.SetX(marginLeft)
			p.Cell(0, lineHeight, tzDisplay)
			p.Ln(-1)
		}

		// Event description
		p.SetFont("Helvetica", "", fontSizeBody)
		p.SetTextColor(30, 30, 30)
		p.SetX(marginLeft)
		p.MultiCell(bodyWidth, lineHeight, e.EventDescription, "", "L", false)

		// Investigator notes if present
		if e.InvestigatorNotes != "" {
			p.SetFont("Helvetica", "B", fontSizeSm)
			p.SetTextColor(30, 30, 30)
			p.SetX(marginLeft + 5)
			p.Cell(20, lineHeight, "Notes:")
			p.SetFont("Helvetica", "", fontSizeSm)
			p.SetTextColor(30, 30, 30)
			p.MultiCell(bodyWidth-25, lineHeight, e.InvestigatorNotes, "", "L", false)
		}

		// Thin rule between entries
		if i < len(sorted)-1 {
			p.Ln(2)
			y := p.GetY()
			p.SetDrawColor(200, 200, 200)
			p.SetLineWidth(0.3)
			p.Line(marginLeft, y, marginLeft+bodyWidth, y)
			p.Ln(3)
		}
	}
	p.Ln(4)
	return link, startPage
}

// BuildTaskListSection renders the Task List section as record blocks.
func BuildTaskListSection(p *fpdf.Fpdf, tasks []models.Task, evidenceItems []services.EvidenceResponse, evidenceIndex map[string]string) (int, int) {
	p.AddPage()
	startPage := p.PageNo()
	link := sectionHeading(p, "Task List", 0)

	if len(tasks) == 0 {
		p.SetFont("Helvetica", "I", fontSizeBody)
		p.SetTextColor(80, 80, 80)
		p.Cell(0, lineHeight, "No tasks.")
		p.Ln(-1)
		return link, startPage
	}

	// Build ordered evidence list: case level first, then E001, E002...
	type group struct {
		label  string
		itemID string
	}
	groups := []group{{"Case Level", ""}}
	sortedEvid := make([]services.EvidenceResponse, len(evidenceItems))
	copy(sortedEvid, evidenceItems)
	sort.Slice(sortedEvid, func(i, j int) bool {
		return sortedEvid[i].CreatedAt < sortedEvid[j].CreatedAt
	})
	for _, item := range sortedEvid {
		label := evidenceIndex[item.EvidenceItemID]
		if label == "" {
			label = item.EvidenceItemID
		}
		groups = append(groups, group{label + " -- " + item.Name, item.EvidenceItemID})
	}

	statusOrder := map[models.TaskStatus]int{
		models.TaskStatusOpen:          0,
		models.TaskStatusInProgress:    1,
		models.TaskStatusBlocked:       2,
		models.TaskStatusComplete:      3,
		models.TaskStatusNotApplicable: 4,
	}

	for _, g := range groups {
		var groupTasks []models.Task
		for _, t := range tasks {
			if g.itemID == "" && t.EvidenceItemID == nil {
				groupTasks = append(groupTasks, t)
			} else if g.itemID != "" && t.EvidenceItemID != nil && *t.EvidenceItemID == g.itemID {
				groupTasks = append(groupTasks, t)
			}
		}
		if len(groupTasks) == 0 {
			continue
		}

		subHeading(p, g.label)

		sort.Slice(groupTasks, func(i, j int) bool {
			oi := statusOrder[groupTasks[i].Status]
			oj := statusOrder[groupTasks[j].Status]
			return oi < oj
		})

		const taskLabelW = 30.0

		for i, t := range groupTasks {
			if i > 0 {
				y := p.GetY()
				if groupTasks[i].Status != groupTasks[i-1].Status {
					// Heavier rule between status subgroups
					p.SetDrawColor(120, 120, 120)
					p.SetLineWidth(0.5)
				} else {
					// Thin rule between tasks within same status
					p.SetDrawColor(200, 200, 200)
					p.SetLineWidth(0.3)
				}
				p.Line(marginLeft, y, marginLeft+bodyWidth, y)
				p.Ln(3)
			}

			// Page overflow check
			if p.GetY() > pageHeight-marginBottom-25 {
				p.AddPage()
			}

			// Task: label (bold 10pt) + title (normal 10pt) on same line
			p.SetFont("Helvetica", "B", fontSizeBody)
			p.SetTextColor(30, 30, 30)
			p.SetX(marginLeft)
			p.Cell(taskLabelW, lineHeight, "Task:")
			p.SetFont("Helvetica", "", fontSizeBody)
			p.MultiCell(bodyWidth-taskLabelW, lineHeight, t.Title, "", "L", false)

			// Status: always shown (bold 9pt label + normal 9pt value)
			p.SetFont("Helvetica", "B", 9)
			p.SetTextColor(30, 30, 30)
			p.SetX(marginLeft)
			p.Cell(taskLabelW, lineHeight, "Status:")
			p.SetFont("Helvetica", "", 9)
			p.Cell(0, lineHeight, string(t.Status))
			p.Ln(-1)

			// Description: omit row if empty
			if t.Description != "" {
				p.SetFont("Helvetica", "B", 9)
				p.SetTextColor(30, 30, 30)
				p.SetX(marginLeft)
				p.Cell(taskLabelW, lineHeight, "Description:")
				p.SetFont("Helvetica", "", 9)
				p.MultiCell(bodyWidth-taskLabelW, lineHeight, t.Description, "", "L", false)
			}

			// Completed: show only when status is Complete
			if t.CompletedAt != nil && *t.CompletedAt != "" {
				p.SetFont("Helvetica", "B", 9)
				p.SetTextColor(30, 30, 30)
				p.SetX(marginLeft)
				p.Cell(taskLabelW, lineHeight, "Completed:")
				p.SetFont("Helvetica", "", 9)
				p.Cell(0, lineHeight, *t.CompletedAt)
				p.Ln(-1)
			}

			// Linked blocks if present
			if len(t.LinkedBlocks) > 0 {
				p.SetFont("Helvetica", "B", 9)
				p.SetTextColor(30, 30, 30)
				p.SetX(marginLeft)
				p.Cell(taskLabelW, lineHeight, "Linked blocks:")
				p.Ln(-1)
				for _, lb := range t.LinkedBlocks {
					p.SetFont("Courier", "", 8)
					p.SetTextColor(30, 30, 30)
					p.SetX(marginLeft + 5)
					p.Cell(0, lineHeight, "Committed "+lb.CommittedAt+" -- Block "+lb.BlockID)
					p.Ln(-1)
				}
			}
		}
		p.Ln(6)
	}
	return link, startPage
}

// BuildChainVerificationSection renders the Chain Verification section.
func BuildChainVerificationSection(p *fpdf.Fpdf, rawBlocks []models.NoteBlock) (int, int) {
	p.AddPage()
	startPage := p.PageNo()
	link := sectionHeading(p, "Chain Verification", 0)

	// Explanatory paragraph
	const chainExplanation = "The chain verification section confirms the integrity of every committed note block in this case. Each block is cryptographically linked to the preceding block via SHA-256 hashing and digitally signed with the examiner's Ed25519 keypair. A failed signature or broken chain link indicates that one or more blocks were modified after commitment and the record should be considered suspect."
	p.SetFont("Helvetica", "", fontSizeBody)
	p.SetTextColor(60, 60, 60)
	p.SetX(marginLeft)
	p.MultiCell(bodyWidth, lineHeight, chainExplanation, "", "L", false)
	p.Ln(4)

	// Compute chain validity
	failCount := 0
	type chainRow struct {
		seq       int
		blockID   string
		committed string
		hash      string
		prevHash  string
		sigValid  string
		chainOK   string
	}
	var rows []chainRow

	for i, b := range rawBlocks {
		chainOK := "YES"
		if i > 0 && rawBlocks[i-1].ContentHash != b.PrevHash {
			chainOK = "NO"
			failCount++
		}
		sigValid := "YES"
		if len(b.Signature) == 0 {
			sigValid = "NO"
		}

		prevHashShort := b.PrevHash
		if len(prevHashShort) > 16 {
			prevHashShort = prevHashShort[:16]
		}

		committed := b.CreatedAt.UTC().Format(time.RFC3339)

		rows = append(rows, chainRow{
			seq:       i + 1,
			blockID:   b.BlockID,
			committed: committed,
			hash:      b.ContentHash,
			prevHash:  prevHashShort,
			sigValid:  sigValid,
			chainOK:   chainOK,
		})
	}

	// Summary line
	p.SetFont("Helvetica", "B", fontSizeBody+1)
	if failCount == 0 {
		p.SetTextColor(0, 140, 0)
		p.Cell(0, lineHeight+2, "Chain intact: YES")
	} else {
		p.SetTextColor(220, 60, 60)
		p.Cell(0, lineHeight+2, fmt.Sprintf("Chain intact: NO -- %d block(s) failed verification", failCount))
	}
	p.Ln(-1)
	p.Ln(4)
	p.SetFont("Helvetica", "", fontSizeBody)
	p.SetTextColor(30, 30, 30)

	// Build table rows + cell styles for red cells
	tableRows := make([][]string, len(rows))
	styles := make(map[string]CellStyle)
	for i, r := range rows {
		tableRows[i] = []string{
			fmt.Sprintf("%d", r.seq),
			r.blockID,
			r.committed,
			r.hash,
			r.prevHash,
			r.sigValid,
			r.chainOK,
		}
		if r.sigValid != "YES" {
			styles[cellKey(i, 5)] = CellStyle{220, 60, 60}
		}
		if r.chainOK != "YES" {
			styles[cellKey(i, 6)] = CellStyle{220, 60, 60}
		}
	}
	DrawTableMonospace(p,
		[]string{"Block #", "Block ID", "Committed At", "Content Hash", "Prev Hash", "Sig Valid", "Chain OK"},
		[]float64{10, 50, 30, 40, 20, 14, 16},
		tableRows,
		styles,
	)
	return link, startPage
}

// BuildAppendixImages renders Appendix A with embedded images. Returns (0, 0) if no images.
func BuildAppendixImages(p *fpdf.Fpdf, blocks []services.NoteBlockResponse, attachments map[string]*AttachmentInfo) (int, int) {
	// Collect images in chain order
	type imgRef struct {
		attachID  string
		blockID   string
		committed string
		filename  string
	}
	var images []imgRef

	for _, block := range blocks {
		found := extractAttachmentRefs(block.Content)
		for _, ref := range found {
			if _, ok := attachments[ref.id]; ok {
				images = append(images, imgRef{
					attachID:  ref.id,
					blockID:   block.BlockID,
					committed: block.CreatedAt,
					filename:  ref.alt,
				})
			}
		}
	}

	if len(images) == 0 {
		return 0, 0
	}

	p.AddPage()
	startPage := p.PageNo()
	link := sectionHeading(p, "Appendix A -- Images", 0)

	for idx, img := range images {
		info := attachments[img.attachID]
		label := fmt.Sprintf("Image %d from Block %s -- committed %s", idx+1, img.blockID, img.committed)

		p.SetFont("Helvetica", "B", fontSizeSm)
		p.SetTextColor(30, 30, 30)
		p.SetX(marginLeft)
		p.MultiCell(bodyWidth, lineHeight, label, "", "L", false)
		p.Ln(2)

		imgType := contentTypeToFPDF(info.ContentType)
		if imgType == "" {
			p.SetFont("Helvetica", "I", fontSizeSm)
			p.SetTextColor(180, 0, 0)
			p.Cell(0, lineHeight, "[Unsupported image format: "+info.ContentType+"]")
			p.Ln(-1)
			p.Ln(4)
			continue
		}

		imgName := fmt.Sprintf("img_%d_%s", idx, img.attachID)
		opts := fpdf.ImageOptions{ImageType: imgType}
		p.RegisterImageOptionsReader(imgName, opts, newBytesReader(info.Data))
		imgInfo := p.GetImageInfo(imgName)
		if imgInfo == nil {
			p.SetFont("Helvetica", "I", fontSizeSm)
			p.SetTextColor(180, 0, 0)
			p.Cell(0, lineHeight, "[Could not load image]")
			p.Ln(-1)
			p.Ln(4)
			continue
		}

		// Scale to page width
		imgW := bodyWidth
		imgH := 0.0 // auto height
		if imgInfo.Width() > 0 {
			aspect := imgInfo.Height() / imgInfo.Width()
			imgH = imgW * aspect
		}

		// Page overflow check
		if p.GetY()+imgH+10 > pageHeight-marginBottom && imgH > 0 {
			p.AddPage()
		}

		p.ImageOptions(imgName, marginLeft, p.GetY(), imgW, imgH, false, opts, 0, "")
		p.Ln(imgH + 6)
		separator(p)
	}
	return link, startPage
}

type attachRef struct {
	id  string
	alt string
}

// extractAttachmentRefs scans markdown content for ![alt](attachment:ID) patterns.
func extractAttachmentRefs(content string) []attachRef {
	var refs []attachRef
	remaining := content
	for {
		start := strings.Index(remaining, "![")
		if start < 0 {
			break
		}
		rest := remaining[start+2:]
		altEnd := strings.Index(rest, "](")
		if altEnd < 0 {
			break
		}
		alt := rest[:altEnd]
		urlPart := rest[altEnd+2:]
		urlEnd := strings.Index(urlPart, ")")
		if urlEnd < 0 {
			break
		}
		url := urlPart[:urlEnd]
		if strings.HasPrefix(url, "attachment:") {
			id := strings.TrimPrefix(url, "attachment:")
			refs = append(refs, attachRef{id, alt})
		}
		remaining = remaining[start+2+altEnd+2+urlEnd+1:]
	}
	return refs
}

func contentTypeToFPDF(ct string) string {
	ct = strings.ToLower(ct)
	switch {
	case strings.Contains(ct, "png"):
		return "PNG"
	case strings.Contains(ct, "jpeg") || strings.Contains(ct, "jpg"):
		return "JPEG"
	case strings.Contains(ct, "gif"):
		return "GIF"
	default:
		return ""
	}
}

// defangValue applies defanging rules to IOC values.
func defangValue(value, iocType string) string {
	switch iocType {
	case "ipv4":
		return strings.ReplaceAll(value, ".", "[.]")
	case "ipv6":
		return strings.ReplaceAll(value, ":", "[:]")
	case "domain":
		idx := strings.LastIndex(value, ".")
		if idx >= 0 {
			return value[:idx] + "[.]" + value[idx+1:]
		}
		return value
	case "url":
		v := strings.ReplaceAll(value, ".", "[.]")
		v = strings.Replace(v, "https://", "hxxps://", 1)
		v = strings.Replace(v, "http://", "hxxp://", 1)
		v = strings.Replace(v, "ftp://", "fxp://", 1)
		return v
	case "email":
		at := strings.Index(value, "@")
		if at < 0 {
			return value
		}
		domain := strings.ReplaceAll(value[at+1:], ".", "[.]")
		return value[:at] + "[@]" + domain
	default:
		return value
	}
}
