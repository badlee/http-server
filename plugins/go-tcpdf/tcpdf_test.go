package gotcpdf

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/tecnickcom/go-tcpdf/classobjects"
	"github.com/tecnickcom/go-tcpdf/color"
	"github.com/tecnickcom/go-tcpdf/encrypt"
	"github.com/tecnickcom/go-tcpdf/page"
	"github.com/tecnickcom/go-tcpdf/metainfo"
)

func newPDF(t *testing.T) *TCPDF {
	t.Helper()
	pdf, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault: %v", err)
	}
	return pdf
}

// ---- Construction -------------------------------------------------------

func TestNewDefault(t *testing.T) {
	pdf, err := NewDefault()
	if err != nil {
		t.Fatal(err)
	}
	if pdf == nil {
		t.Fatal("NewDefault returned nil")
	}
}

func TestNewCustomConfig(t *testing.T) {
	pdf, err := New(classobjects.Config{
		Unit:        "in",
		Format:      "LETTER",
		Orientation: page.Landscape,
		Margins:     page.Margins{Top: 0.5, Right: 0.5, Bottom: 0.5, Left: 0.5},
		SubsetFonts: true,
		Compress:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pdf == nil {
		t.Fatal("New returned nil")
	}
}

func TestNewUnknownUnit(t *testing.T) {
	if _, err := New(classobjects.Config{Unit: "zz", Format: "A4"}); err == nil {
		t.Fatal("expected error for unknown unit")
	}
}

// ---- Metadata -----------------------------------------------------------

func TestMetadataChaining(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetTitle("T").SetAuthor("A").SetSubject("S").SetKeywords("k").SetCreator("C")
	if pdf.Meta.Title != "T" || pdf.Meta.Author != "A" {
		t.Fatal("metadata chaining failed")
	}
}

func TestSetPDFVersion(t *testing.T) {
	pdf := newPDF(t)
	if err := pdf.SetPDFVersion("1.4"); err != nil {
		t.Fatal(err)
	}
	if pdf.Meta.PDFVersion != "1.4" {
		t.Fatalf("version: %q", pdf.Meta.PDFVersion)
	}
}

func TestSetPDFVersionInvalid(t *testing.T) {
	pdf := newPDF(t)
	if err := pdf.SetPDFVersion("9.9"); err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestSetLanguage(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetLanguage("fr-FR")
	if pdf.Meta.Language != "fr-FR" {
		t.Fatalf("language: %q", pdf.Meta.Language)
	}
}

func TestSetViewerPreferences(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetViewerPreferences(metainfo.ViewerPreferences{HideToolbar: true})
	if pdf.Meta.ViewerPrefs == nil || !pdf.Meta.ViewerPrefs.HideToolbar {
		t.Fatal("viewer prefs not set")
	}
}

// ---- Pages --------------------------------------------------------------

func TestAddPage(t *testing.T) {
	pdf := newPDF(t)
	if err := pdf.AddPage(); err != nil {
		t.Fatal(err)
	}
	if pdf.PageCount() != 1 {
		t.Fatalf("expected 1 page, got %d", pdf.PageCount())
	}
}

func TestAddMultiplePages(t *testing.T) {
	pdf := newPDF(t)
	for i := 0; i < 5; i++ {
		if err := pdf.AddPage(); err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
	}
	if pdf.PageCount() != 5 {
		t.Fatalf("expected 5 pages, got %d", pdf.PageCount())
	}
}

func TestAddPageLandscape(t *testing.T) {
	pdf := newPDF(t)
	if err := pdf.AddPage(page.Landscape); err != nil {
		t.Fatal(err)
	}
}

func TestAddPageCustomFormat(t *testing.T) {
	pdf := newPDF(t)
	if err := pdf.AddPage("LETTER"); err != nil {
		t.Fatal(err)
	}
}

func TestSetPage(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.AddPage()
	pdf.AddPage()
	if err := pdf.SetPage(2); err != nil {
		t.Fatal(err)
	}
}

func TestSetPageOutOfRange(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	if err := pdf.SetPage(5); err == nil {
		t.Fatal("expected error for page out of range")
	}
}

// ---- Fonts --------------------------------------------------------------

func TestSetFont(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	if err := pdf.SetFont("Helvetica", "", 12); err != nil {
		t.Fatal(err)
	}
	if pdf.GetFontFamily() != "helvetica" {
		t.Fatalf("font family: %q", pdf.GetFontFamily())
	}
	if pdf.GetFontSize() != 12 {
		t.Fatalf("font size: %v", pdf.GetFontSize())
	}
}

func TestSetFontBold(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	if err := pdf.SetFont("Helvetica", "B", 14); err != nil {
		t.Fatal(err)
	}
}

func TestSetFontItalic(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	if err := pdf.SetFont("Times", "I", 11); err != nil {
		t.Fatal(err)
	}
}

func TestSetFontBoldItalic(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	if err := pdf.SetFont("Courier", "BI", 10); err != nil {
		t.Fatal(err)
	}
}

func TestSetFontUnknown(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	if err := pdf.SetFont("NotAFont", "", 12); err == nil {
		t.Fatal("expected error for unknown font")
	}
}

func TestGetStringWidth(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	w := pdf.GetStringWidth("Hello")
	if w <= 0 {
		t.Fatalf("string width should be positive: %v", w)
	}
}

// ---- Cursor -------------------------------------------------------------

func TestSetGetXY(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetX(50)
	if pdf.GetX() != 50 {
		t.Fatalf("X: %v", pdf.GetX())
	}
	pdf.SetY(80)
	if pdf.GetY() != 80 {
		t.Fatalf("Y: %v", pdf.GetY())
	}
	pdf.SetXY(30, 40)
	if pdf.GetX() != 30 || pdf.GetY() != 40 {
		t.Fatalf("SetXY: %v %v", pdf.GetX(), pdf.GetY())
	}
}

func TestLn(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	y0 := pdf.GetY()
	pdf.Ln(10)
	if pdf.GetY()-y0 < 9.9 {
		t.Fatalf("Ln(10) should advance Y by 10: %v → %v", y0, pdf.GetY())
	}
}

// ---- Colors -------------------------------------------------------------

func TestSetFillColor(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFillColor(color.NewRGB(1, 0, 0)) // should not panic
}

func TestSetDrawColor(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetDrawColor(color.Black)
}

func TestSetTextColor(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetTextColor(color.NewRGB(0, 0, 1))
}

// ---- Drawing ------------------------------------------------------------

func TestLine(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.Line(10, 10, 100, 100)
}

func TestRect(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.Rect(20, 30, 80, 50, "FD")
}

func TestCircle(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.Circle(105, 148, 30, "F")
}

func TestEllipse(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.Ellipse(105, 148, 50, 30, 0, "S")
}

func TestSetLineWidth(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetLineWidth(2.0)
}

func TestSetLineDash(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetLineDash([]float64{3, 1.5}, 0)
	pdf.SetLineDash(nil, 0) // solid
}

// ---- Text cells ---------------------------------------------------------

func TestCellBasic(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	if err := pdf.Cell(80, 10, "Hello", "1", 1, "L", false, ""); err != nil {
		t.Fatal(err)
	}
}

func TestCellBeforeAddPage(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetFont("Helvetica", "", 12)
	if err := pdf.Cell(80, 10, "No page", "1", 1, "L", false, ""); err == nil {
		t.Fatal("expected error: Cell called before AddPage")
	}
}

func TestCellAutoPageBreak(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	pdf.SetAutoPageBreak(true, 10)
	// Fill the page with cells until auto-break triggers
	initial := pdf.PageCount()
	for i := 0; i < 50; i++ {
		pdf.Cell(0, 10, "Row "+string(rune('A'+i%26)), "", 1, "L", false, "")
	}
	if pdf.PageCount() <= initial {
		t.Fatal("auto page break should have created a new page")
	}
}

func TestMultiCell(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 11)
	if err := pdf.MultiCell(0, 6,
		"This is a long paragraph that will be wrapped into multiple lines "+
			"by the MultiCell method automatically.",
		"", "J", false); err != nil {
		t.Fatal(err)
	}
}

func TestWrite(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	if err := pdf.Write(6, "Hello World", ""); err != nil {
		t.Fatal(err)
	}
}

func TestWriteBeforeAddPage(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetFont("Helvetica", "", 12)
	if err := pdf.Write(6, "No page", ""); err == nil {
		t.Fatal("expected error before AddPage")
	}
}

// ---- Bookmarks ----------------------------------------------------------

func TestSetBookmark(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetBookmark("Chapter 1", 0)
	pdf.SetBookmark("Section 1.1", 1)
	if pdf.TOC.Len() != 2 {
		t.Fatalf("expected 2 TOC entries, got %d", pdf.TOC.Len())
	}
}

// ---- Layers -------------------------------------------------------------

func TestStartEndLayer(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.StartLayer("Watermark", true)
	pdf.SetFont("Helvetica", "", 10)
	pdf.EndLayer()
}

// ---- Transforms ---------------------------------------------------------

func TestStartStopTransform(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.StartTransform()
	pdf.ScaleXY(1.5, 1.5, 105, 148)
	pdf.StopTransform()
}

func TestRotate(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.StartTransform()
	pdf.Rotate(45, 105, 148)
	pdf.StopTransform()
}

// ---- JavaScript ---------------------------------------------------------

func TestAppendJavascript(t *testing.T) {
	pdf := newPDF(t)
	pdf.AppendJavascript("app.alert('test');")
	if !strings.Contains(pdf.JS.RawJS(), "app.alert") {
		t.Fatal("JS not appended")
	}
}

// ---- Encryption ---------------------------------------------------------

func TestSetEncryption(t *testing.T) {
	pdf := newPDF(t)
	err := pdf.SetEncryption(encrypt.Config{
		Mode:          encrypt.EncAES_256,
		UserPassword:  "user",
		OwnerPassword: "owner",
		Permissions:   encrypt.PermPrint,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pdf.ClassObjects.Encrypt == nil {
		t.Fatal("encrypt should be set")
	}
}

// ---- Output  ------------------------------------------------------------

func TestGetOutPDFString(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetTitle("Test")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 10, "Hello, World!", "", 1, "C", false, "")

	data, err := pdf.GetOutPDFString()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output should be a valid PDF")
	}
	if !bytes.Contains(data, []byte("%%EOF")) {
		t.Fatal("output should end with EOF marker")
	}
}

func TestOutput(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 10, "Output test", "", 1, "C", false, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("Output should write valid PDF")
	}
}

func TestSavePDF(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 10, "Save test", "", 1, "L", false, "")

	path := t.TempDir() + "/save_test.pdf"
	if err := pdf.SavePDF(path); err != nil {
		t.Fatal(err)
	}
}

// ---- Default cell/CSS ---------------------------------------------------

func TestDefaultCellSettings(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetDefaultCellPadding(1, 2, 1, 2)
	pdf.SetDefaultCellMargin(0, 1, 0, 1)
	pdf.SetDefaultCellBorderPos(0)
}

func TestDefaultCSSSettings(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetDefaultCSSMargin(0, 0, 4, 0)
	pdf.SetDefaultCSSPadding(2, 4, 2, 4)
	pdf.SetDefaultCSSBorderSpacing(2, 2)
}

func TestSetMargins(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetMargins(15, 20, 15)
}

func TestSetAutoPageBreak(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetAutoPageBreak(false, 0)
}

// ---- Hyphenation --------------------------------------------------------

func TestLoadTexHyphenPatterns(t *testing.T) {
	pdf := newPDF(t)
	pdf.LoadTexHyphenPatterns(".un3 \nhy3ph \n")
}

// ---- RTL ----------------------------------------------------------------

func TestSetRTL(t *testing.T) {
	pdf := newPDF(t)
	pdf.SetRTL(true)
	if !pdf.Base.IsRTL {
		t.Fatal("RTL should be enabled")
	}
	pdf.SetRTL(false)
	if pdf.Base.IsRTL {
		t.Fatal("RTL should be disabled")
	}
}

// ---- Full document round-trip -------------------------------------------

func TestFullDocumentRoundTrip(t *testing.T) {
	pdf, err := New(classobjects.Config{
		Unit:        "mm",
		Format:      "A4",
		Orientation: page.Portrait,
		Margins:     page.Margins{Top: 15, Right: 15, Bottom: 20, Left: 15},
		SubsetFonts: true,
		Compress:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	pdf.SetTitle("Full Round Trip Test").
		SetAuthor("Test Suite").
		SetSubject("Integration Test").
		SetLanguage("en-US")

	// Page 1 — text
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 20)
	pdf.Cell(0, 12, "go-tcpdf Integration Test", "", 1, "C", false, "")
	pdf.Ln(4)

	pdf.SetFont("Times", "", 11)
	pdf.MultiCell(0, 6,
		"This document tests the complete rendering pipeline from API calls through "+
			"PDF binary serialization.",
		"", "J", false)
	pdf.Ln(4)

	// Table
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(color.NewGray(0.8))
	for _, col := range []string{"Name", "Value", "Status"} {
		pdf.Cell(55, 7, col, "1", 0, "C", true, "")
	}
	pdf.Ln(0)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetFillColor(color.White)
	rows := [][3]string{
		{"Alpha", "100", "OK"},
		{"Beta", "200", "WARN"},
		{"Gamma", "300", "OK"},
	}
	for i, row := range rows {
		fill := i%2 == 0
		pdf.SetFillColor(color.NewGray(map[bool]float64{true: 0.95, false: 1.0}[fill]))
		for _, cell := range row {
			pdf.Cell(55, 6, cell, "1", 0, "C", fill, "")
		}
		pdf.Ln(0)
	}
	pdf.Ln(6)

	// Drawing
	pdf.SetDrawColor(color.NewRGB(0.12, 0.29, 0.49))
	pdf.SetFillColor(color.NewRGB(0.7, 0.85, 0.95))
	pdf.SetLineWidth(0.5)
	pdf.Rect(15, pdf.GetY(), 80, 30, "FD")
	pdf.Circle(140, pdf.GetY()+15, 12, "FD")
	pdf.Ln(40)

	// Bookmark
	pdf.SetBookmark("Appendix", 0)

	// Page 2
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, "Page 2 — Appendix", "", 1, "L", false, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.Write(6, "Additional content on page 2.", "")

	// Render
	data, err := pdf.GetOutPDFString()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output not a valid PDF")
	}
	if !bytes.Contains(data, []byte("%%EOF")) {
		t.Fatal("output missing EOF marker")
	}
	// Should have 2 pages
	if pdf.PageCount() != 2 {
		t.Fatalf("expected 2 pages, got %d", pdf.PageCount())
	}
}

// ---- WriteTo (streaming) ------------------------------------------------

func TestWriteToImplementsWriterTo(t *testing.T) {
	pdf := newPDF(t)
	var _ io.WriterTo = pdf
}

func TestWriteToBasic(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 10, "Streaming test", "", 1, "L", false, "")

	var buf bytes.Buffer
	n, err := pdf.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("WriteTo should return non-zero byte count")
	}
	if n != int64(buf.Len()) {
		t.Fatalf("WriteTo reported %d but buffer holds %d", n, buf.Len())
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("WriteTo output is not a valid PDF")
	}
}

func TestWriteToHTTPResponseWriter(t *testing.T) {
	// Simulate the HTTP handler use-case with a bytes.Buffer
	// (http.ResponseWriter is also an io.Writer)
	pdf := newPDF(t)
	pdf.SetTitle("HTTP Response PDF")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 10, "Delivered over HTTP", "", 1, "C", false, "")

	var responseBody bytes.Buffer
	// Simulate: w.Header().Set("Content-Type", "application/pdf")
	n, err := pdf.WriteTo(&responseBody)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(responseBody.Bytes(), []byte("%PDF-")) {
		t.Fatal("HTTP response body is not a valid PDF")
	}
	if n != int64(responseBody.Len()) {
		t.Fatalf("byte count mismatch: %d vs %d", n, responseBody.Len())
	}
}

func TestWriteToPipe(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 6, "Pipe streaming", "", 1, "L", false, "")

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	var received bytes.Buffer
	go func() {
		_, e := io.Copy(&received, pr)
		errCh <- e
	}()

	n, err := pdf.WriteTo(pw)
	pw.Close()
	if err != nil {
		t.Fatal(err)
	}
	if copyErr := <-errCh; copyErr != nil {
		t.Fatal(copyErr)
	}
	if n != int64(received.Len()) {
		t.Fatalf("pipe: wrote %d, received %d bytes", n, received.Len())
	}
	if !bytes.HasPrefix(received.Bytes(), []byte("%PDF-")) {
		t.Fatal("piped PDF is not valid")
	}
}

func TestWriteToFile(t *testing.T) {
	pdf := newPDF(t)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 10, "File streaming", "", 1, "L", false, "")

	path := t.TempDir() + "/streamed.pdf"
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	n, err := pdf.WriteTo(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Size() != n {
		t.Fatalf("file size %d != WriteTo count %d", info.Size(), n)
	}
}

func TestWriteToAndOutputEquivalent(t *testing.T) {
	// Both Output and WriteTo must produce valid PDFs (content is
	// deterministic except for the timestamp-based document ID).
	buildPDF := func() *TCPDF {
		pdf := newPDF(t)
		pdf.SetTitle("Equivalence").SetAuthor("Test")
		pdf.AddPage()
		pdf.SetFont("Helvetica", "", 12)
		pdf.Cell(0, 10, "Same content", "", 1, "C", false, "")
		return pdf
	}

	var buf1 bytes.Buffer
	if err := buildPDF().Output(&buf1); err != nil {
		t.Fatal(err)
	}
	var buf2 bytes.Buffer
	if _, err := buildPDF().WriteTo(&buf2); err != nil {
		t.Fatal(err)
	}
	// Both must be valid PDFs with the same structure
	for _, data := range [][]byte{buf1.Bytes(), buf2.Bytes()} {
		if !bytes.HasPrefix(data, []byte("%PDF-")) {
			t.Fatal("output is not a valid PDF")
		}
		if !bytes.Contains(data, []byte("/Catalog")) {
			t.Fatal("output missing /Catalog")
		}
		if !bytes.Contains(data, []byte("%%EOF")) {
			t.Fatal("output missing EOF marker")
		}
	}
}
