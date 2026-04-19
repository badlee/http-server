// Package pdfbuf provides a PDF-aware string builder that avoids fmt.Sprintf
// allocations for the hot paths in stream generation.
// All numeric formatting routines live here so every package can import one
// consistent, allocation-conscious helper.
package pdfbuf

import (
	"math"
	"strconv"
	"strings"
)

// Buf is a reusable PDF stream builder backed by a strings.Builder.
// The zero value is ready to use.
type Buf struct {
	b strings.Builder
}

// Reset clears the buffer for reuse without freeing the underlying memory.
func (b *Buf) Reset() { b.b.Reset() }

// String returns the accumulated string.
func (b *Buf) String() string { return b.b.String() }

// Len returns the number of bytes written.
func (b *Buf) Len() int { return b.b.Len() }

// ---- Write helpers ------------------------------------------------------

// S writes a raw string.
func (b *Buf) S(s string) { b.b.WriteString(s) }

// B writes a single byte.
func (b *Buf) B(c byte) { b.b.WriteByte(c) }

// NL writes a newline.
func (b *Buf) NL() { b.b.WriteByte('\n') }

// SP writes a space.
func (b *Buf) SP() { b.b.WriteByte(' ') }

// ---- Numeric helpers (no fmt, no heap) ----------------------------------

// F writes a PDF-formatted float64 (no trailing zeros, no scientific notation).
func (b *Buf) F(v float64) {
	b.b.WriteString(FmtF(v))
}

// I writes a decimal int.
func (b *Buf) I(v int) {
	b.b.WriteString(strconv.Itoa(v))
}

// I64 writes a decimal int64.
func (b *Buf) I64(v int64) {
	b.b.WriteString(strconv.FormatInt(v, 10))
}

// Hex writes bytes as lowercase hex.
func (b *Buf) Hex(data []byte) {
	const hx = "0123456789abcdef"
	for _, c := range data {
		b.b.WriteByte(hx[c>>4])
		b.b.WriteByte(hx[c&0xF])
	}
}

// ---- PDF primitives -----------------------------------------------------

// ObjHeader writes  "n 0 obj\n".
func (b *Buf) ObjHeader(n int) {
	b.I(n)
	b.b.WriteString(" 0 obj\n")
}

// ObjFooter writes "endobj\n".
func (b *Buf) ObjFooter() { b.b.WriteString("endobj\n") }

// Ref writes "n 0 R".
func (b *Buf) Ref(n int) {
	b.I(n)
	b.b.WriteString(" 0 R")
}

// RefNL writes "n 0 R\n".
func (b *Buf) RefNL(n int) {
	b.Ref(n)
	b.b.WriteByte('\n')
}

// DictKey writes "/Key ".
func (b *Buf) DictKey(key string) {
	b.b.WriteByte('/')
	b.b.WriteString(key)
	b.b.WriteByte('\n')
}

// DictKeyVal writes "/Key value\n".
func (b *Buf) DictKeyVal(key, value string) {
	b.b.WriteByte('/')
	b.b.WriteString(key)
	b.b.WriteByte(' ')
	b.b.WriteString(value)
	b.b.WriteByte('\n')
}

// DictKeyRef writes "/Key n 0 R\n".
func (b *Buf) DictKeyRef(key string, n int) {
	b.b.WriteByte('/')
	b.b.WriteString(key)
	b.b.WriteByte(' ')
	b.I(n)
	b.b.WriteString(" 0 R\n")
}

// DictKeyF writes "/Key fval\n".
func (b *Buf) DictKeyF(key string, v float64) {
	b.b.WriteByte('/')
	b.b.WriteString(key)
	b.b.WriteByte(' ')
	b.F(v)
	b.b.WriteByte('\n')
}

// DictKeyI writes "/Key ival\n".
func (b *Buf) DictKeyI(key string, v int) {
	b.b.WriteByte('/')
	b.b.WriteString(key)
	b.b.WriteByte(' ')
	b.I(v)
	b.b.WriteByte('\n')
}

// Rect writes "[llx lly urx ury]".
func (b *Buf) Rect(llx, lly, urx, ury float64) {
	b.b.WriteByte('[')
	b.F(llx)
	b.b.WriteByte(' ')
	b.F(lly)
	b.b.WriteByte(' ')
	b.F(urx)
	b.b.WriteByte(' ')
	b.F(ury)
	b.b.WriteByte(']')
}

// XRefEntry writes a 20-byte xref entry "nnnnnnnnnn ggggg n \r\n".
func (b *Buf) XRefEntry(offset int64, inUse bool) {
	// offset: 10 digits, generation: 5 digits, keyword: 1 char, space, CRLF
	off := strconv.FormatInt(offset, 10)
	for i := len(off); i < 10; i++ {
		b.b.WriteByte('0')
	}
	b.b.WriteString(off)
	if inUse {
		b.b.WriteString(" 00000 n\r\n")
	} else {
		b.b.WriteString(" 65535 f\r\n")
	}
}

// PDFStr writes a PDF literal string "(escaped)".
func (b *Buf) PDFStr(s string) {
	b.b.WriteByte('(')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			b.b.WriteString(`\(`)
		case ')':
			b.b.WriteString(`\)`)
		case '\\':
			b.b.WriteString(`\\`)
		case '\r':
			b.b.WriteString(`\r`)
		case '\n':
			b.b.WriteString(`\n`)
		default:
			b.b.WriteByte(s[i])
		}
	}
	b.b.WriteByte(')')
}

// ---- Standalone numeric helpers (package-level, no receiver) -----------

// FmtF formats a float64 for PDF: integer form when possible, otherwise
// up to 6 decimal places with trailing zeros stripped.
// No heap allocation for values representable as int64.
func FmtF(v float64) string {
	if v == math.Trunc(v) && v >= math.MinInt64 && v <= math.MaxInt64 {
		return strconv.FormatInt(int64(v), 10)
	}
	s := strconv.FormatFloat(v, 'f', 6, 64)
	// trim trailing zeros after the decimal point
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// FmtI formats an int without fmt.
func FmtI(v int) string { return strconv.Itoa(v) }

// FmtI64 formats an int64 without fmt.
func FmtI64(v int64) string { return strconv.FormatInt(v, 10) }

// HexStr encodes bytes as a lowercase hex string.
func HexStr(data []byte) string {
	const hx = "0123456789abcdef"
	out := make([]byte, len(data)*2)
	for i, c := range data {
		out[i*2] = hx[c>>4]
		out[i*2+1] = hx[c&0xF]
	}
	return string(out)
}

// EscapePDFString returns s with PDF special chars escaped, without allocating
// unless escaping is actually needed.
func EscapePDFString(s string) string {
	// Fast path: no special chars
	needsEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '(' || s[i] == ')' || s[i] == '\\' || s[i] == '\r' || s[i] == '\n' {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		switch s[i] {
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
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
