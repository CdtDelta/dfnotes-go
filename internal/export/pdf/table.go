package pdf

import (
	"fmt"

	"github.com/go-pdf/fpdf"
)

// CellStyle overrides text color for a specific cell (row,col 0-indexed).
type CellStyle struct {
	TextR, TextG, TextB int
}

// DrawTable renders a bordered table with alternating row shading and page break detection.
// widths must have the same length as headers. cellStyles keys are "row,col" strings.
func DrawTable(p *fpdf.Fpdf, headers []string, widths []float64, rows [][]string, cellStyles map[string]CellStyle) {
	if len(headers) == 0 {
		return
	}
	drawHeader := func() {
		p.SetFont("Helvetica", "B", fontSizeBody)
		p.SetFillColor(220, 220, 220)
		p.SetTextColor(30, 30, 30)
		p.SetDrawColor(180, 180, 180)
		p.SetLineWidth(0.3)
		for i, h := range headers {
			w := colWidth(widths, i)
			p.CellFormat(w, lineHeight, h, "1", 0, "L", true, 0, "")
		}
		p.Ln(-1)
		p.SetFont("Helvetica", "", fontSizeBody)
	}

	drawHeader()

	for rowIdx, row := range rows {
		// Estimate row height from the tallest cell.
		rowH := lineHeight
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			lines := countLines(p, cell, widths[i])
			if h := float64(lines) * lineHeight; h > rowH {
				rowH = h
			}
		}

		// Page overflow: start new page and re-draw header.
		if p.GetY()+rowH > pageHeight-marginBottom-5 {
			p.AddPage()
			drawHeader()
		}

		startY := p.GetY()

		if rowIdx%2 == 0 {
			p.SetFillColor(255, 255, 255)
		} else {
			p.SetFillColor(245, 245, 245)
		}

		for colIdx, cell := range row {
			if colIdx >= len(widths) {
				break
			}
			x := marginLeft + colOffset(widths, colIdx)
			p.SetXY(x, startY)

			if cs, ok := cellStyles[cellKey(rowIdx, colIdx)]; ok {
				p.SetTextColor(cs.TextR, cs.TextG, cs.TextB)
			} else {
				p.SetTextColor(30, 30, 30)
			}
			p.MultiCell(widths[colIdx], lineHeight, cell, "1", "L", true)
		}

		// Advance past the tallest cell in this row.
		p.SetXY(marginLeft, startY+rowH)
	}

	p.SetTextColor(30, 30, 30)
	p.SetFillColor(255, 255, 255)
	p.Ln(4)
}

// DrawTableBorderless renders a table with no cell borders, a single underline below
// the header row (RGB 30,30,30, 0.5pt), gray header fill, and alternating row shading.
func DrawTableBorderless(p *fpdf.Fpdf, headers []string, widths []float64, rows [][]string, cellStyles map[string]CellStyle) {
	if len(headers) == 0 {
		return
	}
	drawHeader := func() {
		p.SetFont("Helvetica", "B", fontSizeBody)
		p.SetFillColor(220, 220, 220)
		p.SetTextColor(30, 30, 30)
		for i, h := range headers {
			w := colWidth(widths, i)
			p.CellFormat(w, lineHeight, h, "0", 0, "L", true, 0, "")
		}
		p.Ln(-1)
		y := p.GetY()
		p.SetDrawColor(30, 30, 30)
		p.SetLineWidth(0.5)
		p.Line(marginLeft, y, marginLeft+bodyWidth, y)
		p.SetFont("Helvetica", "", fontSizeBody)
	}

	drawHeader()

	for rowIdx, row := range rows {
		rowH := lineHeight
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			lines := countLines(p, cell, widths[i])
			if h := float64(lines) * lineHeight; h > rowH {
				rowH = h
			}
		}

		if p.GetY()+rowH > pageHeight-marginBottom-5 {
			p.AddPage()
			drawHeader()
		}

		startY := p.GetY()

		if rowIdx%2 == 0 {
			p.SetFillColor(255, 255, 255)
		} else {
			p.SetFillColor(245, 245, 245)
		}

		for colIdx, cell := range row {
			if colIdx >= len(widths) {
				break
			}
			x := marginLeft + colOffset(widths, colIdx)
			p.SetXY(x, startY)

			if cs, ok := cellStyles[cellKey(rowIdx, colIdx)]; ok {
				p.SetTextColor(cs.TextR, cs.TextG, cs.TextB)
			} else {
				p.SetTextColor(30, 30, 30)
			}
			p.MultiCell(widths[colIdx], lineHeight, cell, "0", "L", true)
		}

		p.SetXY(marginLeft, startY+rowH)
	}

	p.SetTextColor(30, 30, 30)
	p.SetFillColor(255, 255, 255)
	p.Ln(4)
}

// DrawTableMonospace renders the same layout as DrawTableBorderless but uses
// Helvetica Bold 8pt for headers and Courier 8pt for all body cells.
// Intended for identifier-heavy content (hashes, IPs, block IDs).
func DrawTableMonospace(p *fpdf.Fpdf, headers []string, widths []float64, rows [][]string, cellStyles map[string]CellStyle) {
	if len(headers) == 0 {
		return
	}
	drawHeader := func() {
		p.SetFont("Helvetica", "B", 8)
		p.SetFillColor(220, 220, 220)
		p.SetTextColor(30, 30, 30)
		for i, h := range headers {
			w := colWidth(widths, i)
			p.CellFormat(w, lineHeight, h, "0", 0, "L", true, 0, "")
		}
		p.Ln(-1)
		y := p.GetY()
		p.SetDrawColor(30, 30, 30)
		p.SetLineWidth(0.5)
		p.Line(marginLeft, y, marginLeft+bodyWidth, y)
	}

	drawHeader()

	for rowIdx, row := range rows {
		// Measure row height using Courier 8pt so wrapping estimates are accurate.
		p.SetFont("Courier", "", 8)
		rowH := lineHeight
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			lines := countLines(p, cell, widths[i])
			if h := float64(lines) * lineHeight; h > rowH {
				rowH = h
			}
		}

		if p.GetY()+rowH > pageHeight-marginBottom-5 {
			p.AddPage()
			drawHeader()
		}

		startY := p.GetY()

		if rowIdx%2 == 0 {
			p.SetFillColor(255, 255, 255)
		} else {
			p.SetFillColor(245, 245, 245)
		}

		p.SetFont("Courier", "", 8)
		for colIdx, cell := range row {
			if colIdx >= len(widths) {
				break
			}
			x := marginLeft + colOffset(widths, colIdx)
			p.SetXY(x, startY)

			if cs, ok := cellStyles[cellKey(rowIdx, colIdx)]; ok {
				p.SetTextColor(cs.TextR, cs.TextG, cs.TextB)
			} else {
				p.SetTextColor(30, 30, 30)
			}
			p.MultiCell(widths[colIdx], lineHeight, cell, "0", "L", true)
		}

		p.SetXY(marginLeft, startY+rowH)
	}

	p.SetTextColor(30, 30, 30)
	p.SetFillColor(255, 255, 255)
	p.SetFont("Helvetica", "", fontSizeBody)
	p.Ln(4)
}

func cellKey(row, col int) string {
	return fmt.Sprintf("%d,%d", row, col)
}

func colWidth(widths []float64, i int) float64 {
	if i < len(widths) {
		return widths[i]
	}
	return 20
}

func colOffset(widths []float64, col int) float64 {
	x := 0.0
	for i := 0; i < col && i < len(widths); i++ {
		x += widths[i]
	}
	return x
}

func countLines(p *fpdf.Fpdf, text string, width float64) int {
	if width <= 0 || text == "" {
		return 1
	}
	lines := p.SplitLines([]byte(text), width)
	if len(lines) == 0 {
		return 1
	}
	return len(lines)
}
