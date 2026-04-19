package font

import (
	"math"
	"testing"
)

// ---- core font metrics --------------------------------------------------

func TestCoreFontHelvetica(t *testing.T) {
	m, err := CoreFontMetrics("Helvetica")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Helvetica" {
		t.Fatalf("name: %v", m.Name)
	}
	if m.Type != FontTypeCore {
		t.Fatal("expected FontTypeCore")
	}
	if m.Ascent <= 0 {
		t.Fatal("ascent should be positive")
	}
	if m.Descent >= 0 {
		t.Fatal("descent should be negative")
	}
}

func TestCoreFontAllStandard(t *testing.T) {
	names := []string{
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Symbol", "ZapfDingbats",
	}
	for _, name := range names {
		if _, err := CoreFontMetrics(name); err != nil {
			t.Errorf("CoreFontMetrics(%q): %v", name, err)
		}
	}
}

func TestCoreFontUnknown(t *testing.T) {
	if _, err := CoreFontMetrics("NotAFont"); err == nil {
		t.Fatal("expected error for unknown font")
	}
}

func TestCoreFontIndependentCopies(t *testing.T) {
	m1, _ := CoreFontMetrics("Helvetica")
	m2, _ := CoreFontMetrics("Helvetica")
	// Modifying one should not affect the other
	m1.Widths['A'] = 9999
	if m2.Widths['A'] == 9999 {
		t.Fatal("CoreFontMetrics should return independent copies")
	}
}

// ---- glyph width --------------------------------------------------------

func TestGlyphWidth(t *testing.T) {
	m, _ := CoreFontMetrics("Helvetica")
	w := m.GlyphWidth(' ')
	if w <= 0 {
		t.Fatalf("space width should be positive, got %v", w)
	}
}

func TestGlyphWidthMissing(t *testing.T) {
	m, _ := CoreFontMetrics("Helvetica")
	// Rune with no entry in map
	w := m.GlyphWidth('\u2603') // snowman
	if w != m.MissingWidth {
		t.Fatalf("missing rune should return MissingWidth: want %v got %v", m.MissingWidth, w)
	}
}

// ---- string width -------------------------------------------------------

func TestStringWidth(t *testing.T) {
	m, _ := CoreFontMetrics("Courier")
	// Courier is monospaced at 600/1000 em
	w := m.StringWidth("AAAA", 10)
	want := 4 * 600.0 * 10 / 1000
	if math.Abs(w-want) > 1e-6 {
		t.Fatalf("Courier 'AAAA' at 10pt: want %v got %v", want, w)
	}
}

func TestStringWidthEmpty(t *testing.T) {
	m, _ := CoreFontMetrics("Helvetica")
	w := m.StringWidth("", 12)
	if w != 0 {
		t.Fatalf("empty string width should be 0, got %v", w)
	}
}

// ---- mark used ----------------------------------------------------------

func TestMarkUsed(t *testing.T) {
	m, _ := CoreFontMetrics("Helvetica")
	m.MarkStringUsed("Hello")
	for _, r := range "Hello" {
		if !m.UsedGlyphs[r] {
			t.Fatalf("rune %q not marked as used", r)
		}
	}
}

// ---- font stack ---------------------------------------------------------

func TestStackLoadCore(t *testing.T) {
	s := NewStack(true)
	rk, err := s.LoadCore("Helvetica")
	if err != nil {
		t.Fatal(err)
	}
	if rk == "" {
		t.Fatal("resource key should not be empty")
	}
}

func TestStackLoadCoreDeduplicate(t *testing.T) {
	s := NewStack(true)
	rk1, _ := s.LoadCore("Helvetica")
	rk2, _ := s.LoadCore("Helvetica")
	if rk1 != rk2 {
		t.Fatalf("same font should return same key: %v vs %v", rk1, rk2)
	}
}

func TestStackGet(t *testing.T) {
	s := NewStack(false)
	rk, _ := s.LoadCore("Times-Roman")
	m, ok := s.Get(rk)
	if !ok {
		t.Fatal("Get should find loaded font")
	}
	if m.Name != "Times-Roman" {
		t.Fatalf("name: %v", m.Name)
	}
}

func TestStackGetMissing(t *testing.T) {
	s := NewStack(false)
	if _, ok := s.Get("F999"); ok {
		t.Fatal("Get should return false for unknown key")
	}
}

func TestStackAll(t *testing.T) {
	s := NewStack(false)
	s.LoadCore("Helvetica")
	s.LoadCore("Times-Roman")
	s.LoadCore("Courier")
	if len(s.All()) != 3 {
		t.Fatalf("expected 3 fonts, got %d", len(s.All()))
	}
}

func TestStackTextWidth(t *testing.T) {
	s := NewStack(false)
	rk, _ := s.LoadCore("Courier")
	w := s.TextWidth(rk, "AAAA", 10)
	want := 4 * 600.0 * 10 / 1000
	if math.Abs(w-want) > 1e-6 {
		t.Fatalf("want %v got %v", want, w)
	}
}

func TestStackMarkUsed(t *testing.T) {
	s := NewStack(true)
	rk, _ := s.LoadCore("Helvetica")
	s.MarkUsed(rk, "ABC")
	m, _ := s.Get(rk)
	if !m.UsedGlyphs['A'] {
		t.Fatal("A should be marked used")
	}
}

func TestStackSubsetEnabled(t *testing.T) {
	s1 := NewStack(true)
	if !s1.SubsetEnabled() {
		t.Fatal("subset should be enabled")
	}
	s2 := NewStack(false)
	if s2.SubsetEnabled() {
		t.Fatal("subset should be disabled")
	}
}

// ---- line height helpers -------------------------------------------------

func TestLineHeight(t *testing.T) {
	m, _ := CoreFontMetrics("Helvetica")
	lh := LineHeight(m, 12)
	if lh <= 0 {
		t.Fatalf("line height should be positive, got %v", lh)
	}
}

func TestCapHeight(t *testing.T) {
	m, _ := CoreFontMetrics("Helvetica")
	ch := CapHeight(m, 12)
	lh := LineHeight(m, 12)
	if ch >= lh {
		t.Fatalf("cap height (%v) should be < line height (%v)", ch, lh)
	}
}
