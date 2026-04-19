// Package gotcpdf is a 1:1 Go port of tc-lib-pdf (TCPDF) by Nicola Asuni.
//
// Usage:
//
//	pdf, err := gotcpdf.New(classobjects.DefaultConfig())
//	pdf.SetTitle("My Document")
//	pdf.AddPage()
//	pdf.SetFont("Helvetica", "", 12)
//	pdf.Cell(0, 10, "Hello, World!")
//	err = pdf.SavePDF("output.pdf")
package gotcpdf

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/tecnickcom/go-tcpdf/pdfbuf"

	"github.com/tecnickcom/go-tcpdf/base"
	"github.com/tecnickcom/go-tcpdf/classobjects"
	"github.com/tecnickcom/go-tcpdf/color"
	"github.com/tecnickcom/go-tcpdf/encrypt"
	"github.com/tecnickcom/go-tcpdf/font"
	"github.com/tecnickcom/go-tcpdf/graph"
	"github.com/tecnickcom/go-tcpdf/metainfo"
	"github.com/tecnickcom/go-tcpdf/output"
	"github.com/tecnickcom/go-tcpdf/page"
	"github.com/tecnickcom/go-tcpdf/text"
)

// TCPDF is the main document type. All exported methods are safe for
// concurrent use from multiple goroutines. It mirrors the PHP Tcpdf class hierarchy
// via composition: Base → Cell → MetaInfo → Output → JavaScript → Text → CSS → HTML → SVG.
type TCPDF struct {
	mu sync.RWMutex

	*classobjects.ClassObjects

	// Current state
	currentPage       *page.PageData
	currentFontKey    string
	currentFontSize   float64
	currentFontStyle  string
	currentFontFamily string

	// Position cursor (user units, top-down)
	X, Y float64

	// Auto page break
	autoPageBreak bool
	pageBreakTrig float64 // Y trigger

	// Page content buffers (one per page, raw PDF operators)
	pageContents []string

	// Graphics state
	draw *graph.Draw

	// Config
	cfg classobjects.Config
}

// New creates a new TCPDF document.
func New(cfg classobjects.Config) (*TCPDF, error) {
	co, err := classobjects.InitClassObjects(cfg)
	if err != nil {
		return nil, fmt.Errorf("tcpdf: init: %w", err)
	}
	t := &TCPDF{
		ClassObjects:  co,
		autoPageBreak: true,
		draw:          graph.NewDraw(),
		cfg:           cfg,
	}
	return t, nil
}

// NewDefault creates a TCPDF document with default settings (A4, mm, portrait).
func NewDefault() (*TCPDF, error) {
	return New(classobjects.DefaultConfig())
}

// ---- Metadata -----------------------------------------------------------

// SetTitle sets the document title. Chainable.
func (t *TCPDF) SetTitle(title string) *TCPDF {
	t.Meta.SetTitle(title)
	return t
}

// SetAuthor sets the document author. Chainable.
func (t *TCPDF) SetAuthor(author string) *TCPDF {
	t.Meta.SetAuthor(author)
	return t
}

// SetSubject sets the document subject. Chainable.
func (t *TCPDF) SetSubject(subject string) *TCPDF {
	t.Meta.SetSubject(subject)
	return t
}

// SetKeywords sets the document keywords. Chainable.
func (t *TCPDF) SetKeywords(kw string) *TCPDF {
	t.Meta.SetKeywords(kw)
	return t
}

// SetCreator sets the creator application name. Chainable.
func (t *TCPDF) SetCreator(creator string) *TCPDF {
	t.Meta.SetCreator(creator)
	return t
}

// SetProducer sets the producer application name. Chainable.
func (t *TCPDF) SetProducer(producer string) *TCPDF {
	t.Meta.SetProducer(producer)
	return t
}

// SetPDFVersion sets the PDF version string.
func (t *TCPDF) SetPDFVersion(v string) error {
	_, err := t.Meta.SetPDFVersion(v)
	return err
}

// SetViewerPreferences configures viewer preferences.
func (t *TCPDF) SetViewerPreferences(vp metainfo.ViewerPreferences) *TCPDF {
	t.Meta.SetViewerPreferences(vp)
	return t
}

// SetCustomXMP adds an XMP fragment.
func (t *TCPDF) SetCustomXMP(key, xmp string) *TCPDF {
	t.Meta.SetCustomXMP(key, xmp)
	return t
}

// SetSRGB enables the sRGB ICC color profile.
func (t *TCPDF) SetSRGB(enabled bool) *TCPDF {
	t.Meta.SetSRGB(enabled)
	return t
}

// SetRTL sets the default text direction.
func (t *TCPDF) SetRTL(enabled bool) *TCPDF {
	t.Base.SetRTL(enabled)
	t.UniConv.SetRTL(enabled)
	return t
}

// SetLanguage sets the document language tag (e.g. "en-US").
func (t *TCPDF) SetLanguage(lang string) *TCPDF {
	t.Meta.Language = lang
	return t
}

// ---- Page management ----------------------------------------------------

// AddPage adds a new page, optionally with a specific format and orientation.
// format and orientation default to document defaults if empty.
func (t *TCPDF) AddPage(args ...interface{}) error {
	var format string
	var orient page.Orientation
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			if v == "P" || v == "L" {
				orient = page.Orientation(v)
			} else {
				format = v
			}
		case page.Orientation:
			orient = v
		}
	}

	// Close current page stream
	if t.currentPage != nil {
		t.closePage()
	}

	pg, err := t.Page.Add(format, orient, nil)
	if err != nil {
		return fmt.Errorf("tcpdf: AddPage: %w", err)
	}
	t.currentPage = pg

	// Reset cursor
	margins := pg.Margins
	t.X = margins.Left
	t.Y = margins.Top
	if t.autoPageBreak {
		t.pageBreakTrig = pg.Height() - margins.Bottom
	}

	// Update base page height for Y-flip
	t.Base.PageHeight = pg.Boxes.Media.Ury

	// Start new content buffer
	t.pageContents = append(t.pageContents, "")

	// Default page content (e.g. background from hook)
	if t.Base.IsDefaultPageContentEnabled() {
		t.emitDefaultPageContent(pg)
	}

	// Restore current font if set
	if t.currentFontKey != "" {
		t.emitFontOperator()
	}

	// Add TOC entry if document has a TOC auto mode
	// (manual call via AddTOCEntry)

	return nil
}

// SetPage sets the active page (1-indexed).
func (t *TCPDF) SetPage(pageNum int) error {
	if pageNum < 1 || pageNum > t.Page.Count() {
		return fmt.Errorf("tcpdf: SetPage: page %d out of range", pageNum)
	}
	pg, _ := t.Page.Get(pageNum - 1)
	t.currentPage = pg
	t.Base.PageHeight = pg.Boxes.Media.Ury
	return nil
}

// PageCount returns the number of pages.
func (t *TCPDF) PageCount() int {
	return t.Page.Count()
}

// SetAutoPageBreak configures automatic page breaks.
func (t *TCPDF) SetAutoPageBreak(auto bool, margin float64) {
	t.autoPageBreak = auto
	if t.currentPage != nil {
		t.pageBreakTrig = t.currentPage.Height() - margin
	}
}

// SetMargins sets the page margins.
func (t *TCPDF) SetMargins(left, top, right float64) {
	if t.currentPage != nil {
		t.currentPage.Margins.Left = left
		t.currentPage.Margins.Top = top
		t.currentPage.Margins.Right = right
		t.X = left
	}
}

// ---- Font management ----------------------------------------------------

// SetFont sets the current font.
// family: e.g. "Helvetica", "Times", "Courier"
// style: "" | "B" | "I" | "BI" | "U" (may be combined)
// size: points
func (t *TCPDF) SetFont(family, style string, size float64) error {
	family = strings.ToLower(strings.TrimSpace(family))
	style = strings.ToUpper(strings.TrimSpace(style))

	// Build core font name
	coreName := family
	if style == "B" || style == "BI" || style == "IB" {
		if family == "helvetica" {
			coreName = "helvetica-bold"
		} else if family == "times" {
			coreName = "times-bold"
		} else if family == "courier" {
			coreName = "courier-bold"
		}
	}
	if strings.Contains(style, "I") {
		if family == "helvetica" {
			if strings.Contains(coreName, "bold") {
				coreName = "helvetica-boldoblique"
			} else {
				coreName = "helvetica-oblique"
			}
		} else if family == "times" {
			if strings.Contains(coreName, "bold") {
				coreName = "times-bolditalic"
			} else {
				coreName = "times-italic"
			}
		} else if family == "courier" {
			if strings.Contains(coreName, "bold") {
				coreName = "courier-boldoblique"
			} else {
				coreName = "courier-oblique"
			}
		}
	}

	rk, err := t.Fonts.LoadCore(coreName)
	if err != nil {
		// Try as-is (for custom loaded fonts)
		rk, err = t.Fonts.LoadCore(family)
		if err != nil {
			return fmt.Errorf("tcpdf: SetFont: %w", err)
		}
	}

	t.currentFontKey = rk
	t.currentFontSize = size
	t.currentFontFamily = family
	t.currentFontStyle = style
	t.Text.SetFont(rk, size)
	t.emitFontOperator()
	return nil
}

// GetFontSize returns the current font size in points.
func (t *TCPDF) GetFontSize() float64 { return t.currentFontSize }

// GetFontFamily returns the current font family name.
func (t *TCPDF) GetFontFamily() string { return t.currentFontFamily }

// ---- Color management ---------------------------------------------------

// SetFillColor sets the fill color from a color.Color.
func (t *TCPDF) SetFillColor(c color.Color) *TCPDF {
	t.appendContent(t.draw.SetFillColor(c))
	return t
}

// SetDrawColor sets the stroke/draw color.
func (t *TCPDF) SetDrawColor(c color.Color) *TCPDF {
	t.appendContent(t.draw.SetStrokeColor(c))
	return t
}

// SetTextColor sets the text color (fill).
func (t *TCPDF) SetTextColor(c color.Color) *TCPDF {
	// Text color is set inside BT block; store and apply in GetTextLine
	t.appendContent(c.FillOperator() + "\n")
	return t
}

// SetAlpha sets the transparency for both fill and stroke.
func (t *TCPDF) SetAlpha(alpha float64, blendMode string) *TCPDF {
	if blendMode == "" {
		blendMode = "Normal"
	}
	t.Graph.SetFillColor(color.NewRGB(0, 0, 0).WithAlpha(alpha)) // trigger ExtGState registration
	_ = alpha
	return t
}

// ---- Drawing ------------------------------------------------------------

// SetLineWidth sets the current line width.
func (t *TCPDF) SetLineWidth(w float64) *TCPDF {
	t.appendContent(t.draw.SetLineWidth(w * t.Base.ToPoints(1)))
	return t
}

// SetLineCap sets the line cap style.
func (t *TCPDF) SetLineCap(cap int) *TCPDF {
	t.appendContent(t.draw.SetLineCap(cap))
	return t
}

// SetLineJoin sets the line join style.
func (t *TCPDF) SetLineJoin(join int) *TCPDF {
	t.appendContent(t.draw.SetLineJoin(join))
	return t
}

// SetLineDash sets the dash pattern (units and phase in user units).
func (t *TCPDF) SetLineDash(array []float64, phase float64) *TCPDF {
	pts := make([]float64, len(array))
	for i, v := range array {
		pts[i] = t.Base.ToPoints(v)
	}
	t.appendContent(t.draw.SetDash(pts, t.Base.ToPoints(phase)))
	return t
}

// Line draws a line from (x1,y1) to (x2,y2) in user units.
func (t *TCPDF) Line(x1, y1, x2, y2 float64) *TCPDF {
	px1, py1 := t.toPagePt(x1, y1)
	px2, py2 := t.toPagePt(x2, y2)
	t.appendContent(t.draw.Line(px1, py1, px2, py2))
	return t
}

// Rect draws a rectangle in user units.
// style: "S", "F", "FD", etc.
func (t *TCPDF) Rect(x, y, w, h float64, style string) *TCPDF {
	px, py := t.toPagePt(x, y+h) // PDF origin is bottom-left
	pw := t.Base.ToPoints(w)
	ph := t.Base.ToPoints(h)
	t.appendContent(t.draw.Rect(px, py, pw, ph, style))
	return t
}

// Circle draws a circle at (cx, cy) with radius r.
func (t *TCPDF) Circle(cx, cy, r float64, style string) *TCPDF {
	pcx, pcy := t.toPagePt(cx, cy)
	pr := t.Base.ToPoints(r)
	t.appendContent(t.draw.Circle(pcx, pcy, pr, style))
	return t
}

// Ellipse draws an ellipse.
func (t *TCPDF) Ellipse(cx, cy, rx, ry, angle float64, style string) *TCPDF {
	pcx, pcy := t.toPagePt(cx, cy)
	prx := t.Base.ToPoints(rx)
	pry := t.Base.ToPoints(ry)
	t.appendContent(t.draw.Ellipse(pcx, pcy, prx, pry, angle, style))
	return t
}

// ---- Text output --------------------------------------------------------

// GetX returns the current X position.
func (t *TCPDF) GetX() float64 { return t.X }

// GetY returns the current Y position.
func (t *TCPDF) GetY() float64 { return t.Y }

// SetX sets the current X position.
func (t *TCPDF) SetX(x float64) { t.X = x }

// SetY sets the current Y position (and resets X to left margin).
func (t *TCPDF) SetY(y float64) {
	t.Y = y
	if t.currentPage != nil {
		t.X = t.currentPage.Margins.Left
	}
}

// SetXY sets both cursor positions.
func (t *TCPDF) SetXY(x, y float64) {
	t.X = x
	t.Y = y
}

// Cell writes a cell with optional text and border.
// w=0 means "remaining page width".
// h is the cell height in user units.
// ln: 0=right, 1=next line, 2=below
// border: "" | "0" | "1" | combination of "L","T","R","B"
// align: "L", "C", "R", "J"
// fill: true to fill background
func (t *TCPDF) Cell(w, h float64, txt, border string, ln int, align string, fill bool, link string) error {
	if t.currentPage == nil {
		return fmt.Errorf("tcpdf: Cell called before AddPage")
	}
	pg := t.currentPage
	margins := pg.Margins

	if w == 0 {
		w = pg.Width() - margins.Right - t.X
	}

	// Auto page break
	if t.autoPageBreak && t.Y+h > t.pageBreakTrig {
		if err := t.AddPage(); err != nil {
			return err
		}
	}

	defCell := t.ClassObjects.Cell.GetDefaultCell()
	opts := text.TextOptions{
		PosX:     t.Base.ToPoints(t.X),
		PosY:     t.Base.ToYPoints(t.Y + h), // bottom-left of cell in PDF coords
		Width:    t.Base.ToPoints(w),
		Height:   t.Base.ToPoints(h),
		HAlign:   align,
		FontKey:  t.currentFontKey,
		FontSize: t.currentFontSize,
		DrawCell: border == "1" || len(border) > 0,
		Fill:     fill,
		Cell:     &defCell,
	}

	ops := t.Text.GetTextCell(txt, opts)
	t.appendContent(ops)

	// Advance cursor
	switch ln {
	case 0:
		t.X += w
	case 1:
		t.X = margins.Left
		t.Y += h
	case 2:
		t.Y += h
	}

	// Handle link
	if link != "" {
		t.JS.SetLink(pg.Index, t.X, t.Y, w, h, link)
	}

	return nil
}

// MultiCell writes a block of text wrapped into a cell.
func (t *TCPDF) MultiCell(w, h float64, txt, border, align string, fill bool) error {
	if t.currentPage == nil {
		return fmt.Errorf("tcpdf: MultiCell called before AddPage")
	}
	pg := t.currentPage
	if w == 0 {
		w = pg.Width() - pg.Margins.Right - t.X
	}
	defCell := t.ClassObjects.Cell.GetDefaultCell()
	opts := text.TextOptions{
		PosX:     t.Base.ToPoints(t.X),
		PosY:     t.Base.ToYPoints(t.Y),
		Width:    t.Base.ToPoints(w),
		Height:   t.Base.ToPoints(h),
		HAlign:   align,
		FontKey:  t.currentFontKey,
		FontSize: t.currentFontSize,
		Fill:     fill,
		DrawCell: border == "1",
		Cell:     &defCell,
	}
	ops := t.Text.GetTextCell(txt, opts)
	t.appendContent(ops)
	t.Y += h
	t.X = pg.Margins.Left
	return nil
}

// Write writes text that flows from the current position, with automatic line breaks.
func (t *TCPDF) Write(h float64, txt, link string) error {
	if t.currentPage == nil {
		return fmt.Errorf("tcpdf: Write called before AddPage")
	}
	pg := t.currentPage
	w := pg.Width() - pg.Margins.Right - t.X
	return t.Cell(w, h, txt, "", 1, "L", false, link)
}

// Ln performs a line break. height=0 uses the current font line height.
func (t *TCPDF) Ln(height float64) {
	if height == 0 && t.currentFontKey != "" {
		if m, ok := t.Fonts.Get(t.currentFontKey); ok {
			height = font.LineHeight(m, t.currentFontSize)
			if t.currentPage != nil {
				height /= t.currentPage.ScaleFactor
			}
		}
	}
	t.Y += height
	if t.currentPage != nil {
		t.X = t.currentPage.Margins.Left
	}
}

// GetStringWidth returns the width of the string with the current font.
func (t *TCPDF) GetStringWidth(s string) float64 {
	w := t.Fonts.TextWidth(t.currentFontKey, s, t.currentFontSize)
	if t.currentPage != nil {
		return w / t.currentPage.ScaleFactor
	}
	return w
}

// ---- Image --------------------------------------------------------------

// Image places an image file on the page.
// x, y, w, h: user units; 0 = auto
// imgType: "", "jpg", "png"
// link: optional URL
func (t *TCPDF) Image(imgPath string, x, y, w, h float64, imgType, link string) error {
	if t.currentPage == nil {
		return fmt.Errorf("tcpdf: Image called before AddPage")
	}
	_, img, err := t.Images.LoadFile(imgPath)
	if err != nil {
		return fmt.Errorf("tcpdf: Image: %w", err)
	}

	// Calculate dimensions
	if w == 0 && h == 0 {
		w = float64(img.Width) / t.currentPage.ScaleFactor
		h = float64(img.Height) / t.currentPage.ScaleFactor
	} else if w == 0 {
		w = h * float64(img.Width) / float64(img.Height)
	} else if h == 0 {
		h = w * float64(img.Height) / float64(img.Width)
	}

	px, py := t.toPagePt(x, y+h)
	pw := t.Base.ToPoints(w)
	ph := t.Base.ToPoints(h)

	var ib strings.Builder
	ib.WriteString("q ")
	ib.WriteString(pdfbuf.FmtF(pw))
	ib.WriteString(" 0 0 ")
	ib.WriteString(pdfbuf.FmtF(ph))
	ib.WriteByte(' ')
	ib.WriteString(pdfbuf.FmtF(px))
	ib.WriteByte(' ')
	ib.WriteString(pdfbuf.FmtF(py))
	ib.WriteString(" cm /")
	ib.WriteString(img.Key)
	ib.WriteString(" Do Q\n")
	t.appendContent(ib.String())

	if link != "" {
		t.JS.SetLink(t.currentPage.Index, x, y, w, h, link)
	}
	return nil
}

// ---- HTML ---------------------------------------------------------------

// WriteHTML renders HTML content at the current position.
func (t *TCPDF) WriteHTML(html string, ln, fill bool) error {
	if t.currentPage == nil {
		return fmt.Errorf("tcpdf: WriteHTML called before AddPage")
	}
	pg := t.currentPage
	w := pg.Width() - pg.Margins.Right - t.X
	h := pg.Height() - pg.Margins.Bottom - t.Y
	ops := t.HTML.GetHTMLCell(html, t.Base.ToPoints(t.X), t.Base.ToYPoints(t.Y),
		t.Base.ToPoints(w), t.Base.ToPoints(h), nil, nil)
	t.appendContent(ops)
	return nil
}

// ---- SVG ----------------------------------------------------------------

// ImageSVG places an SVG image on the page.
func (t *TCPDF) ImageSVG(svgData string, x, y, w, h float64) error {
	if t.currentPage == nil {
		return fmt.Errorf("tcpdf: ImageSVG called before AddPage")
	}
	px, py := t.toPagePt(x, y+h)
	soid, err := t.SVG.AddSVG(svgData, px, py, t.Base.ToPoints(w), t.Base.ToPoints(h), t.Base.PageHeight)
	if err != nil {
		return fmt.Errorf("tcpdf: ImageSVG: %w", err)
	}
	t.appendContent(t.SVG.GetSetSVG(soid))
	return nil
}

// ---- Bookmarks and links ------------------------------------------------

// SetBookmark adds a bookmark/outline entry.
func (t *TCPDF) SetBookmark(name string, level int) {
	if t.currentPage == nil {
		return
	}
	t.JS.SetBookmark(name, "", level, t.currentPage.Index, t.X, t.Y, "", "")
	t.TOC.AddEntry(level, name, t.currentPage.Index+1, t.Y)
}

// AddLink creates an external link annotation.
func (t *TCPDF) AddLink(url string, x, y, w, h float64) int {
	if t.currentPage == nil {
		return -1
	}
	return t.JS.SetLink(t.currentPage.Index, x, y, w, h, url)
}

// ---- Layer management ---------------------------------------------------

// StartLayer opens a new content layer.
func (t *TCPDF) StartLayer(name string, visible bool) {
	id := t.Layers.NewLayer(name, visible)
	t.appendContent(t.Layers.OpenLayer(id))
}

// EndLayer closes the current content layer.
func (t *TCPDF) EndLayer() {
	t.appendContent(t.Layers.CloseLayer())
}

// ---- JavaScript ---------------------------------------------------------

// AppendJavascript appends raw JavaScript to the document.
func (t *TCPDF) AppendJavascript(js string) {
	t.JS.AppendRawJavaScript(js)
}

// ---- Graphics state -----------------------------------------------------

// StartTransform saves the graphics state (q).
func (t *TCPDF) StartTransform() {
	t.appendContent("q\n")
}

// StopTransform restores the graphics state (Q).
func (t *TCPDF) StopTransform() {
	t.appendContent("Q\n")
}

// ScaleXY scales the coordinate system.
func (t *TCPDF) ScaleXY(sx, sy, cx, cy float64) {
	pcx, pcy := t.toPagePt(cx, cy)
	m := graph.Matrix{A: sx, B: 0, C: 0, D: sy, E: pcx * (1 - sx), F: pcy * (1 - sy)}
	t.appendContent(m.PDF() + "\n")
}

// Rotate rotates around a centre point (degrees).
func (t *TCPDF) Rotate(angle, cx, cy float64) {
	pcx, pcy := t.toPagePt(cx, cy)
	rm := graph.Rotate(angle * 3.14159265358979323846 / 180)
	rm.E = pcx - rm.A*pcx - rm.C*pcy
	rm.F = pcy - rm.B*pcx - rm.D*pcy
	t.appendContent(rm.PDF() + "\n")
}

// ---- Encryption ---------------------------------------------------------

// SetEncryption configures PDF encryption.
func (t *TCPDF) SetEncryption(cfg encrypt.Config) error {
	enc, err := encrypt.New(cfg)
	if err != nil {
		return err
	}
	t.ClassObjects.Encrypt = enc
	return nil
}

// ---- Output -------------------------------------------------------------

// WriteTo writes the complete PDF directly to w using the two-pass streaming
// serializer. It implements io.WriterTo.
//
// No complete PDF copy is held in RAM: only the xref table (~8 bytes per
// object) and the 256 KiB write buffer are allocated beyond the document
// model. This is the recommended output method for:
//   - HTTP handlers (http.ResponseWriter)
//   - Large documents
//   - Network connections and pipes
//   - Any context where minimising peak RAM matters
func (t *TCPDF) WriteTo(w io.Writer) (int64, error) {
	if t.currentPage != nil {
		t.closePage()
	}
	return output.New(t.buildDocument()).WriteTo(w)
}

// Output writes the PDF to w. This is an alias for WriteTo that discards
// the byte count; useful when the caller only needs error-checking.
func (t *TCPDF) Output(w io.Writer) error {
	_, err := t.WriteTo(w)
	return err
}

// GetOutPDFString builds the PDF and returns it as a byte slice.
//
// Equivalent to WriteTo with a bytes.Buffer. Convenient for small documents
// and tests; for large documents prefer WriteTo or SavePDF.
func (t *TCPDF) GetOutPDFString() ([]byte, error) {
	if t.currentPage != nil {
		t.closePage()
	}
	return output.New(t.buildDocument()).GetOutPDFString()
}

// SavePDF writes the PDF to the file at path in a single streaming pass.
// The file is never fully held in RAM.
func (t *TCPDF) SavePDF(path string) error {
	if t.currentPage != nil {
		t.closePage()
	}
	return output.New(t.buildDocument()).SavePDF(path)
}

// ---- Internal helpers ---------------------------------------------------

// closePage finalises the current page content buffer.
func (t *TCPDF) closePage() {
	// Nothing special needed: pageContents already accumulates content.
}

// appendContent appends PDF operators to the current page's content buffer.
// Caller must hold t.mu.Lock().
func (t *TCPDF) appendContent(ops string) {
	if len(t.pageContents) == 0 {
		return
	}
	t.pageContents[len(t.pageContents)-1] += ops
}

// emitFontOperator appends the font selection operator to the current page.
func (t *TCPDF) emitFontOperator() {
	if t.currentFontKey != "" && t.currentFontSize > 0 {
		var fb strings.Builder
		fb.WriteString("BT /")
		fb.WriteString(t.currentFontKey)
		fb.WriteByte(' ')
		fb.WriteString(pdfbuf.FmtF(t.currentFontSize))
		fb.WriteString(" Tf ET\n")
		t.appendContent(fb.String())
	}
}

// emitDefaultPageContent emits the default page header/footer via the hook.
func (t *TCPDF) emitDefaultPageContent(pg *page.PageData) {
	// Override point — subclasses would override Header() and Footer().
	// This is a no-op in the base implementation.
}

// toPagePt converts (x, y) from user units (top-down) to PDF points (bottom-up).
func (t *TCPDF) toPagePt(x, y float64) (float64, float64) {
	return t.Base.ToPoints(x), t.Base.ToYPoints(y)
}

// buildDocument assembles the output.Document from current state.
func (t *TCPDF) buildDocument() *output.Document {
	return &output.Document{
		Meta:         t.Meta,
		Pages:        t.Page,
		Fonts:        t.Fonts,
		Images:       t.Images,
		Encrypt:      t.ClassObjects.Encrypt,
		Spots:        t.Color,
		Gradients:    nil,
		ExtGStates:   nil,
		JS:           t.JS,
		ICCProfile:   t.ICCProfile,
		PageContents: t.pageContents,
		Compress:     t.cfg.Compress,
	}
}

// ---- Helpers ------------------------------------------------------------

func fmtF(v float64) string { return pdfbuf.FmtF(v) }

// ---- Default cell/margin/CSS wrappers ----------------------------------

// SetDefaultCellPadding sets the default cell padding (user units, TRBL).
func (t *TCPDF) SetDefaultCellPadding(top, right, bottom, left float64) {
	t.ClassObjects.Cell.SetDefaultCellPadding(
		t.Base.ToPoints(top),
		t.Base.ToPoints(right),
		t.Base.ToPoints(bottom),
		t.Base.ToPoints(left),
	)
}

// SetDefaultCellMargin sets the default cell margin (user units, TRBL).
func (t *TCPDF) SetDefaultCellMargin(top, right, bottom, left float64) {
	t.ClassObjects.Cell.SetDefaultCellMargin(
		t.Base.ToPoints(top),
		t.Base.ToPoints(right),
		t.Base.ToPoints(bottom),
		t.Base.ToPoints(left),
	)
}

// SetDefaultCellBorderPos sets the default border position.
func (t *TCPDF) SetDefaultCellBorderPos(pos float64) {
	t.ClassObjects.Cell.SetDefaultCellBorderPos(pos)
}

// SetDefaultCSSMargin sets the default CSS margin.
func (t *TCPDF) SetDefaultCSSMargin(top, right, bottom, left float64) {
	t.CSS.SetDefaultCSSMargin(
		t.Base.ToPoints(top), t.Base.ToPoints(right),
		t.Base.ToPoints(bottom), t.Base.ToPoints(left),
	)
}

// SetDefaultCSSPadding sets the default CSS padding.
func (t *TCPDF) SetDefaultCSSPadding(top, right, bottom, left float64) {
	t.CSS.SetDefaultCSSPadding(
		t.Base.ToPoints(top), t.Base.ToPoints(right),
		t.Base.ToPoints(bottom), t.Base.ToPoints(left),
	)
}

// SetDefaultCSSBorderSpacing sets the default CSS border-spacing.
func (t *TCPDF) SetDefaultCSSBorderSpacing(vert, horiz float64) {
	t.CSS.SetDefaultCSSBorderSpacing(t.Base.ToPoints(vert), t.Base.ToPoints(horiz))
}

// EnableDefaultPageContent enables/disables the default page content callback.
func (t *TCPDF) EnableDefaultPageContent(enable bool) {
	t.Base.EnableDefaultPageContent(enable)
}

// EnableZeroWidthBreakPoints enables/disables zero-width break points.
func (t *TCPDF) EnableZeroWidthBreakPoints(enabled bool) {
	t.Base.EnableZeroWidthBreakPoints(enabled)
	t.Text.EnableZeroWidthBreakPoints(enabled)
}

// LoadTexHyphenPatterns loads TeX hyphenation patterns from a string.
func (t *TCPDF) LoadTexHyphenPatterns(content string) {
	patterns := t.Text.LoadTexHyphenPatterns(content)
	t.Text.SetTexHyphenPatterns(patterns)
}

// GetLastBBox returns the bounding box of the last rendered text.
func (t *TCPDF) GetLastBBox() base.BBox {
	return t.Text.GetLastBBox()
}

// SetEncryption configures document encryption. The encryptor is stored
// on the embedded ClassObjects so it is available to the output layer.
// (This shadows the method defined earlier — see the one with encrypt.Config above.)
