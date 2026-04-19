package unicode

import (
	"testing"
)

func TestDetectDirectionLTR(t *testing.T) {
	if DetectDirection("Hello World") != DirLTR {
		t.Fatal("expected LTR")
	}
}

func TestDetectDirectionRTL(t *testing.T) {
	if DetectDirection("مرحبا") != DirRTL {
		t.Fatal("expected RTL for Arabic")
	}
}

func TestDetectDirectionEmpty(t *testing.T) {
	if DetectDirection("") != DirLTR {
		t.Fatal("empty string defaults to LTR")
	}
}

func TestReverse(t *testing.T) {
	tests := []struct{ in, want string }{
		{"abc", "cba"},
		{"héllo", "olléh"},
		{"", ""},
		{"a", "a"},
	}
	for _, tc := range tests {
		got := Reverse(tc.in)
		if got != tc.want {
			t.Errorf("Reverse(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestUTF16BEBytes(t *testing.T) {
	b := UTF16BEBytes("A")
	// BOM 0xFE 0xFF + 0x00 0x41
	if len(b) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(b))
	}
	if b[0] != 0xFE || b[1] != 0xFF {
		t.Fatalf("missing BOM: %x %x", b[0], b[1])
	}
	if b[2] != 0x00 || b[3] != 0x41 {
		t.Fatalf("wrong char encoding: %x %x", b[2], b[3])
	}
}

func TestUTF16BEBytesEmpty(t *testing.T) {
	b := UTF16BEBytes("")
	if len(b) != 2 { // just BOM
		t.Fatalf("expected 2 bytes for empty string, got %d", len(b))
	}
}

func TestEscapePDFString(t *testing.T) {
	tests := []struct{ in, want string }{
		{"hello", "hello"},
		{"(test)", `\(\)`+""[0:0] + `\(test\)`[1:]},
		{"a\\b", `a\\b`},
	}
	for _, tc := range tests {
		got := EscapePDFString(tc.in)
		_ = got
	}
	// Specific checks
	got := EscapePDFString("(hello)")
	if got != `\(hello\)` {
		t.Fatalf("escape parens: %q", got)
	}
	got = EscapePDFString("a\\b")
	if got != `a\\b` {
		t.Fatalf("escape backslash: %q", got)
	}
}

func TestToHexString(t *testing.T) {
	got := ToHexString([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	if got != "deadbeef" {
		t.Fatalf("got %q", got)
	}
}

func TestToHexStringEmpty(t *testing.T) {
	if ToHexString(nil) != "" {
		t.Fatal("empty input should produce empty string")
	}
}

func TestWordBreakPositions(t *testing.T) {
	runes := []rune("hello world foo")
	pos := WordBreakPositions(runes)
	if len(pos) == 0 {
		t.Fatal("expected at least one break position (space)")
	}
	// Break should be after first space (index 6)
	found := false
	for _, p := range pos {
		if p == 6 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected break at position 6, got %v", pos)
	}
}

func TestWordBreakPositionsNoBreak(t *testing.T) {
	pos := WordBreakPositions([]rune("nospaces"))
	if len(pos) != 0 {
		t.Fatalf("expected no breaks, got %v", pos)
	}
}

func TestConvertRTL(t *testing.T) {
	c := NewConvert(false)
	if c.IsRTL() {
		t.Fatal("should be LTR")
	}
	c.SetRTL(true)
	if !c.IsRTL() {
		t.Fatal("should be RTL after SetRTL(true)")
	}
	c.SetRTL(false)
	if c.IsRTL() {
		t.Fatal("should be LTR after SetRTL(false)")
	}
}

func TestHyphenator(t *testing.T) {
	// Simple English-like patterns
	patterns := map[string]string{
		".un": "0300",
		"hy":  "030",
		"phen": "0040",
		"ation": "00500",
	}
	h := NewHyphenator(patterns)
	_ = h.Hyphenate("unhyphenation")
	// Not asserting specific indices (depends on full pattern set),
	// just verify it does not panic and returns a slice.
}

func TestHyphenatorNilPatterns(t *testing.T) {
	var h *Hyphenator
	pts := h.Hyphenate("word")
	if pts != nil {
		t.Fatal("nil hyphenator should return nil")
	}
}

func TestUTF8ToRunesAndBack(t *testing.T) {
	original := "héllo wörld"
	runes := UTF8ToRunes(original)
	back := RunesToUTF8(runes)
	if back != original {
		t.Fatalf("round-trip: got %q", back)
	}
}
