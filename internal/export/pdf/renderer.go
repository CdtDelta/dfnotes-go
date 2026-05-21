package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmtext "github.com/yuin/goldmark/text"
)

// RenderMarkdown parses md as Markdown and draws it into p starting at the current position.
func RenderMarkdown(p *fpdf.Fpdf, md string) {
	if strings.TrimSpace(md) == "" {
		return
	}
	src := []byte(md)
	gm := goldmark.New(goldmark.WithExtensions(extension.Table))
	reader := gmtext.NewReader(src)
	doc := gm.Parser().Parse(reader)
	r := &mdRenderer{pdf: p, src: src}
	r.walkChildren(doc)
}

type mdRenderer struct {
	pdf      *fpdf.Fpdf
	src      []byte
	listKind []bool // true = ordered; stack for nested lists
	listNum  []int
}

func (r *mdRenderer) walkChildren(node ast.Node) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.renderNode(child)
	}
}

func (r *mdRenderer) renderNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.Heading:
		r.renderHeading(n)
	case *ast.Paragraph:
		r.renderParagraph(n)
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		r.renderCodeBlock(node)
	case *ast.Blockquote:
		r.renderBlockquote(n)
	case *ast.List:
		r.renderList(n)
	case *ast.ListItem:
		r.renderListItem(n)
	case *ast.ThematicBreak:
		r.renderHR()
	case *ast.HTMLBlock:
		// skip HTML
	default:
		// fallthrough: walk children
		r.walkChildren(node)
	}
}

func (r *mdRenderer) renderHeading(n *ast.Heading) {
	text := r.inlineText(n)
	size := fontSizeH3
	switch n.Level {
	case 1:
		size = fontSizeH1
	case 2:
		size = fontSizeH2
	}
	r.pdf.Ln(3)
	r.pdf.SetFont("Helvetica", "B", size)
	r.pdf.SetTextColor(30, 30, 30)
	r.pdf.MultiCell(bodyWidth, lineHeight, text, "", "L", false)
	r.pdf.Ln(2)
	r.pdf.SetFont("Helvetica", "", fontSizeBody)
}

func (r *mdRenderer) renderParagraph(n *ast.Paragraph) {
	r.renderInlineNodes(n, bodyWidth)
	r.pdf.Ln(3)
}

func (r *mdRenderer) renderInlineNodes(parent ast.Node, width float64) {
	var buf strings.Builder
	flush := func(style string) {
		t := buf.String()
		if t == "" {
			return
		}
		buf.Reset()
		r.pdf.SetFont("Helvetica", style, fontSizeBody)
		r.pdf.SetTextColor(30, 30, 30)
		r.pdf.MultiCell(width, lineHeight, t, "", "L", false)
	}

	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.Text:
			seg := c.Segment
			buf.Write(seg.Value(r.src))
			if c.SoftLineBreak() {
				buf.WriteByte(' ')
			}
			if c.HardLineBreak() {
				flush("")
				r.pdf.Ln(lineHeight)
			}
		case *ast.String:
			buf.Write(c.Value)
		case *ast.Emphasis:
			flush("")
			txt := r.inlineText(c)
			style := "I"
			if c.Level == 2 {
				style = "B"
			}
			r.pdf.SetFont("Helvetica", style, fontSizeBody)
			r.pdf.SetTextColor(30, 30, 30)
			r.pdf.MultiCell(width, lineHeight, txt, "", "L", false)
			r.pdf.SetFont("Helvetica", "", fontSizeBody)
		case *ast.CodeSpan:
			flush("")
			txt := string(c.Text(r.src))
			r.pdf.SetFont("Courier", "", fontSizeBody)
			r.pdf.SetTextColor(30, 100, 30)
			r.pdf.MultiCell(width, lineHeight, txt, "", "L", false)
			r.pdf.SetFont("Helvetica", "", fontSizeBody)
			r.pdf.SetTextColor(30, 30, 30)
		case *ast.Link:
			flush("")
			linkText := r.inlineText(c)
			dest := string(c.Destination)
			r.pdf.SetFont("Helvetica", "", fontSizeBody)
			r.pdf.SetTextColor(30, 30, 30)
			r.pdf.MultiCell(width, lineHeight, linkText+" ("+dest+")", "", "L", false)
		case *ast.Image:
			flush("")
			// Images rendered as placeholder; actual image in Appendix A
			filename := string(c.Destination)
			if strings.HasPrefix(filename, "attachment:") {
				filename = strings.TrimPrefix(filename, "attachment:")
			}
			r.pdf.SetFont("Helvetica", "I", fontSizeSm)
			r.pdf.SetTextColor(100, 100, 100)
			r.pdf.MultiCell(width, lineHeight, "[Image: "+filename+" -- see Appendix A]", "", "L", false)
			r.pdf.SetFont("Helvetica", "", fontSizeBody)
			r.pdf.SetTextColor(30, 30, 30)
		case *ast.AutoLink:
			flush("")
			url := string(c.URL(r.src))
			r.pdf.SetFont("Helvetica", "", fontSizeBody)
			r.pdf.SetTextColor(30, 30, 30)
			r.pdf.MultiCell(width, lineHeight, url, "", "L", false)
		default:
			// Collect raw text from unknown inline nodes
			if child.HasChildren() {
				buf.WriteString(r.inlineText(child))
			}
		}
	}
	flush("")
}

func (r *mdRenderer) renderCodeBlock(node ast.Node) {
	var code []byte
	switch n := node.(type) {
	case *ast.FencedCodeBlock:
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			code = append(code, line.Value(r.src)...)
		}
	case *ast.CodeBlock:
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			code = append(code, line.Value(r.src)...)
		}
	}
	text := strings.TrimRight(string(code), "\n")
	if text == "" {
		return
	}

	r.pdf.Ln(2)
	// Light gray background rect
	x := marginLeft
	y := r.pdf.GetY()
	lines := strings.Split(text, "\n")
	h := float64(len(lines)) * lineHeight
	r.pdf.SetFillColor(240, 240, 240)
	r.pdf.Rect(x, y, bodyWidth, h+4, "F")
	r.pdf.SetXY(x+2, y+2)
	r.pdf.SetFont("Courier", "", fontSizeCode)
	r.pdf.SetTextColor(30, 80, 30)
	r.pdf.MultiCell(bodyWidth-4, lineHeight, text, "", "L", false)
	r.pdf.SetFont("Helvetica", "", fontSizeBody)
	r.pdf.SetTextColor(30, 30, 30)
	r.pdf.Ln(3)
}

func (r *mdRenderer) renderBlockquote(n *ast.Blockquote) {
	r.pdf.Ln(2)
	x := marginLeft + 4
	y := r.pdf.GetY()
	// Draw left border
	r.pdf.SetDrawColor(180, 180, 180)
	r.pdf.Line(marginLeft+1, y, marginLeft+1, y+20)
	r.pdf.SetXY(x, y)
	r.pdf.SetFont("Helvetica", "I", fontSizeBody)
	r.pdf.SetTextColor(80, 80, 80)
	r.walkChildren(n)
	r.pdf.SetFont("Helvetica", "", fontSizeBody)
	r.pdf.SetTextColor(30, 30, 30)
	r.pdf.Ln(2)
}

func (r *mdRenderer) renderList(n *ast.List) {
	r.listKind = append(r.listKind, n.IsOrdered())
	r.listNum = append(r.listNum, n.Start)
	r.walkChildren(n)
	r.listKind = r.listKind[:len(r.listKind)-1]
	r.listNum = r.listNum[:len(r.listNum)-1]
}

func (r *mdRenderer) renderListItem(n *ast.ListItem) {
	depth := len(r.listKind)
	indent := marginLeft + float64(depth)*6
	r.pdf.SetX(indent)
	r.pdf.SetFont("Helvetica", "", fontSizeBody)
	r.pdf.SetTextColor(30, 30, 30)

	var prefix string
	if depth > 0 && r.listKind[depth-1] {
		num := r.listNum[depth-1]
		prefix = strings.Repeat(" ", (depth-1)*2) + strings.TrimSpace(fmt.Sprintf("%d. ", num))
		r.listNum[depth-1]++
	} else {
		prefix = strings.Repeat(" ", (depth-1)*2) + "• "
	}

	// Gather item text
	var buf bytes.Buffer
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.TextBlock, *ast.Paragraph:
			buf.WriteString(r.inlineText(c))
		default:
			buf.WriteString(r.inlineText(child))
		}
	}

	r.pdf.SetX(indent)
	itemWidth := bodyWidth - (indent - marginLeft)
	r.pdf.MultiCell(itemWidth, lineHeight, prefix+buf.String(), "", "L", false)
}

func (r *mdRenderer) renderHR() {
	r.pdf.Ln(3)
	y := r.pdf.GetY()
	r.pdf.SetDrawColor(180, 180, 180)
	r.pdf.Line(marginLeft, y, marginLeft+bodyWidth, y)
	r.pdf.Ln(3)
}

// inlineText extracts plain text (no styling) from an inline node subtree.
func (r *mdRenderer) inlineText(node ast.Node) string {
	var buf strings.Builder
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch c := n.(type) {
		case *ast.Text:
			buf.Write(c.Segment.Value(r.src))
			if c.SoftLineBreak() {
				buf.WriteByte(' ')
			}
		case *ast.String:
			buf.Write(c.Value)
		}
		return ast.WalkContinue, nil
	})
	return buf.String()
}
