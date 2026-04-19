// Package unicode provides Unicode utilities for PDF text: bidi, normalization, conversion.
// Ported from tc-lib-unicode-data (PHP) by Nicola Asuni.
package unicode

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Direction indicates text direction.
type Direction int

const (
	DirLTR Direction = iota
	DirRTL
	DirAuto
)

// Convert provides Unicode text conversion utilities.
type Convert struct {
	defaultDir Direction
}

// NewConvert creates a Convert instance.
func NewConvert(rtl bool) *Convert {
	d := DirLTR
	if rtl {
		d = DirRTL
	}
	return &Convert{defaultDir: d}
}

// IsRTL returns true if the default direction is right-to-left.
func (c *Convert) IsRTL() bool { return c.defaultDir == DirRTL }

// SetRTL changes the default document direction.
func (c *Convert) SetRTL(rtl bool) {
	if rtl {
		c.defaultDir = DirRTL
	} else {
		c.defaultDir = DirLTR
	}
}

// DetectDirection analyses a string and returns its primary direction.
func DetectDirection(s string) Direction {
	for _, r := range s {
		switch {
		case isRTLRune(r):
			return DirRTL
		case isLTRRune(r):
			return DirLTR
		}
	}
	return DirLTR
}

// Reverse reverses a UTF-8 string rune by rune (for RTL rendering).
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// UTF8ToRunes converts a UTF-8 string to a rune slice.
func UTF8ToRunes(s string) []rune {
	runes := make([]rune, 0, utf8.RuneCountInString(s))
	for _, r := range s {
		runes = append(runes, r)
	}
	return runes
}

// RunesToUTF8 converts runes back to a UTF-8 string.
func RunesToUTF8(runes []rune) string {
	return string(runes)
}

// UTF16BEBytes encodes a string as UTF-16 Big Endian bytes with BOM.
// Used for PDF string objects.
func UTF16BEBytes(s string) []byte {
	runes := []rune(s)
	out := make([]byte, 2+len(runes)*2)
	out[0] = 0xFE; out[1] = 0xFF // BOM
	for i, r := range runes {
		out[2+i*2] = byte(r >> 8)
		out[2+i*2+1] = byte(r)
	}
	return out
}

// EscapePDFString escapes a raw byte string for use inside PDF literal string ( ).
func EscapePDFString(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch c {
		case '(':
			b.WriteString(`\(`)
		case ')':
			b.WriteString(`\)`)
		case '\\':
			b.WriteString(`\\`)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// ToHexString encodes bytes as a PDF hex string < >.
func ToHexString(data []byte) string {
	const hex = "0123456789abcdef"
	var b strings.Builder
	for _, c := range data {
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0xF])
	}
	return b.String()
}

// WordBreakPositions returns the indices in runes where line breaks are allowed.
// Breaks allowed after spaces, hyphens, and zero-width break points.
func WordBreakPositions(runes []rune) []int {
	var positions []int
	for i, r := range runes {
		if isBreakable(r) {
			positions = append(positions, i+1) // break after this rune
		}
	}
	return positions
}

// ---- Hyphenation (TeX patterns) -----------------------------------------

// Hyphenator applies TeX hyphenation patterns to a word.
type Hyphenator struct {
	patterns map[string]string
}

// NewHyphenator creates a Hyphenator with the given patterns.
// patterns maps a letter sequence to its hyphenation weight string.
func NewHyphenator(patterns map[string]string) *Hyphenator {
	return &Hyphenator{patterns: patterns}
}

// Hyphenate returns suggested hyphenation points (indices into runes).
func (h *Hyphenator) Hyphenate(word string) []int {
	if h == nil || len(h.patterns) == 0 {
		return nil
	}
	word = strings.ToLower(word)
	runes := []rune("." + word + ".")
	n := len(runes)
	levels := make([]int, n+1)

	for start := 0; start < n; start++ {
		for end := start + 1; end <= n; end++ {
			sub := string(runes[start:end])
			if pat, ok := h.patterns[sub]; ok {
				for i, ch := range pat {
					if ch >= '0' && ch <= '9' {
						lv := int(ch - '0')
						pos := start + i
						if pos < len(levels) && lv > levels[pos] {
							levels[pos] = lv
						}
					}
				}
			}
		}
	}

	var points []int
	// Offset by 1 to account for leading '.'
	for i := 2; i < len(levels)-2; i++ {
		if levels[i]%2 == 1 {
			points = append(points, i-1) // index in original word
		}
	}
	return points
}

// ---- helpers ------------------------------------------------------------

func isRTLRune(r rune) bool {
	return unicode.Is(unicode.Arabic, r) ||
		unicode.Is(unicode.Hebrew, r) ||
		unicode.Is(unicode.Syriac, r) ||
		unicode.Is(unicode.Thaana, r)
}

func isLTRRune(r rune) bool {
	return unicode.IsLetter(r) && !isRTLRune(r)
}

func isBreakable(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
		r == '-' || r == '\u00AD' || // soft hyphen
		r == '\u200B' // zero-width space
}
