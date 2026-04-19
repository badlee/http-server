// Package output provides the PDF binary serializer.
// Uses two-pass streaming (io.Writer, zero full-document RAM).
// All formatting uses strings.Builder / pdfbuf — no fmt.Sprintf in hot paths.
// The Document type is protected by sync.RWMutex for concurrent access.
package output

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tecnickcom/go-tcpdf/color"
	"github.com/tecnickcom/go-tcpdf/encrypt"
	"github.com/tecnickcom/go-tcpdf/font"
	"github.com/tecnickcom/go-tcpdf/graph"
	imgpkg "github.com/tecnickcom/go-tcpdf/image"
	"github.com/tecnickcom/go-tcpdf/page"
	"github.com/tecnickcom/go-tcpdf/pdfbuf"
	"github.com/tecnickcom/go-tcpdf/javascript"
	"github.com/tecnickcom/go-tcpdf/metainfo"
)

// Document bundles all data the Output layer needs to produce a PDF.
// It is safe to read from multiple goroutines after initial population;
// the embedded RWMutex protects writes during generation.
type Document struct {
	mu sync.RWMutex // protects all fields below

	Meta       *metainfo.MetaInfo
	Pages      *page.Page
	Fonts      *font.Stack
	Images     *imgpkg.Import
	Encrypt    *encrypt.Encrypt
	Spots      *color.SpotRegistry
	Gradients  []graph.Gradient
	ExtGStates []graph.ExtGState
	JS         *javascript.JavaScript
	ICCProfile *color.ICCProfile
	// PageContents holds the raw PDF operator stream for each page (by index).
	PageContents []string
	PageThumbs   []int
	// Compress enables zlib (FlateDecode) compression of page/stream content.
	Compress bool
	// Linearized is a placeholder; not yet implemented.
	Linearized bool
}

// RLock / RUnlock expose the document mutex so callers can hold a read-lock
// while calling WriteTo, preventing concurrent document mutation.
func (d *Document) RLock()   { d.mu.RLock() }
func (d *Document) RUnlock() { d.mu.RUnlock() }

// Output serializes a Document to PDF.
type Output struct {
	doc    *Document
	cw     *countingWriter
	xref   []int64
	objNum int
	info   int
	root   int
	pages  int
}

// New creates an Output for the given Document.
func New(doc *Document) *Output { return &Output{doc: doc} }

// =========================================================================
// Public API
// =========================================================================

// WriteTo writes the complete PDF to w using the two-pass streaming strategy.
// Implements io.WriterTo. Thread-safe: holds a read-lock on the Document
// for the duration of both passes.
func (o *Output) WriteTo(w io.Writer) (int64, error) {
	o.doc.mu.RLock()
	defer o.doc.mu.RUnlock()

	// Pass 1 — dry-run to io.Discard, collect xref offsets.
	o.reset()
	o.cw = newCountingWriter(io.Discard)
	if err := o.serialize(); err != nil {
		return 0, errors.New("output: xref pass: " + err.Error())
	}
	xref1 := append([]int64(nil), o.xref...)
	info1, root1 := o.info, o.root

	// Pass 2 — real write using Pass-1 xref offsets.
	o.reset()
	real := newCountingWriter(w)
	o.cw = real
	o.xref = make([]int64, len(xref1))
	copy(o.xref, xref1)
	o.info = info1
	o.root = root1

	if err := o.serialize(); err != nil {
		_ = real.Flush()
		return real.Pos(), errors.New("output: write pass: " + err.Error())
	}
	if err := real.Flush(); err != nil {
		return real.Pos(), err
	}
	return real.Pos(), real.Err()
}

// RenderPDF writes the PDF to w (alias for WriteTo, discards byte count).
func (o *Output) RenderPDF(w io.Writer) error { _, err := o.WriteTo(w); return err }

// GetOutPDFString returns the complete PDF as a byte slice (convenience wrapper).
func (o *Output) GetOutPDFString() ([]byte, error) {
	var buf bytes.Buffer
	if _, err := o.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SavePDF streams the PDF to the file at path.
func (o *Output) SavePDF(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return errors.New("output: SavePDF create " + path + ": " + err.Error())
	}
	defer f.Close()
	if _, err = o.WriteTo(f); err != nil {
		return errors.New("output: SavePDF write: " + err.Error())
	}
	return f.Sync()
}

// GetMIMEAttachmentPDF returns the PDF MIME type.
func (o *Output) GetMIMEAttachmentPDF() string { return "application/pdf" }

// =========================================================================
// Internal reset / serialization
// =========================================================================

func (o *Output) reset() {
	o.xref = nil
	o.objNum = 0
	o.info = 0
	o.root = 0
	o.pages = 0
}

func (o *Output) serialize() error {
	o.writeHeader()
	if err := o.writeBody(); err != nil {
		return err
	}
	return o.writeXRef()
}

// ---- Header -------------------------------------------------------------

func (o *Output) writeHeader() {
	ver := o.doc.Meta.PDFVersion
	if ver == "" {
		ver = "1.7"
	}
	o.cw.WriteString("%PDF-")
	o.cw.WriteString(ver)
	o.cw.WriteString("\n%\xe2\xe3\xcf\xd3\n")
}

// ---- Body ---------------------------------------------------------------

func (o *Output) writeBody() error {
	pagesObjNum := o.reserveObj()
	o.pages = pagesObjNum

	if err := o.writeFonts(); err != nil {
		return err
	}
	if err := o.writeImages(); err != nil {
		return err
	}
	if err := o.writeSpotColors(); err != nil {
		return err
	}
	if err := o.writeGradients(); err != nil {
		return err
	}
	if err := o.writeExtGStates(); err != nil {
		return err
	}
	if err := o.writeXObjectTemplates(); err != nil {
		return err
	}
	if err := o.writeEmbeddedFiles(); err != nil {
		return err
	}

	iccObjNum := 0
	if o.doc.Meta.SRGB && o.doc.ICCProfile != nil {
		var err error
		if iccObjNum, err = o.writeICCProfile(o.doc.ICCProfile); err != nil {
			return err
		}
	}

	pageObjNums := make([]int, 0, o.doc.Pages.Count())
	for i := 0; i < o.doc.Pages.Count(); i++ {
		pg, _ := o.doc.Pages.Get(i)
		n, err := o.writePage(pg, i, pagesObjNum, iccObjNum)
		if err != nil {
			return err
		}
		pageObjNums = append(pageObjNums, n)
	}

	o.writePagesObj(pagesObjNum, pageObjNums)

	outlineObjNum := 0
	if len(o.doc.JS.Bookmarks()) > 0 {
		outlineObjNum = o.writeOutlines()
	}
	acroFormObjNum := 0
	if len(o.doc.JS.FormFields()) > 0 {
		acroFormObjNum = o.writeAcroForm()
	}
	jsObjNum := 0
	if o.doc.JS.RawJS() != "" {
		jsObjNum = o.writeJS()
	}
	if len(o.doc.JS.NamedDests()) > 0 {
		o.writeNamedDests()
	}

	xmpObjNum := o.writeXMPMetadata()
	o.info = o.writeInfoDict()
	o.root = o.writeCatalog(pagesObjNum, outlineObjNum, acroFormObjNum, jsObjNum, xmpObjNum)
	return nil
}

// =========================================================================
// Object primitives
// =========================================================================

func (o *Output) reserveObj() int {
	o.objNum++
	return o.objNum
}

func (o *Output) startObj(n int) {
	for len(o.xref) <= n {
		o.xref = append(o.xref, 0)
	}
	o.xref[n] = o.cw.Pos()
	var b pdfbuf.Buf
	b.ObjHeader(n)
	o.cw.WriteString(b.String())
}

func (o *Output) endObj() { o.cw.WriteString("endobj\n") }

func (o *Output) writeStream(n int, dict, streamData string, compress bool) error {
	data := []byte(streamData)
	filters := ""

	if compress && len(data) > 0 {
		var buf bytes.Buffer
		w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
		if err != nil {
			return err
		}
		_, _ = w.Write(data)
		_ = w.Close()
		data = buf.Bytes()
		filters = "/Filter /FlateDecode\n"
	}

	if !o.doc.Encrypt.Disabled() {
		o.doc.Encrypt.SetObjNum(n, 0)
		enc, err := o.doc.Encrypt.EncryptBytes(data)
		if err != nil {
			return err
		}
		data = enc
	}

	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<<\n")
	b.S(dict)
	b.S(filters)
	b.S("/Length ")
	b.I(len(data))
	b.S("\n>>\nstream\r\n")
	o.cw.WriteString(b.String())
	o.cw.Write(data)
	o.cw.WriteString("\r\nendstream\n")
	o.endObj()
	return nil
}

// =========================================================================
// Fonts
// =========================================================================

func (o *Output) writeFonts() error {
	for rk, m := range o.doc.Fonts.All() {
		var err error
		switch m.Type {
		case font.FontTypeCore:
			err = o.writeCoreFont(rk, m)
		case font.FontTypeTrueType, font.FontTypeOpenType:
			err = o.writeTTFont(rk, m)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Output) writeCoreFont(rk string, m *font.FontMetric) error {
	n := o.reserveObj()
	m.ObjNum = n
	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<<\n/Type /Font\n/Subtype /Type1\n/BaseFont /")
	b.S(m.Name)
	b.NL()
	if m.Encoding != "" {
		b.S("/Encoding /")
		b.S(m.Encoding)
		b.NL()
	} else {
		b.S("/Encoding /WinAnsiEncoding\n")
	}
	b.S(">>\n")
	o.cw.WriteString(b.String())
	o.endObj()
	return nil
}

func (o *Output) writeTTFont(_ string, m *font.FontMetric) error {
	descN := o.reserveObj()
	m.DescObjNum = descN

	fontFileN := 0
	if len(m.Data) > 0 {
		fontFileN = o.reserveObj()
		var buf bytes.Buffer
		w, _ := zlib.NewWriterLevel(&buf, zlib.BestCompression)
		_, _ = w.Write(m.Data)
		_ = w.Close()
		o.startObj(fontFileN)
		var b pdfbuf.Buf
		b.S("<< /Length ")
		b.I(buf.Len())
		b.S(" /Length1 ")
		b.I(len(m.Data))
		b.S(" /Filter /FlateDecode >>\nstream\r\n")
		o.cw.WriteString(b.String())
		o.cw.Write(buf.Bytes())
		o.cw.WriteString("\r\nendstream\n")
		o.endObj()
	}

	o.startObj(descN)
	var bd pdfbuf.Buf
	bd.S("<<\n/Type /FontDescriptor\n/FontName /")
	bd.S(m.Name)
	bd.S("\n/Flags ")
	bd.I(int(m.Flags))
	bd.S("\n/FontBBox [")
	bd.F(m.BBox[0])
	bd.SP()
	bd.F(m.BBox[1])
	bd.SP()
	bd.F(m.BBox[2])
	bd.SP()
	bd.F(m.BBox[3])
	bd.S("]\n")
	bd.DictKeyF("ItalicAngle", m.ItalicAngle)
	bd.DictKeyF("Ascent", m.Ascent)
	bd.DictKeyF("Descent", m.Descent)
	bd.DictKeyF("CapHeight", m.CapHeight)
	bd.DictKeyF("MissingWidth", m.MissingWidth)
	bd.DictKeyF("StemV", stemV(m.Flags))
	if fontFileN > 0 {
		bd.DictKeyRef("FontFile2", fontFileN)
	}
	bd.S(">>\n")
	o.cw.WriteString(bd.String())
	o.endObj()

	widthsN := o.reserveObj()
	o.startObj(widthsN)
	var bw pdfbuf.Buf
	bw.B('[')
	for i := 32; i <= 255; i++ {
		if i > 32 {
			bw.SP()
		}
		bw.F(m.GlyphWidth(rune(i)))
	}
	bw.S("]\n")
	o.cw.WriteString(bw.String())
	o.endObj()

	n := o.reserveObj()
	m.ObjNum = n
	o.startObj(n)
	var bf pdfbuf.Buf
	bf.S("<<\n/Type /Font\n/Subtype /TrueType\n/BaseFont /")
	bf.S(m.Name)
	bf.S("\n/Encoding /WinAnsiEncoding\n/FirstChar 32\n/LastChar 255\n")
	bf.DictKeyRef("Widths", widthsN)
	bf.DictKeyRef("FontDescriptor", descN)
	bf.S(">>\n")
	o.cw.WriteString(bf.String())
	o.endObj()
	return nil
}

// =========================================================================
// Images
// =========================================================================

func (o *Output) writeImages() error {
	for _, img := range o.doc.Images.All() {
		if img.HasAlpha {
			maskN := o.reserveObj()
			img.MaskObjNum = maskN
			var b pdfbuf.Buf
			b.S("/Type /XObject\n/Subtype /Image\n/Width ")
			b.I(img.Width)
			b.S("\n/Height ")
			b.I(img.Height)
			b.S("\n/ColorSpace /DeviceGray\n/BitsPerComponent 8\n")
			if err := o.writeStream(maskN, b.String(), "", o.doc.Compress); err != nil {
				return err
			}
		}

		n := o.reserveObj()
		img.ObjNum = n
		o.startObj(n)
		var b pdfbuf.Buf
		b.S("<<\n/Type /XObject\n/Subtype /Image\n/Width ")
		b.I(img.Width)
		b.S("\n/Height ")
		b.I(img.Height)
		b.NL()
		if len(img.Palette) > 0 {
			b.S("/ColorSpace [/Indexed /DeviceRGB ")
			b.I(len(img.Palette)/3 - 1)
			b.S(" <")
			b.Hex(img.Palette)
			b.S(">]\n")
		} else {
			b.S("/ColorSpace /")
			b.S(img.ColorSpace)
			b.NL()
		}
		b.S("/BitsPerComponent ")
		b.I(img.BitsPerComp)
		b.NL()
		if img.Filter != "" {
			b.S("/Filter /")
			b.S(img.Filter)
			b.NL()
		}
		if img.DecodeParms != "" {
			b.S("/DecodeParms ")
			b.S(img.DecodeParms)
			b.NL()
		}
		if img.MaskObjNum > 0 {
			b.DictKeyRef("SMask", img.MaskObjNum)
		}
		if len(img.ColorKeyMask) > 0 {
			b.S("/Mask [")
			for i, v := range img.ColorKeyMask {
				if i > 0 {
					b.SP()
				}
				b.I(v)
			}
			b.S("]\n")
		}
		b.S("/Length ")
		b.I(len(img.Data))
		b.S("\n>>\nstream\r\n")
		o.cw.WriteString(b.String())
		o.cw.Write(img.Data)
		o.cw.WriteString("\r\nendstream\n")
		o.endObj()
	}
	return nil
}

// =========================================================================
// Spot colors
// =========================================================================

func (o *Output) writeSpotColors() error {
	if o.doc.Spots == nil {
		return nil
	}
	for _, sp := range o.doc.Spots.All() {
		n := o.reserveObj()
		o.startObj(n)
		var b pdfbuf.Buf
		b.S("[/Separation /")
		b.S(sp.Name)
		b.S(" /")
		b.S(sp.AltSpace)
		b.S(" {")
		for range sp.AltValues {
			b.S(" 1")
		}
		b.S(" pop")
		for _, v := range sp.AltValues {
			b.SP()
			b.F(v)
		}
		b.S("}]\n")
		o.cw.WriteString(b.String())
		o.endObj()
	}
	return nil
}

// =========================================================================
// Gradients
// =========================================================================

func (o *Output) writeGradients() error {
	for i := range o.doc.Gradients {
		gr := &o.doc.Gradients[i]
		funcN := o.reserveObj()
		o.startObj(funcN)
		if len(gr.Stops) == 2 {
			o.cw.WriteString(o.twoStopFunction(gr.Stops[0].Color.ToRGB(), gr.Stops[1].Color.ToRGB()))
		} else {
			o.cw.WriteString(o.stitchingFunction(gr.Stops))
		}
		o.endObj()

		shadingN := o.reserveObj()
		gr.ObjNum = shadingN
		o.startObj(shadingN)
		var b pdfbuf.Buf
		b.S("<<\n")
		if gr.Type == graph.GradientLinear {
			b.S("/ShadingType 2\n/Coords [")
			b.F(gr.X1)
			b.SP()
			b.F(gr.Y1)
			b.SP()
			b.F(gr.X2)
			b.SP()
			b.F(gr.Y2)
			b.S("]\n")
		} else {
			b.S("/ShadingType 3\n/Coords [")
			b.F(gr.X1)
			b.SP()
			b.F(gr.Y1)
			b.SP()
			b.F(gr.R1)
			b.SP()
			b.F(gr.X2)
			b.SP()
			b.F(gr.Y2)
			b.SP()
			b.F(gr.R2)
			b.S("]\n")
		}
		b.S("/ColorSpace /DeviceRGB\n/Extend [true true]\n")
		b.DictKeyRef("Function", funcN)
		b.S(">>\n")
		o.cw.WriteString(b.String())
		o.endObj()
	}
	return nil
}

func (o *Output) twoStopFunction(c0, c1 color.Color) string {
	var b pdfbuf.Buf
	b.S("<< /FunctionType 2 /Domain [0 1] /C0 [")
	b.F(c0.R)
	b.SP()
	b.F(c0.G)
	b.SP()
	b.F(c0.B)
	b.S("] /C1 [")
	b.F(c1.R)
	b.SP()
	b.F(c1.G)
	b.SP()
	b.F(c1.B)
	b.S("] /N 1 >>\n")
	return b.String()
}

func (o *Output) stitchingFunction(stops []graph.GradientStop) string {
	n := len(stops)
	var funcs, bounds, encode []string
	for i := 0; i < n-1; i++ {
		fN := o.reserveObj()
		for len(o.xref) <= fN {
			o.xref = append(o.xref, 0)
		}
		o.xref[fN] = o.cw.Pos()
		var bh pdfbuf.Buf
		bh.ObjHeader(fN)
		bh.S(o.twoStopFunction(stops[i].Color.ToRGB(), stops[i+1].Color.ToRGB()))
		bh.ObjFooter()
		o.cw.WriteString(bh.String())

		var ref pdfbuf.Buf
		ref.I(fN)
		ref.S(" 0 R")
		funcs = append(funcs, ref.String())
		encode = append(encode, "0 1")
		if i < n-2 {
			bounds = append(bounds, pdfbuf.FmtF(stops[i+1].Offset))
		}
	}
	var b pdfbuf.Buf
	b.S("<< /FunctionType 3 /Domain [")
	b.F(stops[0].Offset)
	b.SP()
	b.F(stops[n-1].Offset)
	b.S("] /Functions [")
	b.S(strings.Join(funcs, " "))
	b.S("] /Bounds [")
	b.S(strings.Join(bounds, " "))
	b.S("] /Encode [")
	b.S(strings.Join(encode, " "))
	b.S("] >>\n")
	return b.String()
}

// =========================================================================
// ExtGStates
// =========================================================================

func (o *Output) writeExtGStates() error {
	for i := range o.doc.ExtGStates {
		eg := &o.doc.ExtGStates[i]
		n := o.reserveObj()
		eg.ObjNum = n
		o.startObj(n)
		var b pdfbuf.Buf
		b.S("<<\n/Type /ExtGState\n")
		if eg.CA >= 0 {
			b.DictKeyF("CA", eg.CA)
		}
		if eg.Ca >= 0 {
			b.DictKeyF("ca", eg.Ca)
		}
		if eg.BM != "" {
			b.S("/BM /")
			b.S(eg.BM)
			b.NL()
		}
		b.S(">>\n")
		o.cw.WriteString(b.String())
		o.endObj()
	}
	return nil
}

// =========================================================================
// XObject templates
// =========================================================================

func (o *Output) writeXObjectTemplates() error {
	for _, xobj := range o.doc.JS.XObjects() {
		n := o.reserveObj()
		xobj.ObjNum = n
		var b pdfbuf.Buf
		b.S("/Type /XObject\n/Subtype /Form\n/FormType 1\n/BBox [0 0 ")
		b.F(xobj.Width)
		b.SP()
		b.F(xobj.Height)
		b.S("]\n/Resources << >>\n")
		if err := o.writeStream(n, b.String(), xobj.Content, o.doc.Compress); err != nil {
			return err
		}
	}
	return nil
}

// =========================================================================
// Embedded files
// =========================================================================

func (o *Output) writeEmbeddedFiles() error {
	for _, ef := range o.doc.JS.EmbeddedFiles() {
		n := o.reserveObj()
		ef.ObjNum = n
		mime := ef.MIME
		if mime == "" {
			mime = "application/octet-stream"
		}
		var b pdfbuf.Buf
		b.S("/Type /EmbeddedFile\n/Subtype (")
		b.S(mime)
		b.S(")\n")
		if err := o.writeStream(n, b.String(), string(ef.Content), o.doc.Compress); err != nil {
			return err
		}
	}
	return nil
}

// =========================================================================
// ICC Profile
// =========================================================================

func (o *Output) writeICCProfile(icc *color.ICCProfile) (int, error) {
	n := o.reserveObj()
	var b pdfbuf.Buf
	b.S("/N ")
	b.I(icc.Components)
	b.NL()
	if err := o.writeStream(n, b.String(), string(icc.Data), true); err != nil {
		return 0, err
	}
	return n, nil
}

// =========================================================================
// Pages
// =========================================================================

func (o *Output) writePage(pg *page.PageData, idx, pagesParent, iccObjNum int) (int, error) {
	resourcesN := o.reserveObj()
	o.writePageResources(resourcesN, pg, iccObjNum)

	annotRefs := o.writePageAnnotations(pg, idx)

	contentN := o.reserveObj()
	content := ""
	if idx < len(o.doc.PageContents) {
		content = o.doc.PageContents[idx]
	}
	if err := o.writeStream(contentN, "", content, o.doc.Compress); err != nil {
		return 0, err
	}

	n := o.reserveObj()
	pg.ID = n
	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<<\n/Type /Page\n")
	b.DictKeyRef("Parent", pagesParent)
	mb := pg.Boxes.Media
	b.S("/MediaBox ")
	b.Rect(mb.Llx, mb.Lly, mb.Urx, mb.Ury)
	b.NL()
	if pg.Boxes.Crop != mb {
		cb := pg.Boxes.Crop
		b.S("/CropBox ")
		b.Rect(cb.Llx, cb.Lly, cb.Urx, cb.Ury)
		b.NL()
	}
	if pg.Boxes.Bleed != mb {
		bb := pg.Boxes.Bleed
		b.S("/BleedBox ")
		b.Rect(bb.Llx, bb.Lly, bb.Urx, bb.Ury)
		b.NL()
	}
	if pg.Boxes.Trim != mb {
		tb := pg.Boxes.Trim
		b.S("/TrimBox ")
		b.Rect(tb.Llx, tb.Lly, tb.Urx, tb.Ury)
		b.NL()
	}
	if pg.Boxes.Art != mb {
		ab := pg.Boxes.Art
		b.S("/ArtBox ")
		b.Rect(ab.Llx, ab.Lly, ab.Urx, ab.Ury)
		b.NL()
	}
	if pg.Rotation != 0 {
		b.DictKeyI("Rotate", pg.Rotation)
	}
	b.DictKeyRef("Resources", resourcesN)
	b.DictKeyRef("Contents", contentN)
	if len(annotRefs) > 0 {
		b.S("/Annots [")
		for i, ref := range annotRefs {
			if i > 0 {
				b.SP()
			}
			b.I(ref)
			b.S(" 0 R")
		}
		b.S("]\n")
	}
	b.S(">>\n")
	o.cw.WriteString(b.String())
	o.endObj()
	return n, nil
}

func (o *Output) writePageResources(n int, _ *page.PageData, iccObjNum int) {
	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<<\n/Font <<\n")
	for rk, m := range o.doc.Fonts.All() {
		if m.ObjNum > 0 {
			b.B('/')
			b.S(rk)
			b.SP()
			b.I(m.ObjNum)
			b.S(" 0 R\n")
		}
	}
	b.S(">>\n/XObject <<\n")
	for _, img := range o.doc.Images.All() {
		if img.ObjNum > 0 {
			b.B('/')
			b.S(img.Key)
			b.SP()
			b.I(img.ObjNum)
			b.S(" 0 R\n")
		}
	}
	for _, xobj := range o.doc.JS.XObjects() {
		if xobj.ObjNum > 0 {
			b.S("/XObjSVG")
			b.I(xobj.ObjNum)
			b.SP()
			b.I(xobj.ObjNum)
			b.S(" 0 R\n")
		}
	}
	b.S(">>\n")
	if o.doc.Spots != nil && len(o.doc.Spots.All()) > 0 {
		b.S("/ColorSpace <<\n")
		for _, sp := range o.doc.Spots.All() {
			b.S("/CS")
			b.S(strings.ReplaceAll(sp.Name, " ", "_"))
			b.S(" [/Separation /")
			b.S(sp.Name)
			b.S(" /DeviceRGB {}]\n")
		}
		if iccObjNum > 0 {
			b.S("/DefaultRGB [/ICCBased ")
			b.I(iccObjNum)
			b.S(" 0 R]\n")
		}
		b.S(">>\n")
	}
	if len(o.doc.Gradients) > 0 {
		b.S("/Shading <<\n")
		for i, gr := range o.doc.Gradients {
			if gr.ObjNum > 0 {
				b.S("/Sh")
				b.I(i)
				b.SP()
				b.I(gr.ObjNum)
				b.S(" 0 R\n")
			}
		}
		b.S(">>\n")
	}
	if len(o.doc.ExtGStates) > 0 {
		b.S("/ExtGState <<\n")
		for i, eg := range o.doc.ExtGStates {
			if eg.ObjNum > 0 {
				b.S("/GS")
				b.I(i)
				b.SP()
				b.I(eg.ObjNum)
				b.S(" 0 R\n")
			}
		}
		b.S(">>\n")
	}
	b.S("/ProcSet [/PDF /Text /ImageB /ImageC /ImageI]\n>>\n")
	o.cw.WriteString(b.String())
	o.endObj()
}

func (o *Output) writePageAnnotations(pg *page.PageData, pageIdx int) []int {
	var refs []int
	pageH := pg.Boxes.Media.Ury

	for _, annot := range o.doc.JS.Annotations() {
		if annot.Page != pageIdx {
			continue
		}
		n := o.reserveObj()
		o.startObj(n)
		x := annot.PosX
		y := pageH - annot.PosY - annot.Height
		var b pdfbuf.Buf
		b.F(x)
		b.SP()
		b.F(y)
		b.SP()
		b.F(x + annot.Width)
		b.SP()
		b.F(y + annot.Height)
		o.cw.WriteString(javascript.AnnotDict(annot, b.String()))
		o.cw.WriteString("\n")
		o.endObj()
		refs = append(refs, n)
	}

	for _, lnk := range o.doc.JS.Links() {
		if lnk.Page != pageIdx {
			continue
		}
		n := o.reserveObj()
		o.startObj(n)
		x := lnk.PosX
		y := pageH - lnk.PosY - lnk.Height
		var b pdfbuf.Buf
		b.S("<< /Type /Annot /Subtype /Link\n/Rect [")
		b.F(x)
		b.SP()
		b.F(y)
		b.SP()
		b.F(x + lnk.Width)
		b.SP()
		b.F(y + lnk.Height)
		b.S("]\n/Border [0 0 0]\n")
		if strings.HasPrefix(lnk.URL, "#") {
			b.S("/Dest [0 0 R /XYZ 0 0 null]\n")
		} else {
			b.S("/A << /Type /Action /S /URI /URI (")
			b.S(lnk.URL)
			b.S(") >>\n")
		}
		b.S(">>\n")
		o.cw.WriteString(b.String())
		o.endObj()
		refs = append(refs, n)
	}
	return refs
}

func (o *Output) writePagesObj(pagesN int, pageObjNums []int) {
	for len(o.xref) <= pagesN {
		o.xref = append(o.xref, 0)
	}
	o.xref[pagesN] = o.cw.Pos()
	var b pdfbuf.Buf
	b.ObjHeader(pagesN)
	b.S("<<\n/Type /Pages\n/Count ")
	b.I(len(pageObjNums))
	b.S("\n/Kids [")
	for i, pn := range pageObjNums {
		if i > 0 {
			b.SP()
		}
		b.I(pn)
		b.S(" 0 R")
	}
	b.S("]\n>>\n")
	b.ObjFooter()
	o.cw.WriteString(b.String())
}

// =========================================================================
// Outlines
// =========================================================================

func (o *Output) writeOutlines() int {
	bms := o.doc.JS.Bookmarks()
	if len(bms) == 0 {
		return 0
	}
	outlineN := o.reserveObj()
	itemNums := make([]int, len(bms))
	for i := range bms {
		itemNums[i] = o.reserveObj()
	}
	for i, bm := range bms {
		o.startObj(itemNums[i])
		var b pdfbuf.Buf
		b.S("<<\n/Title ")
		b.PDFStr(bm.Name)
		b.NL()
		b.DictKeyRef("Parent", outlineN)
		if i > 0 {
			b.DictKeyRef("Prev", itemNums[i-1])
		}
		if i < len(bms)-1 {
			b.DictKeyRef("Next", itemNums[i+1])
		}
		b.S("/Dest [0 0 R /XYZ ")
		b.F(bm.PosX)
		b.SP()
		b.F(bm.PosY)
		b.S(" null]\n>>\n")
		o.cw.WriteString(b.String())
		o.endObj()
	}
	o.startObj(outlineN)
	var b pdfbuf.Buf
	b.S("<<\n/Type /Outlines\n")
	b.DictKeyRef("First", itemNums[0])
	b.DictKeyRef("Last", itemNums[len(itemNums)-1])
	b.DictKeyI("Count", len(bms))
	b.S(">>\n")
	o.cw.WriteString(b.String())
	o.endObj()
	return outlineN
}

// =========================================================================
// AcroForm
// =========================================================================

func (o *Output) writeAcroForm() int {
	fields := o.doc.JS.FormFields()
	if len(fields) == 0 {
		return 0
	}
	fieldNums := make([]int, len(fields))
	for i, ff := range fields {
		n := o.reserveObj()
		ff.ObjNum = n
		fieldNums[i] = n
		o.startObj(n)
		var b pdfbuf.Buf
		b.S("<<\n/Type /Annot\n/Subtype /Widget\n/FT /")
		b.S(ff.Type)
		b.S("\n/T ")
		b.PDFStr(ff.Name)
		b.S("\n>>\n")
		o.cw.WriteString(b.String())
		o.endObj()
	}
	n := o.reserveObj()
	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<<\n/Fields [")
	for i, fn := range fieldNums {
		if i > 0 {
			b.SP()
		}
		b.I(fn)
		b.S(" 0 R")
	}
	b.S("]\n/DR << >>\n>>\n")
	o.cw.WriteString(b.String())
	o.endObj()
	return n
}

// =========================================================================
// JavaScript
// =========================================================================

func (o *Output) writeJS() int {
	script := o.doc.JS.RawJS()
	if script == "" {
		return 0
	}
	streamN := o.reserveObj()
	_ = o.writeStream(streamN, "/S /JavaScript\n", script, false)
	n := o.reserveObj()
	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<< /Names [(EmbeddedJS) ")
	b.I(streamN)
	b.S(" 0 R] >>\n")
	o.cw.WriteString(b.String())
	o.endObj()
	return n
}

// =========================================================================
// Named destinations
// =========================================================================

func (o *Output) writeNamedDests() int {
	dests := o.doc.JS.NamedDests()
	if len(dests) == 0 {
		return 0
	}
	n := o.reserveObj()
	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<< /Names [")
	for _, nd := range dests {
		b.PDFStr(nd.Name)
		b.S(" [0 0 R /XYZ ")
		b.F(nd.PosX)
		b.SP()
		b.F(nd.PosY)
		b.S(" null] ")
	}
	b.S("] >>\n")
	o.cw.WriteString(b.String())
	o.endObj()
	return n
}

// =========================================================================
// XMP metadata
// =========================================================================

func (o *Output) writeXMPMetadata() int {
	n := o.reserveObj()
	_ = o.writeStream(n, "/Type /Metadata\n/Subtype /XML\n", o.doc.Meta.XMPData(), false)
	return n
}

// =========================================================================
// Info dictionary
// =========================================================================

func (o *Output) writeInfoDict() int {
	n := o.reserveObj()
	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<<\n")
	b.S(o.doc.Meta.InfoDictEntries(pdfbuf.EscapePDFString))
	b.S(">>\n")
	o.cw.WriteString(b.String())
	o.endObj()
	return n
}

// =========================================================================
// Catalog
// =========================================================================

func (o *Output) writeCatalog(pagesN, outlineN, acroFormN, jsN, xmpN int) int {
	n := o.reserveObj()
	o.startObj(n)
	var b pdfbuf.Buf
	b.S("<<\n/Type /Catalog\n")
	b.DictKeyRef("Pages", pagesN)
	if xmpN > 0 {
		b.DictKeyRef("Metadata", xmpN)
	}
	if outlineN > 0 {
		b.DictKeyRef("Outlines", outlineN)
		b.S("/PageMode /UseOutlines\n")
	}
	dm := o.doc.Meta.DisplayMode
	if dm.Layout != "" && dm.Layout != "SinglePage" {
		b.S("/PageLayout /")
		b.S(dm.Layout)
		b.NL()
	}
	if dm.Mode != "" && dm.Mode != "UseNone" {
		b.S("/PageMode /")
		b.S(dm.Mode)
		b.NL()
	}
	if vp := o.doc.Meta.ViewerPrefDict(); vp != "" {
		b.S("/ViewerPreferences <<\n")
		b.S(vp)
		b.S(">>\n")
	}
	if acroFormN > 0 {
		b.DictKeyRef("AcroForm", acroFormN)
	}
	if jsN > 0 {
		b.S("/Names << /JavaScript ")
		b.I(jsN)
		b.S(" 0 R >>\n")
	}
	if o.doc.Meta.Language != "" {
		b.S("/Lang ")
		b.PDFStr(o.doc.Meta.Language)
		b.NL()
	}
	if efFiles := o.doc.JS.EmbeddedFiles(); len(efFiles) > 0 {
		b.S("/Names << /EmbeddedFiles << /Names [")
		for _, ef := range efFiles {
			if ef.ObjNum > 0 {
				b.PDFStr(ef.Name)
				b.SP()
				b.I(ef.ObjNum)
				b.S(" 0 R ")
			}
		}
		b.S("] >> >>\n")
	}
	b.S(">>\n")
	o.cw.WriteString(b.String())
	o.endObj()
	return n
}

// =========================================================================
// Cross-reference table & trailer
// =========================================================================

func (o *Output) writeXRef() error {
	startXRef := o.cw.Pos()
	count := len(o.xref)

	var b pdfbuf.Buf
	b.S("xref\n0 ")
	b.I(count)
	b.NL()
	b.XRefEntry(0, false) // object 0: free list head (generation 65535)
	for i := 1; i < count; i++ {
		if i < len(o.xref) && o.xref[i] > 0 {
			b.XRefEntry(o.xref[i], true)
		} else {
			b.XRefEntry(0, false)
		}
	}
	b.S("trailer\n<<\n/Size ")
	b.I(count)
	b.NL()
	b.DictKeyRef("Root", o.root)
	if o.info > 0 {
		b.DictKeyRef("Info", o.info)
	}
	if !o.doc.Encrypt.Disabled() {
		fid := pdfbuf.HexStr(o.doc.Encrypt.FileID())
		b.S("/ID [<")
		b.S(fid)
		b.S("> <")
		b.S(fid)
		b.S(">]\n")
		encN := o.writeEncryptDictInto(&b)
		b.DictKeyRef("Encrypt", encN)
	} else {
		ts := strconv.FormatInt(time.Now().UnixNano(), 16)
		b.S("/ID [<")
		b.S(ts)
		b.S("> <")
		b.S(ts)
		b.S(">]\n")
	}
	b.S(">>\nstartxref\n")
	b.I64(startXRef)
	b.S("\n%%EOF\n")
	o.cw.WriteString(b.String())
	return nil
}

// writeEncryptDictInto writes the encryption dictionary as a PDF object
// and returns its object number. It appends the object bytes to b.
func (o *Output) writeEncryptDictInto(b *pdfbuf.Buf) int {
	enc := o.doc.Encrypt
	n := o.reserveObj()
	for len(o.xref) <= n {
		o.xref = append(o.xref, 0)
	}
	o.xref[n] = o.cw.Pos() + int64(b.Len())
	var eb pdfbuf.Buf
	eb.ObjHeader(n)
	eb.S("<<\n/Filter /Standard\n")
	eb.DictKeyI("V", enc.Version())
	eb.DictKeyI("R", enc.Revision())
	eb.DictKeyI("Length", enc.KeyLength())
	eb.DictKeyI("P", -1)
	eb.S("/O <")
	eb.Hex(enc.OwnerKey())
	eb.S(">\n/U <")
	eb.Hex(enc.UserKey())
	eb.S(">\n")
	if enc.UE() != nil {
		eb.S("/UE <")
		eb.Hex(enc.UE())
		eb.S(">\n")
	}
	if enc.OE() != nil {
		eb.S("/OE <")
		eb.Hex(enc.OE())
		eb.S(">\n")
	}
	if enc.Perms() != nil {
		eb.S("/Perms <")
		eb.Hex(enc.Perms())
		eb.S(">\n")
	}
	eb.S(">>\n")
	eb.ObjFooter()
	b.S(eb.String())
	return n
}

// =========================================================================
// Emit helper
// =========================================================================

func (o *Output) emit(s string) { o.cw.WriteString(s) }

// =========================================================================
// Numeric helpers (no fmt)
// =========================================================================

func fmtF(v float64) string { return pdfbuf.FmtF(v) }

func stemV(flags uint32) float64 {
	if flags&(1<<6) != 0 {
		return 120
	}
	return 70
}
