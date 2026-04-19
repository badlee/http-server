package pdfbuf

import (
	"strings"
	"testing"
)

func TestFmtF(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"}, {1, "1"}, {-1, "-1"}, {100, "100"},
		{595.28, "595.28"}, {1.5, "1.5"}, {0.5, "0.5"},
		{1.000001, "1.000001"}, {1.5000, "1.5"}, {-595.28, "-595.28"},
	}
	for _, tc := range tests {
		got := FmtF(tc.in)
		if got != tc.want {
			t.Errorf("FmtF(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFmtFNoTrailingZeros(t *testing.T) {
	for _, v := range []float64{1.10, 1.100, 2.50000, 0.30} {
		s := FmtF(v)
		if strings.HasSuffix(s, "0") {
			t.Errorf("FmtF(%v) = %q has trailing zero", v, s)
		}
	}
}

func TestFmtI(t *testing.T) {
	tests := map[int]string{0: "0", 1: "1", -1: "-1", 999: "999"}
	for in, want := range tests {
		if got := FmtI(in); got != want {
			t.Errorf("FmtI(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFmtI64(t *testing.T) {
	if got := FmtI64(1234567890123); got != "1234567890123" {
		t.Fatalf("FmtI64: %q", got)
	}
}

func TestHexStr(t *testing.T) {
	if got := HexStr([]byte{0xDE, 0xAD, 0xBE, 0xEF}); got != "deadbeef" {
		t.Fatalf("HexStr: %q", got)
	}
	if HexStr(nil) != "" {
		t.Fatal("HexStr(nil) should be empty")
	}
}

func TestEscapePDFString(t *testing.T) {
	tests := []struct{ in, want string }{
		{"hello", "hello"},
		{"(test)", `\(test\)`},
		{`a\b`, `a\\b`},
		{"cr\rn\n", `cr\rn\n`},
	}
	for _, tc := range tests {
		got := EscapePDFString(tc.in)
		if got != tc.want {
			t.Errorf("EscapePDFString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEscapePDFStringNoAlloc(t *testing.T) {
	// Plain strings should not allocate (fast path)
	s := "hello world plain"
	got := EscapePDFString(s)
	if got != s {
		t.Fatalf("plain string modified: %q", got)
	}
}

func TestBufS(t *testing.T) {
	var b Buf
	b.S("hello")
	b.S(" world")
	if b.String() != "hello world" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufReset(t *testing.T) {
	var b Buf
	b.S("data")
	b.Reset()
	if b.Len() != 0 {
		t.Fatal("Reset should clear buffer")
	}
}

func TestBufF(t *testing.T) {
	var b Buf
	b.F(595.28)
	if b.String() != "595.28" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufI(t *testing.T) {
	var b Buf
	b.I(42)
	if b.String() != "42" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufI64(t *testing.T) {
	var b Buf
	b.I64(9876543210)
	if b.String() != "9876543210" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufHex(t *testing.T) {
	var b Buf
	b.Hex([]byte{0xCA, 0xFE})
	if b.String() != "cafe" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufObjHeader(t *testing.T) {
	var b Buf
	b.ObjHeader(5)
	if b.String() != "5 0 obj\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufObjFooter(t *testing.T) {
	var b Buf
	b.ObjFooter()
	if b.String() != "endobj\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufRef(t *testing.T) {
	var b Buf
	b.Ref(7)
	if b.String() != "7 0 R" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufRefNL(t *testing.T) {
	var b Buf
	b.RefNL(3)
	if b.String() != "3 0 R\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufDictKeyVal(t *testing.T) {
	var b Buf
	b.DictKeyVal("Type", "/Page")
	if b.String() != "/Type /Page\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufDictKeyRef(t *testing.T) {
	var b Buf
	b.DictKeyRef("Parent", 4)
	if b.String() != "/Parent 4 0 R\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufDictKeyF(t *testing.T) {
	var b Buf
	b.DictKeyF("Ascent", 718)
	if b.String() != "/Ascent 718\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufDictKeyI(t *testing.T) {
	var b Buf
	b.DictKeyI("Count", 3)
	if b.String() != "/Count 3\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestBufRect(t *testing.T) {
	var b Buf
	b.Rect(0, 0, 595.28, 841.89)
	s := b.String()
	if !strings.HasPrefix(s, "[0 0 595.28") {
		t.Fatalf("got %q", s)
	}
}

func TestBufXRefEntryInUse(t *testing.T) {
	var b Buf
	b.XRefEntry(1234, true)
	got := b.String()
	if len(got) != 20 {
		t.Fatalf("xref entry should be 20 bytes, got %d: %q", len(got), got)
	}
	if !strings.Contains(got, "n") {
		t.Fatal("in-use entry should contain 'n'")
	}
}

func TestBufXRefEntryFree(t *testing.T) {
	var b Buf
	b.XRefEntry(0, false)
	got := b.String()
	if len(got) != 20 {
		t.Fatalf("free xref entry should be 20 bytes, got %d: %q", len(got), got)
	}
	if !strings.Contains(got, "f") {
		t.Fatal("free entry should contain 'f'")
	}
}

func TestBufPDFStr(t *testing.T) {
	var b Buf
	b.PDFStr("hello (world)")
	got := b.String()
	if got != `(hello \(world\))` {
		t.Fatalf("got %q", got)
	}
}

func TestXRefEntryOffset(t *testing.T) {
	var b Buf
	b.XRefEntry(9876543210, true)
	got := b.String()
	if !strings.HasPrefix(got, "9876543210") {
		t.Fatalf("offset not zero-padded correctly: %q", got)
	}
}

func TestXRefEntryShortOffset(t *testing.T) {
	var b Buf
	b.XRefEntry(42, true)
	got := b.String()
	if !strings.HasPrefix(got, "0000000042") {
		t.Fatalf("short offset not zero-padded: %q", got)
	}
}
