// Package text provides PDF text rendering: cells, lines, hyphenation.
// Ported from tc-lib-pdf Text.php by Nicola Asuni.
package text

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/tecnickcom/go-tcpdf/base"
	internalFont "github.com/tecnickcom/go-tcpdf/font"
	"github.com/tecnickcom/go-tcpdf/pdfbuf"
	internalUni "github.com/tecnickcom/go-tcpdf/unicode"
)

// TextShadow holds shadow parameters for text.
type TextShadow struct {
	OffsetX float64
	OffsetY float64
	Color   string // CSS color string
}

// TextOptions bundles all options for rendering a text block.
type TextOptions struct {
	PosX        float64
	PosY        float64
	Width       float64
	Height      float64
	Offset      float64 // horizontal offset for first line
	LineSpace   float64 // extra inter-line spacing
	VAlign      string  // "T", "C", "B"
	HAlign      string  // "L", "C", "R", "J"
	Cell        *base.CellDef
	StrokeWidth float64
	WordSpacing float64
	Leading     float64
	Rise        float64
	JLast       bool // don't justify last line when HAlign=="J"
	Fill        bool
	Stroke      bool
	Underline   bool
	LineThrough bool
	Overline    bool
	Clip        bool
	DrawCell    bool
	ForceDir    string // "L" or "R"
	Shadow      *TextShadow
	FontKey     string
	FontSize    float64
}

// Text manages text rendering operations.
type Text struct {
	fonts       *internalFont.Stack
	uniconv     *internalUni.Convert
	hyphenator  *internalUni.Hyphenator
	breakPoints bool
	lastBBox    base.BBox

	// Current font
	currentFontKey  string
	currentFontSize float64
}

// New creates a Text renderer.
func New(fonts *internalFont.Stack, uniconv *internalUni.Convert) *Text {
	return &Text{
		fonts:   fonts,
		uniconv: uniconv,
	}
}

// SetFont sets the current active font.
func (t *Text) SetFont(key string, size float64) {
	t.currentFontKey = key
	t.currentFontSize = size
}

// LoadTexHyphenPatterns parses a TeX hyphenation pattern file content into a map.
func (t *Text) LoadTexHyphenPatterns(content string) map[string]string {
	patterns := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") {
			continue
		}
		// Extract digit-interleaved pattern
		var key strings.Builder
		var val strings.Builder
		lastWasDigit := false
		for _, r := range line {
			if r >= '0' && r <= '9' {
				val.WriteRune(r)
				lastWasDigit = true
			} else {
				if !lastWasDigit {
					val.WriteRune('0')
				}
				key.WriteRune(r)
				lastWasDigit = false
			}
		}
		if !lastWasDigit {
			val.WriteRune('0')
		}
		patterns[key.String()] = val.String()
	}
	return patterns
}

// SetTexHyphenPatterns configures the hyphenation engine.
func (t *Text) SetTexHyphenPatterns(patterns map[string]string) {
	t.hyphenator = internalUni.NewHyphenator(patterns)
}

// EnableZeroWidthBreakPoints enables or disables zero-width break points.
func (t *Text) EnableZeroWidthBreakPoints(enabled bool) {
	t.breakPoints = enabled
}

// GetLastBBox returns the last rendered text bounding box.
func (t *Text) GetLastBBox() base.BBox { return t.lastBBox }

// ---- Line rendering -----------------------------------------------------

// GetTextLine returns the PDF stream operator string for a single line of text.
func (t *Text) GetTextLine(txt string, opts TextOptions) string {
	if txt == "" {
		return ""
	}
	fk := opts.FontKey
	if fk == "" {
		fk = t.currentFontKey
	}
	sz := opts.FontSize
	if sz == 0 {
		sz = t.currentFontSize
	}

	m, ok := t.fonts.Get(fk)
	if !ok {
		return ""
	}
	t.fonts.MarkUsed(fk, txt)

	var sb strings.Builder

	// Save/restore graphics state if we have stroke or clipping
	needQ := opts.Stroke || opts.Clip || opts.Shadow != nil
	if needQ {
		sb.WriteString("q\n")
	}

	// Text state operators
	sb.WriteString("BT\n")
	sb.WriteByte('/'); sb.WriteString(fk); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(sz)); sb.WriteString(" Tf\n")

	// Text rendering mode
	renderMode := 0
	if opts.Fill && opts.Stroke {
		renderMode = 2
	} else if opts.Stroke {
		renderMode = 1
	} else if opts.Clip {
		renderMode = 7
	}
	if renderMode != 0 {
		sb.WriteString(pdfbuf.FmtI(renderMode)); sb.WriteString(" Tr\n")
	}

	// Word spacing
	if opts.WordSpacing != 0 {
		sb.WriteString(pdfbuf.FmtF(opts.WordSpacing)); sb.WriteString(" Tw\n")
	}

	// Char spacing / leading / rise
	if opts.Leading != 0 {
		sb.WriteString(pdfbuf.FmtF(opts.Leading)); sb.WriteString(" TL\n")
	}
	if opts.Rise != 0 {
		sb.WriteString(pdfbuf.FmtF(opts.Rise)); sb.WriteString(" Ts\n")
	}

	// Position
	sb.WriteString(pdfbuf.FmtF(opts.PosX)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(opts.PosY)); sb.WriteString(" Td\n")

	// Stroke width for outlined text
	if opts.Stroke && opts.StrokeWidth > 0 {
		sb.WriteString(pdfbuf.FmtF(opts.StrokeWidth)); sb.WriteString(" w\n")
	}

	// Justify: calculate word spacing
	if opts.HAlign == "J" && opts.Width > 0 {
		tw := m.StringWidth(txt, sz)
		spaces := strings.Count(txt, " ")
		if spaces > 0 && tw < opts.Width {
			ws := (opts.Width - tw) / float64(spaces)
			sb.WriteString(pdfbuf.FmtF(ws)); sb.WriteString(" Tw\n")
		}
	}

	// Text string
	sb.WriteByte('('); sb.WriteString(internalUni.EscapePDFString(txt)); sb.WriteString(") Tj\n")
	sb.WriteString("ET\n")

	// Decorations
	if opts.Underline || opts.LineThrough || opts.Overline {
		tw := m.StringWidth(txt, sz)
		sb.WriteString(t.textDecorations(opts, m, sz, tw))
	}

	if needQ {
		sb.WriteString("Q\n")
	}

	// Update last bounding box
	tw := m.StringWidth(txt, sz)
	t.lastBBox = base.BBox{
		Llx: opts.PosX, Lly: opts.PosY - math.Abs(m.Descent)*sz/1000,
		Urx: opts.PosX + tw, Ury: opts.PosY + m.Ascent*sz/1000,
	}

	return sb.String()
}

// textDecorations emits underline/linethrough/overline operators.
func (t *Text) textDecorations(opts TextOptions, m *internalFont.FontMetric, sz, tw float64) string {
	var sb strings.Builder
	y := opts.PosY
	if opts.Underline {
		uy := y + m.UnderlinePos*sz/1000
		uw := m.UnderlineThick * sz / 1000
		sb.WriteString(pdfbuf.FmtF(uw)); sb.WriteString(" w ")
		sb.WriteString(pdfbuf.FmtF(opts.PosX)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(uy)); sb.WriteString(" m ")
		sb.WriteString(pdfbuf.FmtF(opts.PosX+tw)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(uy)); sb.WriteString(" l S\n")
	}
	if opts.LineThrough {
		ly := y + (m.Ascent+m.Descent)/2*sz/1000
		lw := m.UnderlineThick * sz / 1000
		sb.WriteString(pdfbuf.FmtF(lw)); sb.WriteString(" w ")
		sb.WriteString(pdfbuf.FmtF(opts.PosX)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(ly)); sb.WriteString(" m ")
		sb.WriteString(pdfbuf.FmtF(opts.PosX+tw)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(ly)); sb.WriteString(" l S\n")
	}
	if opts.Overline {
		oly := y + m.Ascent*sz/1000
		ow := m.UnderlineThick * sz / 1000
		sb.WriteString(pdfbuf.FmtF(ow)); sb.WriteString(" w ")
		sb.WriteString(pdfbuf.FmtF(opts.PosX)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(oly)); sb.WriteString(" m ")
		sb.WriteString(pdfbuf.FmtF(opts.PosX+tw)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(oly)); sb.WriteString(" l S\n")
	}
	return sb.String()
}

// ---- Cell rendering -----------------------------------------------------

// GetTextCell returns the PDF operators to render a text block in a cell.
// This implements the core cell layout logic.
func (t *Text) GetTextCell(txt string, opts TextOptions) string {
	if opts.FontKey == "" {
		opts.FontKey = t.currentFontKey
	}
	if opts.FontSize == 0 {
		opts.FontSize = t.currentFontSize
	}
	m, ok := t.fonts.Get(opts.FontKey)
	if !ok {
		return ""
	}
	cell := base.ZeroCell
	if opts.Cell != nil {
		cell = *opts.Cell
	}

	// Effective content area
	contentX := opts.PosX + cell.Margin.L + cell.Padding.L
	contentY := opts.PosY + cell.Margin.T + cell.Padding.T
	contentW := opts.Width - cell.Margin.L - cell.Margin.R - cell.Padding.L - cell.Padding.R
	contentH := opts.Height - cell.Margin.T - cell.Margin.B - cell.Padding.T - cell.Padding.B

	// Wrap text into lines
	lines := t.wrapText(txt, opts.FontKey, opts.FontSize, contentW, opts.Offset)

	// Calculate total text height
	lineH := internalFont.LineHeight(m, opts.FontSize) + opts.LineSpace
	totalH := float64(len(lines)) * lineH

	// Vertical alignment
	startY := contentY
	switch opts.VAlign {
	case "C":
		startY = contentY + (contentH-totalH)/2
	case "B":
		startY = contentY + contentH - totalH
	}

	var sb strings.Builder

	// Draw cell border if requested
	if opts.DrawCell && opts.Width > 0 && opts.Height > 0 {
		sb.WriteString(t.drawCellBorder(opts))
	}

	// Render each line
	curY := startY
	for i, line := range lines {
		if curY > opts.PosY+opts.Height {
			break
		}
		isLast := i == len(lines)-1
		lineOpts := opts
		lineOpts.PosY = curY + m.Ascent*opts.FontSize/1000 // baseline
		lineOpts.Width = contentW

		// Horizontal alignment
		lineW := m.StringWidth(line, opts.FontSize)
		switch opts.HAlign {
		case "R":
			lineOpts.PosX = contentX + contentW - lineW
		case "C":
			lineOpts.PosX = contentX + (contentW-lineW)/2
		case "J":
			lineOpts.PosX = contentX
			if !isLast || !opts.JLast {
				lineOpts.Width = contentW // signal to justify
			}
		default:
			lineOpts.PosX = contentX + opts.Offset
			opts.Offset = 0 // only first line
		}

		sb.WriteString(t.GetTextLine(line, lineOpts))
		curY += lineH
	}

	return sb.String()
}

// drawCellBorder emits a rectangle for the cell border.
func (t *Text) drawCellBorder(opts TextOptions) string {
	cell := base.ZeroCell
	if opts.Cell != nil {
		cell = *opts.Cell
	}
	x := opts.PosX + cell.Margin.L
	y := opts.PosY + cell.Margin.T
	w := opts.Width - cell.Margin.L - cell.Margin.R
	h := opts.Height - cell.Margin.T - cell.Margin.B
	var cb strings.Builder
	cb.WriteString(pdfbuf.FmtF(x)); cb.WriteByte(' ')
	cb.WriteString(pdfbuf.FmtF(y)); cb.WriteByte(' ')
	cb.WriteString(pdfbuf.FmtF(w)); cb.WriteByte(' ')
	cb.WriteString(pdfbuf.FmtF(h)); cb.WriteString(" re S\n")
	return cb.String()
}

// ---- Text wrapping ------------------------------------------------------

// wrapText wraps a text string into lines that fit within maxWidth.
// offset is applied only to the first line.
func (t *Text) wrapText(txt, fontKey string, size, maxWidth, offset float64) []string {
	if maxWidth <= 0 {
		return []string{txt}
	}
	m, ok := t.fonts.Get(fontKey)
	if !ok {
		return []string{txt}
	}

	// Split into paragraphs at explicit newlines
	var lines []string
	for _, para := range strings.Split(txt, "\n") {
		wrapped := t.wrapParagraph(para, m, size, maxWidth, offset)
		lines = append(lines, wrapped...)
		offset = 0 // only first paragraph
	}
	return lines
}

// wrapParagraph wraps a single paragraph.
func (t *Text) wrapParagraph(para string, m *internalFont.FontMetric, size, maxWidth, offset float64) []string {
	if para == "" {
		return []string{""}
	}
	words := strings.Fields(para)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var cur strings.Builder
	curW := offset
	spaceW := m.GlyphWidth(' ') * size / 1000

	for _, word := range words {
		ww := m.StringWidth(word, size)
		if cur.Len() > 0 {
			if curW+spaceW+ww > maxWidth {
				lines = append(lines, cur.String())
				cur.Reset()
				curW = 0
				cur.WriteString(word)
				curW = ww
			} else {
				cur.WriteByte(' ')
				cur.WriteString(word)
				curW += spaceW + ww
			}
		} else {
			if ww > maxWidth {
				// Force-break long word using hyphenation if available
				subLines := t.breakWord(word, m, size, maxWidth)
				for i, sl := range subLines {
					if i < len(subLines)-1 {
						lines = append(lines, sl)
					} else {
						cur.WriteString(sl)
						curW = m.StringWidth(sl, size)
					}
				}
			} else {
				cur.WriteString(word)
				curW = ww
			}
		}
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// breakWord attempts to hyphenate or force-break a word that exceeds maxWidth.
func (t *Text) breakWord(word string, m *internalFont.FontMetric, size, maxWidth float64) []string {
	// Try hyphenation first
	if t.hyphenator != nil {
		points := t.hyphenator.Hyphenate(word)
		runes := []rune(word)
		if len(points) > 0 {
			var lines []string
			start := 0
			for _, pt := range points {
				chunk := string(runes[start:pt]) + "-"
				if m.StringWidth(chunk, size) <= maxWidth {
					lines = append(lines, chunk)
					start = pt
				}
			}
			lines = append(lines, string(runes[start:]))
			return lines
		}
	}
	// Hard break at character boundary
	var lines []string
	var cur strings.Builder
	curW := 0.0
	for _, r := range word {
		rw := m.GlyphWidth(r) * size / 1000
		if curW+rw > maxWidth && cur.Len() > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
		}
		cur.WriteRune(r)
		curW += rw
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// StringWidth returns the advance width of s using the current font.
func (t *Text) StringWidth(s string) float64 {
	return t.fonts.TextWidth(t.currentFontKey, s, t.currentFontSize)
}

// CharCount returns the rune count in a string.
func CharCount(s string) int {
	return utf8.RuneCountInString(s)
}

// ---- helpers ------------------------------------------------------------

func fmtF(v float64) string { return pdfbuf.FmtF(v) }
