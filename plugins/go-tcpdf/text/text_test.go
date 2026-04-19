package text

import (
	"strings"
	"testing"

	internalFont "github.com/tecnickcom/go-tcpdf/font"
	internalUni "github.com/tecnickcom/go-tcpdf/unicode"
)

func newText() (*Text, string) {
	fonts := internalFont.NewStack(false)
	rk, _ := fonts.LoadCore("Helvetica")
	uni := internalUni.NewConvert(false)
	t := New(fonts, uni)
	t.SetFont(rk, 12)
	return t, rk
}

// ---- SetFont / GetLastBBox ----------------------------------------------

func TestSetFont(t *testing.T) {
	tx, rk := newText()
	if tx.currentFontKey != rk {
		t.Fatalf("font key: %q", tx.currentFontKey)
	}
	if tx.currentFontSize != 12 {
		t.Fatalf("font size: %v", tx.currentFontSize)
	}
}

func TestGetLastBBoxDefault(t *testing.T) {
	tx, _ := newText()
	bbox := tx.GetLastBBox()
	// Before any rendering, should be zero
	if bbox.Llx != 0 || bbox.Lly != 0 {
		t.Fatalf("initial bbox should be zero: %+v", bbox)
	}
}

// ---- GetTextLine --------------------------------------------------------

func TestGetTextLineBasic(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{
		PosX:     10,
		PosY:     700,
		FontSize: 12,
	}
	ops := tx.GetTextLine("Hello", opts)
	if ops == "" {
		t.Fatal("GetTextLine should produce output")
	}
	if !strings.Contains(ops, "BT") {
		t.Fatal("should contain BT")
	}
	if !strings.Contains(ops, "ET") {
		t.Fatal("should contain ET")
	}
	if !strings.Contains(ops, "Tj") {
		t.Fatal("should contain Tj")
	}
}

func TestGetTextLineEmpty(t *testing.T) {
	tx, _ := newText()
	ops := tx.GetTextLine("", TextOptions{PosX: 0, PosY: 700})
	if ops != "" {
		t.Fatal("empty text should produce no output")
	}
}

func TestGetTextLineUnderline(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{
		PosX:      10,
		PosY:      700,
		FontSize:  12,
		Underline: true,
	}
	ops := tx.GetTextLine("Test", opts)
	// Should contain line operators for underline
	if !strings.Contains(ops, "S") {
		t.Fatal("underlined text should contain stroke operator")
	}
}

func TestGetTextLineLineThrough(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{
		PosX:        10,
		PosY:        700,
		FontSize:    12,
		LineThrough: true,
	}
	ops := tx.GetTextLine("Strike", opts)
	if !strings.Contains(ops, "S") {
		t.Fatal("strikethrough text should contain stroke operator")
	}
}

func TestGetTextLineStroke(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{
		PosX:        10,
		PosY:        700,
		FontSize:    12,
		Stroke:      true,
		StrokeWidth: 0.5,
	}
	ops := tx.GetTextLine("Outlined", opts)
	if !strings.Contains(ops, "q") {
		t.Fatal("stroke text should save state")
	}
	if !strings.Contains(ops, "Q") {
		t.Fatal("stroke text should restore state")
	}
}

func TestGetTextLineBBoxUpdated(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{PosX: 50, PosY: 700, FontSize: 12}
	tx.GetTextLine("Hello", opts)
	bbox := tx.GetLastBBox()
	if bbox.Llx != 50 {
		t.Fatalf("bbox Llx: want 50 got %v", bbox.Llx)
	}
	if bbox.Urx <= bbox.Llx {
		t.Fatalf("bbox Urx should be > Llx: %v vs %v", bbox.Urx, bbox.Llx)
	}
	if bbox.Ury <= bbox.Lly {
		t.Fatalf("bbox Ury should be > Lly: %v vs %v", bbox.Ury, bbox.Lly)
	}
}

// ---- GetTextCell --------------------------------------------------------

func TestGetTextCellBasic(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{
		PosX:     10,
		PosY:     700,
		Width:    100,
		Height:   20,
		HAlign:   "L",
		FontSize: 12,
	}
	ops := tx.GetTextCell("Cell text", opts)
	if ops == "" {
		t.Fatal("GetTextCell should produce output")
	}
	if !strings.Contains(ops, "Tj") {
		t.Fatal("should contain Tj")
	}
}

func TestGetTextCellCentered(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{
		PosX:     10,
		PosY:     700,
		Width:    200,
		Height:   20,
		HAlign:   "C",
		FontSize: 12,
	}
	ops := tx.GetTextCell("Centered", opts)
	if !strings.Contains(ops, "Td") {
		t.Fatal("should contain Td positioning")
	}
}

func TestGetTextCellWithBorder(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{
		PosX:     10,
		PosY:     700,
		Width:    100,
		Height:   20,
		HAlign:   "L",
		FontSize: 12,
		DrawCell: true,
	}
	ops := tx.GetTextCell("Border cell", opts)
	if !strings.Contains(ops, "re") {
		t.Fatal("cell with border should contain 're' rectangle operator")
	}
}

func TestGetTextCellJustify(t *testing.T) {
	tx, _ := newText()
	opts := TextOptions{
		PosX:     10,
		PosY:     700,
		Width:    150,
		Height:   20,
		HAlign:   "J",
		FontSize: 12,
	}
	ops := tx.GetTextCell("Justified text here", opts)
	if ops == "" {
		t.Fatal("justified cell should produce output")
	}
}

// ---- Text wrapping ------------------------------------------------------

func TestWrapTextFits(t *testing.T) {
	tx, rk := newText()
	tx.SetFont(rk, 12)
	// Very wide cell — should not wrap
	lines := tx.wrapText("Hello World", rk, 12, 1000, 0)
	if len(lines) != 1 {
		t.Fatalf("short text should not wrap: got %d lines", len(lines))
	}
}

func TestWrapTextWraps(t *testing.T) {
	tx, rk := newText()
	tx.SetFont(rk, 12)
	// Narrow cell — long text must wrap
	lines := tx.wrapText(
		"This is a long sentence that should wrap into multiple lines when the width is narrow",
		rk, 12, 50, 0)
	if len(lines) <= 1 {
		t.Fatal("long text in narrow cell should wrap into multiple lines")
	}
}

func TestWrapTextExplicitNewline(t *testing.T) {
	tx, rk := newText()
	lines := tx.wrapText("Line one\nLine two\nLine three", rk, 12, 1000, 0)
	if len(lines) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(lines))
	}
}

// ---- Hyphenation --------------------------------------------------------

func TestLoadTexHyphenPatterns(t *testing.T) {
	tx, _ := newText()
	// Minimal TeX pattern content
	content := ".un3 \nhy3ph \ntion4 \n"
	patterns := tx.LoadTexHyphenPatterns(content)
	if len(patterns) == 0 {
		t.Fatal("should parse at least some patterns")
	}
}

func TestSetTexHyphenPatterns(t *testing.T) {
	tx, _ := newText()
	content := ".un3 \nhy3ph \n"
	patterns := tx.LoadTexHyphenPatterns(content)
	tx.SetTexHyphenPatterns(patterns) // should not panic
	if tx.hyphenator == nil {
		t.Fatal("hyphenator should be set")
	}
}

// ---- EnableZeroWidthBreakPoints -----------------------------------------

func TestEnableZeroWidthBreakPoints(t *testing.T) {
	tx, _ := newText()
	tx.EnableZeroWidthBreakPoints(true)
	if !tx.breakPoints {
		t.Fatal("breakPoints should be true")
	}
	tx.EnableZeroWidthBreakPoints(false)
	if tx.breakPoints {
		t.Fatal("breakPoints should be false")
	}
}

// ---- StringWidth / CharCount --------------------------------------------

func TestStringWidth(t *testing.T) {
	tx, _ := newText()
	w := tx.StringWidth("ABC")
	if w <= 0 {
		t.Fatalf("width should be positive: %v", w)
	}
}

func TestStringWidthEmpty(t *testing.T) {
	tx, _ := newText()
	if tx.StringWidth("") != 0 {
		t.Fatal("empty string width should be 0")
	}
}

func TestCharCount(t *testing.T) {
	if CharCount("héllo") != 5 {
		t.Fatalf("expected 5 runes, got %d", CharCount("héllo"))
	}
	if CharCount("") != 0 {
		t.Fatal("empty string should have 0 chars")
	}
}
