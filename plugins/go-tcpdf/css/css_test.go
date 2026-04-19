package css

import (
	"testing"
)

func TestParseProperties(t *testing.T) {
	props := ParseProperties("color: red; font-size: 12pt; font-weight: bold")
	if props["color"] != "red" {
		t.Fatalf("color: %q", props["color"])
	}
	if props["font-size"] != "12pt" {
		t.Fatalf("font-size: %q", props["font-size"])
	}
	if props["font-weight"] != "bold" {
		t.Fatalf("font-weight: %q", props["font-weight"])
	}
}

func TestParsePropertiesEmpty(t *testing.T) {
	props := ParseProperties("")
	if len(props) != 0 {
		t.Fatalf("expected empty map, got %v", props)
	}
}

func TestParsePropertiesTrailingSemicolon(t *testing.T) {
	props := ParseProperties("color: red;")
	if props["color"] != "red" {
		t.Fatalf("trailing semicolon: %q", props["color"])
	}
}

func TestParseBoxValues1(t *testing.T) {
	bs := ParseBoxValues("10px", 1)
	if bs.Top != bs.Right || bs.Right != bs.Bottom || bs.Bottom != bs.Left {
		t.Fatal("1-value: all sides should be equal")
	}
}

func TestParseBoxValues2(t *testing.T) {
	bs := ParseBoxValues("10px 20px", 1)
	if bs.Top != bs.Bottom {
		t.Fatal("2-value: Top and Bottom should be equal")
	}
	if bs.Right != bs.Left {
		t.Fatal("2-value: Right and Left should be equal")
	}
	if bs.Top == bs.Right {
		t.Fatal("2-value: Top and Right should differ")
	}
}

func TestParseBoxValues4(t *testing.T) {
	bs := ParseBoxValues("1px 2px 3px 4px", 1)
	if bs.Top == bs.Right || bs.Right == bs.Bottom {
		t.Fatal("4-value: all sides should differ")
	}
}

func TestFontWeight(t *testing.T) {
	if !FontWeight("bold") {
		t.Fatal("'bold' should be bold")
	}
	if !FontWeight("700") {
		t.Fatal("'700' should be bold")
	}
	if FontWeight("normal") {
		t.Fatal("'normal' should not be bold")
	}
	if FontWeight("400") {
		t.Fatal("'400' should not be bold")
	}
}

func TestFontStyle(t *testing.T) {
	if !FontStyle("italic") {
		t.Fatal("'italic' should be italic")
	}
	if !FontStyle("oblique") {
		t.Fatal("'oblique' should be italic")
	}
	if FontStyle("normal") {
		t.Fatal("'normal' should not be italic")
	}
}

func TestTextDecoration(t *testing.T) {
	u, lt, ol := TextDecoration("underline line-through overline")
	if !u {
		t.Fatal("expected underline")
	}
	if !lt {
		t.Fatal("expected line-through")
	}
	if !ol {
		t.Fatal("expected overline")
	}
}

func TestTextDecorationNone(t *testing.T) {
	u, lt, ol := TextDecoration("none")
	if u || lt || ol {
		t.Fatal("'none' should produce no decorations")
	}
}

func TestTextAlign(t *testing.T) {
	tests := map[string]string{
		"left":    "L",
		"right":   "R",
		"center":  "C",
		"justify": "J",
		"CENTER":  "C",
	}
	for input, want := range tests {
		got := TextAlign(input)
		if got != want {
			t.Errorf("TextAlign(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTextAlignUnknown(t *testing.T) {
	got := TextAlign("inherit")
	if got != "" {
		t.Fatalf("unknown align should return empty, got %q", got)
	}
}

func TestVerticalAlign(t *testing.T) {
	tests := map[string]string{
		"top":    "T",
		"middle": "C",
		"bottom": "B",
	}
	for input, want := range tests {
		got := VerticalAlign(input)
		if got != want {
			t.Errorf("VerticalAlign(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBorderStyleSolid(t *testing.T) {
	w, style, _ := BorderStyle("1px solid #000000")
	if w <= 0 {
		t.Fatalf("border width: %v", w)
	}
	if style != "solid" {
		t.Fatalf("style: %q", style)
	}
}

func TestBorderStyleNone(t *testing.T) {
	w, style, _ := BorderStyle("none")
	if w != 0 {
		t.Fatalf("none border should have width 0, got %v", w)
	}
	if style != "none" {
		t.Fatalf("style: %q", style)
	}
}

func TestToCellBound(t *testing.T) {
	bs := BoxSpacing{Top: 1, Right: 2, Bottom: 3, Left: 4}
	cb := ToCellBound(bs)
	if cb.T != 1 || cb.R != 2 || cb.B != 3 || cb.L != 4 {
		t.Fatalf("ToCellBound: %+v", cb)
	}
}

func TestCSSDefaults(t *testing.T) {
	c := New()
	c.SetDefaultCSSMargin(1, 2, 3, 4)
	if c.DefaultMargin.Top != 1 || c.DefaultMargin.Right != 2 {
		t.Fatalf("margin: %+v", c.DefaultMargin)
	}
	c.SetDefaultCSSPadding(5, 6, 7, 8)
	if c.DefaultPadding.Bottom != 7 {
		t.Fatalf("padding: %+v", c.DefaultPadding)
	}
	c.SetDefaultCSSBorderSpacing(2, 4)
	if c.DefaultBorderSpacing.Top != 2 || c.DefaultBorderSpacing.Right != 4 {
		t.Fatalf("border-spacing: %+v", c.DefaultBorderSpacing)
	}
}
