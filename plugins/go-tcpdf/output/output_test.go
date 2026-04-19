package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/tecnickcom/go-tcpdf/base"
	"github.com/tecnickcom/go-tcpdf/color"
	"github.com/tecnickcom/go-tcpdf/font"
	imgpkg "github.com/tecnickcom/go-tcpdf/image"
	"github.com/tecnickcom/go-tcpdf/page"
	"github.com/tecnickcom/go-tcpdf/javascript"
	"github.com/tecnickcom/go-tcpdf/metainfo"
)

func newMinimalDoc() *Document {
	pon := &base.PON{}
	pm, _ := page.New(page.UnitMM, "A4", page.Portrait, page.Margins{10, 10, 10, 10})
	pm.Add("", "", nil)
	return &Document{
		Meta:         metainfo.New(),
		Pages:        pm,
		Fonts:        font.NewStack(false),
		Images:       imgpkg.NewImport(),
		Encrypt:      nil,
		Spots:        color.NewSpotRegistry(),
		JS:           javascript.New(pon),
		PageContents: []string{"BT /F1 12 Tf 100 700 Td (Hello) Tj ET\n"},
		Compress:     false,
	}
}

// ---- WriteTo (streaming) ------------------------------------------------

func TestWriteToImplementsWriterTo(t *testing.T) {
	var _ io.WriterTo = New(newMinimalDoc())
}

func TestWriteToByteCount(t *testing.T) {
	doc := newMinimalDoc()
	var buf bytes.Buffer
	n, err := New(doc).WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(buf.Len()) {
		t.Fatalf("WriteTo reported %d but buffer has %d bytes", n, buf.Len())
	}
	if n == 0 {
		t.Fatal("expected non-zero byte count")
	}
}

func TestWriteToIsValidPDF(t *testing.T) {
	var buf bytes.Buffer
	if _, err := New(newMinimalDoc()).WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("WriteTo output is not a valid PDF")
	}
	if !bytes.Contains(buf.Bytes(), []byte("%%EOF")) {
		t.Fatal("WriteTo output missing EOF marker")
	}
}

func TestWriteToStreamsViaPipe(t *testing.T) {
	// io.Pipe has no internal buffer — confirms the serializer does not
	// require a seekable or buffering writer.
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	var received bytes.Buffer
	go func() {
		_, e := io.Copy(&received, pr)
		errCh <- e
	}()
	n, writeErr := New(newMinimalDoc()).WriteTo(pw)
	pw.Close()
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	if copyErr := <-errCh; copyErr != nil {
		t.Fatal(copyErr)
	}
	if n != int64(received.Len()) {
		t.Fatalf("pipe: wrote %d, received %d", n, received.Len())
	}
	if !bytes.HasPrefix(received.Bytes(), []byte("%PDF-")) {
		t.Fatal("pipe output is not a valid PDF")
	}
}

func TestWriteToFile(t *testing.T) {
	path := t.TempDir() + "/stream.pdf"
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	n, err := New(newMinimalDoc()).WriteTo(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Size() != n {
		t.Fatalf("file size %d != reported bytes %d", info.Size(), n)
	}
}

func TestWriteToMultiplePages(t *testing.T) {
	pon := &base.PON{}
	pm, _ := page.New(page.UnitMM, "A4", page.Portrait, page.Margins{10, 10, 10, 10})
	pm.Add("", "", nil)
	pm.Add("", "", nil)
	pm.Add("", "", nil)
	doc := &Document{
		Meta:         metainfo.New(),
		Pages:        pm,
		Fonts:        font.NewStack(false),
		Images:       imgpkg.NewImport(),
		Spots:        color.NewSpotRegistry(),
		JS:           javascript.New(pon),
		PageContents: []string{"p1\n", "p2\n", "p3\n"},
		Compress:     false,
	}
	var buf bytes.Buffer
	if _, err := New(doc).WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "/Count 3") {
		t.Fatal("3-page PDF should contain /Count 3")
	}
}

func TestWriteToCompressed(t *testing.T) {
	doc := newMinimalDoc()
	doc.Compress = true
	var buf bytes.Buffer
	if _, err := New(doc).WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "FlateDecode") {
		t.Fatal("compressed streaming PDF should reference FlateDecode")
	}
}

// ---- GetOutPDFString ----------------------------------------------------

func TestGetOutPDFStringNotEmpty(t *testing.T) {
	data, err := New(newMinimalDoc()).GetOutPDFString()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("PDF output should not be empty")
	}
}

func TestGetOutPDFStringHeader(t *testing.T) {
	data, err := New(newMinimalDoc()).GetOutPDFString()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("PDF should start with %%PDF-: %q", string(data[:10]))
	}
}

func TestGetOutPDFStringFooter(t *testing.T) {
	data, err := New(newMinimalDoc()).GetOutPDFString()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "%%EOF") {
		t.Fatal("PDF should contain EOF marker")
	}
}

func TestGetOutPDFStringVersion(t *testing.T) {
	doc := newMinimalDoc()
	doc.Meta.PDFVersion = "1.4"
	data, _ := New(doc).GetOutPDFString()
	if !bytes.HasPrefix(data, []byte("%PDF-1.4")) {
		t.Fatalf("expected %%PDF-1.4 header, got %q", string(data[:15]))
	}
}

func TestGetOutPDFStringXRef(t *testing.T) {
	data, _ := New(newMinimalDoc()).GetOutPDFString()
	s := string(data)
	if !strings.Contains(s, "xref") {
		t.Fatal("PDF should contain xref table")
	}
	if !strings.Contains(s, "startxref") {
		t.Fatal("PDF should contain startxref")
	}
}

func TestGetOutPDFStringCatalog(t *testing.T) {
	data, _ := New(newMinimalDoc()).GetOutPDFString()
	s := string(data)
	if !strings.Contains(s, "/Catalog") {
		t.Fatal("PDF should contain /Catalog")
	}
	if !strings.Contains(s, "/Pages") {
		t.Fatal("PDF should contain /Pages")
	}
}

func TestGetOutPDFStringInfoDict(t *testing.T) {
	doc := newMinimalDoc()
	doc.Meta.SetTitle("Test PDF").SetAuthor("Tester")
	data, _ := New(doc).GetOutPDFString()
	s := string(data)
	if !strings.Contains(s, "/Info") {
		t.Fatal("PDF should contain /Info reference")
	}
	if !strings.Contains(s, "Test PDF") {
		t.Fatal("PDF should contain title")
	}
}

// ---- RenderPDF ----------------------------------------------------------

func TestRenderPDF(t *testing.T) {
	var buf bytes.Buffer
	if err := New(newMinimalDoc()).RenderPDF(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("RenderPDF: expected PDF header")
	}
}

// ---- SavePDF ------------------------------------------------------------

func TestSavePDF(t *testing.T) {
	if err := New(newMinimalDoc()).SavePDF(t.TempDir() + "/test.pdf"); err != nil {
		t.Fatal(err)
	}
}

func TestSavePDFFileSize(t *testing.T) {
	path := t.TempDir() + "/sized.pdf"
	if err := New(newMinimalDoc()).SavePDF(path); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Size() == 0 {
		t.Fatal("saved PDF should not be empty")
	}
}

// ---- MIME ---------------------------------------------------------------

func TestGetMIMEAttachmentPDF(t *testing.T) {
	if New(newMinimalDoc()).GetMIMEAttachmentPDF() != "application/pdf" {
		t.Fatal("unexpected MIME type")
	}
}

// ---- Fonts in output ----------------------------------------------------

func TestOutputWithFont(t *testing.T) {
	pon := &base.PON{}
	pm, _ := page.New(page.UnitMM, "A4", page.Portrait, page.Margins{10, 10, 10, 10})
	pm.Add("", "", nil)
	fonts := font.NewStack(false)
	rk, _ := fonts.LoadCore("Helvetica")
	m, _ := fonts.Get(rk)
	m.ObjNum = 0

	doc := &Document{
		Meta: metainfo.New(), Pages: pm,
		Fonts: fonts, Images: imgpkg.NewImport(),
		Spots: color.NewSpotRegistry(), JS: javascript.New(pon),
		PageContents: []string{""}, Compress: false,
	}
	data, err := New(doc).GetOutPDFString()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Helvetica") {
		t.Fatal("PDF should reference Helvetica")
	}
}

// ---- Multiple pages (GetOutPDFString) -----------------------------------

func TestOutputMultiplePages(t *testing.T) {
	pon := &base.PON{}
	pm, _ := page.New(page.UnitMM, "A4", page.Portrait, page.Margins{10, 10, 10, 10})
	pm.Add("", "", nil)
	pm.Add("", "", nil)
	pm.Add("", "", nil)
	doc := &Document{
		Meta: metainfo.New(), Pages: pm,
		Fonts: font.NewStack(false), Images: imgpkg.NewImport(),
		Spots: color.NewSpotRegistry(), JS: javascript.New(pon),
		PageContents: []string{"p1\n", "p2\n", "p3\n"}, Compress: false,
	}
	data, _ := New(doc).GetOutPDFString()
	if !strings.Contains(string(data), "/Count 3") {
		t.Fatal("PDF with 3 pages should have /Count 3")
	}
}

// ---- Compression --------------------------------------------------------

func TestOutputCompressed(t *testing.T) {
	doc := newMinimalDoc()
	doc.Compress = true
	doc.PageContents = []string{"BT /F1 12 Tf 100 700 Td (Compressed) Tj ET\n"}
	data, err := New(doc).GetOutPDFString()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("compressed PDF should have valid header")
	}
	if !strings.Contains(string(data), "FlateDecode") {
		t.Fatal("compressed PDF should contain FlateDecode filter")
	}
}

// ---- XMP metadata -------------------------------------------------------

func TestOutputXMP(t *testing.T) {
	doc := newMinimalDoc()
	doc.Meta.SetTitle("XMP Test")
	data, _ := New(doc).GetOutPDFString()
	if !strings.Contains(string(data), "/Metadata") {
		t.Fatal("PDF should contain /Metadata reference")
	}
}

// ---- fmtF helper --------------------------------------------------------

func TestFmtFIntegers(t *testing.T) {
	tests := map[float64]string{
		0: "0", 1: "1", -1: "-1", 100: "100", -100: "-100",
	}
	for in, want := range tests {
		got := fmtF(in)
		if got != want {
			t.Errorf("fmtF(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestFmtFDecimals(t *testing.T) {
	got := fmtF(595.28)
	if got == "" || got == "595" {
		t.Fatalf("fmtF(595.28) should preserve decimal: %q", got)
	}
}

func TestFmtFNoTrailingZeros(t *testing.T) {
	got := fmtF(1.5000)
	if strings.HasSuffix(got, "0") {
		t.Fatalf("fmtF should strip trailing zeros: %q", got)
	}
	if got != "1.5" {
		t.Fatalf("fmtF(1.5): got %q", got)
	}
}
