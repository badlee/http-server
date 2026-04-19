# go-tcpdf — API Reference & Usage Guide

## Table of Contents

1. [Creating a document](#1-creating-a-document)
2. [Metadata](#2-metadata)
3. [Page management](#3-page-management)
4. [Fonts](#4-fonts)
5. [Colors](#5-colors)
6. [Text output](#6-text-output)
7. [Drawing](#7-drawing)
8. [Images](#8-images)
9. [SVG](#9-svg)
10. [HTML](#10-html)
11. [Links and annotations](#11-links-and-annotations)
12. [Bookmarks and TOC](#12-bookmarks-and-toc)
13. [Form fields](#13-form-fields)
14. [JavaScript](#14-javascript)
15. [XObject templates](#15-xobject-templates)
16. [Layers](#16-layers)
17. [Transformations](#17-transformations)
18. [Gradients and transparency](#18-gradients-and-transparency)
19. [Encryption](#19-encryption)
20. [**Output and streaming**](#20-output-and-streaming)
21. [Advanced configuration](#21-advanced-configuration)
22. [Package-level APIs](#22-package-level-apis)
23. [Streaming internals](#23-streaming-internals)

---

## 1. Creating a document

```go
import (
    gotcpdf "github.com/tecnickcom/go-tcpdf"
    "github.com/tecnickcom/go-tcpdf/classobjects"
    "github.com/tecnickcom/go-tcpdf/internal/page"
)

// Default: A4, mm, portrait, 10mm margins
pdf, err := gotcpdf.NewDefault()

// Custom config
pdf, err := gotcpdf.New(classobjects.Config{
    Unit:        "mm",
    Format:      "A4",
    Orientation: page.Portrait,
    Margins:     page.Margins{Top: 15, Right: 15, Bottom: 20, Left: 15},
    SubsetFonts: true,
    Compress:    true,
})
```

Available page formats: `A0`–`A10`, `B0`–`B10`, `C0`–`C10`, `LETTER`, `LEGAL`,
`LEDGER`, `TABLOID`, `EXECUTIVE`, `FOLIO`, `CREDIT` — see `internal/page/page.go`.

---

## 2. Metadata

All setters return `*TCPDF` for chaining.

```go
pdf.
    SetTitle("Annual Report 2024").
    SetAuthor("Acme Corp").
    SetSubject("Financial results").
    SetKeywords("finance report annual").
    SetCreator("MyApp v1.0").
    SetLanguage("en-US").
    SetSRGB(true)
```

### PDF version

```go
err := pdf.SetPDFVersion("1.7")   // "1.0"–"1.7", "2.0"
```

### Viewer preferences

```go
import "github.com/tecnickcom/go-tcpdf/metainfo"

pdf.SetViewerPreferences(metainfo.ViewerPreferences{
    HideToolbar:     true,
    HideMenubar:     false,
    FitWindow:       true,
    CenterWindow:    true,
    DisplayDocTitle: true,
    Duplex:          "DuplexFlipLongEdge",
    NumCopies:       2,
})
```

### Custom XMP metadata

```go
pdf.SetCustomXMP("custom", `<rdf:Description rdf:about=""
    xmlns:myns="http://example.com/ns/">
  <myns:Department>Engineering</myns:Department>
</rdf:Description>`)
```

---

## 3. Page management

```go
pdf.AddPage()                      // default format/orientation
pdf.AddPage("LETTER")              // named format
pdf.AddPage(page.Landscape)        // orientation
pdf.AddPage("A3", page.Landscape)  // format + orientation

pdf.SetPage(2)          // switch active page (1-indexed)
n := pdf.PageCount()    // total count
```

### Margins and auto-break

```go
pdf.SetMargins(15, 20, 15)        // left, top, right
pdf.SetAutoPageBreak(true, 20)    // trigger when 20mm from bottom
```

---

## 4. Fonts

```go
pdf.SetFont("Helvetica", "",   12)   // regular
pdf.SetFont("Helvetica", "B",  14)   // bold
pdf.SetFont("Helvetica", "I",  12)   // italic
pdf.SetFont("Helvetica", "BI", 12)   // bold-italic
pdf.SetFont("Times",     "",   11)
pdf.SetFont("Courier",   "B",  10)
pdf.SetFont("Symbol",    "",   12)
pdf.SetFont("ZapfDingbats", "", 12)

family := pdf.GetFontFamily()    // "helvetica"
size   := pdf.GetFontSize()      // 12.0
w      := pdf.GetStringWidth("Hello World")  // user units
```

### TeX hyphenation

```go
content, _ := os.ReadFile("hyph-en-us.tex")
pdf.LoadTexHyphenPatterns(string(content))
```

---

## 5. Colors

```go
import "github.com/tecnickcom/go-tcpdf/internal/color"

pdf.SetFillColor(color.NewRGB(0.2, 0.4, 0.8))
pdf.SetFillColor(color.NewRGB255(255, 128, 0))
pdf.SetDrawColor(color.Black)
pdf.SetTextColor(color.White)

pdf.SetFillColor(color.NewCMYK(0, 0.5, 1, 0))
pdf.SetFillColor(color.NewGray(0.5))

// CSS parsing
c, _ := color.ParseCSS("#ff6600")
c, _ = color.ParseCSS("rgba(255, 0, 0, 0.8)")
c, _ = color.ParseCSS("cmyk(0%, 60%, 100%, 0%)")

// Spot color
spot := color.NewSpot("PANTONE 485 C", 1.0)

// Alpha
c = color.NewRGB(1, 0, 0).WithAlpha(0.5)
```

---

## 6. Text output

### Cell

```go
// Cell(width, height, text, border, ln, align, fill, link)
//   border: "" | "0"=none | "1"=all | "L","T","R","B"=sides
//   ln:     0=cursor right | 1=new line | 2=below
//   align:  "L" | "C" | "R" | "J"
pdf.Cell(0, 10, "Full width", "1", 1, "C", false, "")
pdf.Cell(60, 8, "Left",       "1", 0, "L", true,  "")
pdf.Cell(60, 8, "Right",      "1", 1, "R", true,  "")
```

### MultiCell

```go
pdf.MultiCell(0, 6,
    "This long paragraph wraps automatically within the cell width.",
    "1", "J", false)
```

### Write (inline flowing text)

```go
pdf.Write(6, "Visit ",  "")
pdf.Write(6, "our site", "https://example.com")
pdf.Write(6, " today.", "")
```

### Line break and cursor

```go
pdf.Ln(5)          // explicit height
pdf.Ln(0)          // use current font line height

pdf.SetX(30)
pdf.SetY(50)
pdf.SetXY(30, 50)

x := pdf.GetX()
y := pdf.GetY()
bbox := pdf.GetLastBBox()   // last rendered text bounding box
```

---

## 7. Drawing

### Line properties

```go
pdf.SetLineWidth(0.5)
pdf.SetLineCap(0)   // 0=butt | 1=round | 2=square
pdf.SetLineJoin(0)  // 0=miter | 1=round | 2=bevel
pdf.SetLineDash([]float64{3, 1.5}, 0)  // dash array, phase
pdf.SetLineDash(nil, 0)                // solid
```

### Paint styles

| String | Effect |
|--------|--------|
| `"S"` | Stroke only |
| `"F"` | Fill (non-zero winding) |
| `"F*"` | Fill (even-odd) |
| `"FD"` / `"DF"` | Fill + stroke |
| `""` / `"n"` | No paint |

### Shapes

```go
pdf.Line(x1, y1, x2, y2)
pdf.Rect(x, y, w, h, "FD")
pdf.Circle(cx, cy, r, "F")
pdf.Ellipse(cx, cy, rx, ry, angleDeg, "S")
```

---

## 8. Images

```go
// Image(path, x, y, width, height, type, link)
// width=0, height=0 → natural size; one=0 → auto-scale
pdf.Image("photo.jpg",  10, 20, 80,  0, "jpg", "")
pdf.Image("logo.png",   10, 10,  0, 30, "png", "https://example.com")
```

---

## 9. SVG

```go
svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <circle cx="50" cy="50" r="40" fill="red" stroke="black" stroke-width="2"/>
  <path d="M10 80 C 40 10,65 10,95 80" stroke="blue" fill="none"/>
</svg>`

pdf.ImageSVG(svg, 20, 30, 80, 80)
```

Supported elements: `rect`, `circle`, `ellipse`, `line`, `polyline`, `polygon`,
`path` (M/L/H/V/C/S/Q/Z), `g`.
Supported transforms: `matrix()`, `translate()`, `scale()`, `rotate()`.

---

## 10. HTML

```go
html := `<h1>Title</h1>
<p>Normal <b>bold</b> <i>italic</i> <u>underline</u> text.</p>
<p style="color: red; font-size: 14pt">Styled.</p>
<ul><li>Item A</li><li>Item B</li></ul>`

pdf.WriteHTML(html, true, false)
```

---

## 11. Links and annotations

```go
// External URL over an area
pdf.AddLink("https://example.com", x, y, w, h)

// Link on a Cell
pdf.Cell(60, 8, "Click here", "", 1, "L", false, "https://example.com")

// Internal link
dest := pdf.JS.AddInternalLink(pageNum, posY)
pdf.Cell(60, 8, "Go to section", "", 1, "L", false, dest)

// Named destination
pdf.JS.SetNamedDestination("chapter2", pageNum, x, y)
```

### Annotations

```go
import "github.com/tecnickcom/go-tcpdf/javascript"

opts := javascript.AnnotOpts{Subtype: "Text", Contents: "Review this"}
pdf.JS.SetAnnotation(pageIndex, x, y, w, h, "Note text", opts)
```

---

## 12. Bookmarks and TOC

```go
pdf.SetBookmark("Chapter 1 — Overview", 0)  // level 0 = top
pdf.SetBookmark("Section 1.1", 1)            // level 1 = sub-entry

// Render TOC
pdf.SetPage(1)
for _, entry := range pdf.TOC.Entries() {
    pdf.SetX(15 + float64(entry.Level)*6)
    pdf.Cell(0, 6, entry.Title, "", 1, "L", false, "")
}
```

---

## 13. Form fields

```go
pdf.JS.AddJSText("name_field",  pageIdx, 20, 50, 80, 8, map[string]string{"value": "Default"})
pdf.JS.AddJSCheckBox("agree",   pageIdx, 20, 65, 5, map[string]string{})
pdf.JS.AddJSComboBox("color",   pageIdx, 20, 75, 60, 8, []interface{}{"Red","Green","Blue"}, map[string]string{})
pdf.JS.AddJSButton("submit_btn",pageIdx, 20, 90, 40, 8, "Submit",
    "this.submitForm('https://example.com/submit');", map[string]string{})
```

---

## 14. JavaScript

```go
pdf.AppendJavascript(`app.alert("Document opened");`)
pdf.JS.AddRawJavaScriptObj(`this.getField("name_field").value = "JS set";`, false)
```

---

## 15. XObject templates

```go
tid := pdf.JS.NewXObjectTemplate(210, 15, "")
pdf.JS.AddXObjectContent(tid, "% header PDF operators\n")
pdf.JS.ExitXObjectTemplate()

ops := pdf.JS.GetXObjectTemplate(tid, 0, 0, 210, 15, "T", "L")
```

---

## 16. Layers

```go
pdf.StartLayer("Background", true)
pdf.SetFillColor(color.NewGray(0.95))
pdf.Rect(0, 0, 210, 297, "F")
pdf.EndLayer()

pdf.StartLayer("Watermark", true)
pdf.SetFont("Helvetica", "B", 60)
pdf.SetTextColor(color.NewGray(0.8))
pdf.Cell(0, 20, "CONFIDENTIAL", "", 1, "C", false, "")
pdf.EndLayer()
```

---

## 17. Transformations

```go
pdf.StartTransform()
pdf.ScaleXY(1.5, 1.5, 105, 148)   // scale around centre of A4
pdf.Rotate(30, 105, 148)           // degrees
pdf.Rect(80, 130, 50, 30, "FD")
pdf.StopTransform()
```

---

## 18. Gradients and transparency

```go
import "github.com/tecnickcom/go-tcpdf/internal/graph"

stops := []graph.GradientStop{
    {Offset: 0, Color: color.NewRGB(0, 0.4, 0.8)},
    {Offset: 1, Color: color.NewRGB(0.8, 0.9, 1)},
}
linear := graph.NewLinearGradient(0, 0, 210, 0, stops)
radial := graph.NewRadialGradient(105, 148, 0, 105, 148, 100, stops)

pdf.SetAlpha(0.5, "Normal")   // 50% opacity
// ... draw semi-transparent content ...
pdf.SetAlpha(1.0, "Normal")   // restore
```

---

## 19. Encryption

```go
import "github.com/tecnickcom/go-tcpdf/internal/encrypt"

err := pdf.SetEncryption(encrypt.Config{
    Mode:          encrypt.EncAES_256,
    UserPassword:  "userpass",
    OwnerPassword: "ownerpass",
    Permissions:   encrypt.PermPrint | encrypt.PermCopy,
})
```

| Constant | Algorithm | PDF |
|----------|-----------|-----|
| `EncRC4_40` | RC4 40-bit | 1.1+ |
| `EncRC4_128` | RC4 128-bit | 1.4+ |
| `EncAES_128` | AES-128 CBC | 1.6+ |
| `EncAES_256` | AES-256 CBC | 1.7+ |

Permission flags: `PermPrint`, `PermModify`, `PermCopy`, `PermAnnot`,
`PermFillForms`, `PermExtract`, `PermAssemble`, `PermPrintHighQual`, `PermAll`.

---

## 20. Output and streaming

### Method overview

```go
// ── Streaming (preferred — O(objects) RAM, not O(file size)) ────────────

// WriteTo — implements io.WriterTo.
// Works with any io.Writer: files, pipes, HTTP responses, sockets.
// Returns (bytesWritten int64, err error).
n, err := pdf.WriteTo(w)

// Output — alias for WriteTo that discards the byte count.
err := pdf.Output(w)

// SavePDF — streams directly to a file.
err := pdf.SavePDF("report.pdf")

// ── In-memory (convenience) ─────────────────────────────────────────────

// GetOutPDFString — returns the complete PDF as []byte.
// Wraps WriteTo with a bytes.Buffer. Use only for small docs or tests.
data, err := pdf.GetOutPDFString()
```

### HTTP handler

```go
func reportHandler(w http.ResponseWriter, r *http.Request) {
    pdf, err := buildReport()
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
    if _, err := pdf.WriteTo(w); err != nil {
        // Headers already sent — log only
        log.Printf("pdf write error: %v", err)
    }
}
```

### Content-Length pre-calculation

```go
// Dry-run to get size without writing
n, _ := pdf.WriteTo(io.Discard)
w.Header().Set("Content-Length", fmt.Sprintf("%d", n))
w.Header().Set("Content-Type", "application/pdf")
pdf.WriteTo(w)  // second pass — identical content
```

> **Note:** The two-pass serializer is deterministic within a single
> program run, so the size from the dry-run equals the real write.
> The only non-deterministic field is the document ID timestamp, which
> is computed once and reused across both passes.

### Pipe / producer-consumer

```go
pr, pw := io.Pipe()

go func() {
    defer pw.Close()
    pdf.WriteTo(pw)
}()

// Consumer — e.g. S3 multipart upload, gzip, archive
io.Copy(destination, pr)
```

### Buffered file write

```go
f, _ := os.Create("output.pdf")
defer f.Close()

bw := bufio.NewWriterSize(f, 1<<20) // 1 MiB extra buffer
pdf.WriteTo(bw)
bw.Flush()
```

> `SavePDF` already uses a 256 KiB write buffer internally, so extra
> buffering is only needed when calling `WriteTo` directly with an `*os.File`.

### io.Writer compatibility

`WriteTo` accepts any `io.Writer` — it does **not** require `io.WriteSeeker`
or `io.WriterAt`. All PDF cross-reference offsets are computed in the
dry-run (Pass 1) before any bytes reach the real writer (Pass 2).

```go
// All of these work:
pdf.WriteTo(os.Stdout)
pdf.WriteTo(bytes.NewBuffer(nil))
pdf.WriteTo(gzipWriter)
pdf.WriteTo(httpResponseWriter)
pdf.WriteTo(bufio.NewWriter(tcpConn))
```

### output.Output — lower-level API

For cases where you already have a `*output.Document`, use `output.New`
directly:

```go
import "github.com/tecnickcom/go-tcpdf/output"

doc := &output.Document{
    Meta:         meta,
    Pages:        pageManager,
    Fonts:        fontStack,
    // ...
}
out := output.New(doc)

// Same four methods as on *TCPDF:
n, err   := out.WriteTo(w)
err       = out.RenderPDF(w)
data, err = out.GetOutPDFString()
err       = out.SavePDF("out.pdf")
```

---

## 21. Advanced configuration

### Cell geometry defaults

```go
pdf.SetDefaultCellPadding(1, 2, 1, 2)          // TRBL in user units
pdf.SetDefaultCellMargin(0, 1, 0, 1)
pdf.SetDefaultCellBorderPos(base.BorderPosDefault)
// base.BorderPosExternal (-0.5) — outside
// base.BorderPosInternal ( 0.5) — inside
```

### CSS defaults

```go
pdf.SetDefaultCSSMargin(0, 0, 4, 0)
pdf.SetDefaultCSSPadding(2, 4, 2, 4)
pdf.SetDefaultCSSBorderSpacing(2, 2)
```

### RTL documents

```go
pdf.SetRTL(true)
```

### Disable default page content

```go
pdf.EnableDefaultPageContent(false)
```

---

## 22. Package-level APIs

### `internal/color`

```go
color.ParseCSS(s) (Color, error)
color.NewRGB(r, g, b float64) Color           // [0,1]
color.NewRGB255(r, g, b int) Color            // [0,255]
color.NewCMYK(c, m, y, k float64) Color      // [0,1]
color.NewCMYK100(c, m, y, k float64) Color   // [0,100]
color.NewGray(g float64) Color               // [0,1]
color.NewSpot(name string, tint float64) Color
c.WithAlpha(alpha float64) Color
c.FillOperator() string                       // e.g. "1 0 0 rg"
c.StrokeOperator() string                     // e.g. "1 0 0 RG"
c.ToRGB() Color
c.ToCMYK() Color
c.HexString() string                          // "#rrggbb"
color.RGBToCMYK(r, g, b float64) (c, m, y, k float64)
color.CMYKToRGB(c, m, y, k float64) (r, g, b float64)
color.RGBToGray(r, g, b float64) float64
```

### `internal/page`

```go
page.ToPoints(value float64, unit page.Unit) (float64, error)
page.ToUnit(points float64, unit page.Unit) (float64, error)
page.GetFormat(name string) (page.FormatSize, error)
page.BoxFromFormat(name string, orient page.Orientation) (page.Box, error)
```

### `internal/encrypt`

```go
enc, err := encrypt.New(encrypt.Config{...})
enc.EncryptBytes(data []byte) ([]byte, error)
enc.SetObjNum(objNum, genNum int)
enc.FileID() []byte
enc.UserKey() []byte
enc.OwnerKey() []byte
enc.KeyLength() int
enc.Version() int
enc.Revision() int
enc.Disabled() bool
```

### `internal/font`

```go
font.CoreFontMetrics(name string) (*font.FontMetric, error)
font.LineHeight(m *font.FontMetric, size float64) float64
m.GlyphWidth(r rune) float64
m.StringWidth(s string, size float64) float64
m.MarkStringUsed(s string)

stack := font.NewStack(subsetFonts bool)
rk, err := stack.LoadCore(name string) (string, error)
m, ok   := stack.Get(key string) (*font.FontMetric, bool)
w        = stack.TextWidth(key, text string, size float64) float64
```

### `internal/graph`

```go
d := graph.NewDraw()
d.MoveTo(x, y) string
d.LineTo(x, y) string
d.CurveTo(x1, y1, x2, y2, x3, y3) string
d.Rect(x, y, w, h, style) string
d.Circle(cx, cy, r, style) string
d.Ellipse(cx, cy, rx, ry, angle, style) string
d.RoundedRect(x, y, w, h, rx, ry, style) string
d.Arrow(x1, y1, x2, y2, width, headLen, headW) string
d.SetLineWidth(w) string
d.SetDash(array, phase) string
d.SetFillColor(c) string
d.SetStrokeColor(c) string
d.SaveState() string
d.RestoreState() (string, error)
d.SetCTM(m graph.Matrix) string

graph.Identity() Matrix
graph.Translate(tx, ty) Matrix
graph.Scale(sx, sy) Matrix
graph.Rotate(angle) Matrix     // radians
m.Multiply(n Matrix) Matrix
m.PDF() string                 // "a b c d e f cm"
```

### `output` package — direct use

```go
import "github.com/tecnickcom/go-tcpdf/output"

doc := &output.Document{
    Meta:         meta,
    Pages:        pageManager,
    Fonts:        fontStack,
    Images:       imageImport,
    Encrypt:      encryptor,    // nil = no encryption
    Spots:        spotRegistry,
    JS:           jsManager,
    PageContents: []string{"page1 ops", "page2 ops"},
    Compress:     true,
}
out := output.New(doc)

n, err   := out.WriteTo(w)           // streaming, io.WriterTo
err       = out.RenderPDF(w)         // streaming, error only
data, err = out.GetOutPDFString()    // in-memory bytes.Buffer
err       = out.SavePDF("out.pdf")   // file, streaming
mime      := out.GetMIMEAttachmentPDF()  // "application/pdf"
```

### `output/writer.go` — countingWriter

The `countingWriter` is an internal type used by the serializer. It wraps
any `io.Writer` with a `bufio.Writer` (256 KiB) and tracks the number of
bytes emitted. This is how xref byte offsets are computed without seeking.

```
type countingWriter struct {
    w   *bufio.Writer
    pos int64         // bytes written so far
    err error         // first error (sticky)
}
```

You do not normally need to use `countingWriter` directly.

---

## 23. Streaming internals

### Two-pass approach

```
┌─────────────────────────────────────────────────────────────┐
│  Pass 1 (dry-run)                                           │
│                                                             │
│  serialize() ──► countingWriter(io.Discard)                 │
│                                                             │
│  Each startObj() records:  xref[n] = countingWriter.Pos()   │
│  Result: xref[]int64 — one offset per PDF object            │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│  Pass 2 (real write)                                        │
│                                                             │
│  serialize() ──► countingWriter(w)                          │
│                                                             │
│  Identical code path → identical byte layout → xref from   │
│  Pass 1 is correct for Pass 2.                             │
│                                                             │
│  writeXRef() emits the table using Pass-1 offsets.         │
└─────────────────────────────────────────────────────────────┘
```

### Why two passes instead of seek-patch?

The alternative — write a placeholder at the /Pages position, then seek
back and overwrite it — requires `io.WriteSeeker`. HTTP responses,
gzip writers, pipes, and most network connections are not seekable.

The two-pass approach works with **any** `io.Writer` and adds only one
extra traversal of the logical document structure (no file I/O), making
the overhead negligible even for large documents.

### Memory budget

| Item | Size |
|------|------|
| xref table | 8 bytes × number of objects |
| countingWriter buffer | 256 KiB (fixed) |
| Document model | proportional to content |
| **PDF bytes in RAM** | **zero** |

### Determinism guarantee

Both passes call the identical `serialize()` function. The only
potentially non-deterministic element is the document ID in the PDF
trailer, which is based on `time.Now().UnixNano()`. The value is computed
**once** (in `writeXRef`, which runs in both passes), but since the
xref section is at the very end of the file, the offset recorded for it
is only ever read by the `startxref` keyword immediately after — so the
timestamp value does not affect any xref offset.

For encrypted documents, `Encrypt.FileID()` returns a random 16-byte
value generated at `encrypt.New()` time, before either pass runs.
Both passes use the same value.

---

## Full example — HTTP report server

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"

    gotcpdf "github.com/tecnickcom/go-tcpdf"
    "github.com/tecnickcom/go-tcpdf/classobjects"
    "github.com/tecnickcom/go-tcpdf/internal/color"
    "github.com/tecnickcom/go-tcpdf/internal/page"
)

func buildReport(title string) (*gotcpdf.TCPDF, error) {
    pdf, err := gotcpdf.New(classobjects.Config{
        Unit:        "mm",
        Format:      "A4",
        Orientation: page.Portrait,
        Margins:     page.Margins{Top: 20, Right: 15, Bottom: 20, Left: 15},
        Compress:    true,
    })
    if err != nil {
        return nil, err
    }
    pdf.SetTitle(title).SetAuthor("Report Server").SetLanguage("en-US")

    // Cover
    pdf.AddPage()
    pdf.SetFillColor(color.NewRGB(0.1, 0.3, 0.55))
    pdf.Rect(0, 0, 210, 60, "F")
    pdf.SetTextColor(color.White)
    pdf.SetFont("Helvetica", "B", 28)
    pdf.SetXY(15, 18)
    pdf.Cell(0, 12, title, "", 1, "C", false, "")
    pdf.SetFont("Helvetica", "", 12)
    pdf.SetXY(15, 38)
    pdf.Cell(0, 8, time.Now().Format("2 January 2006"), "", 1, "C", false, "")
    pdf.SetTextColor(color.Black)

    // Body
    pdf.AddPage()
    pdf.SetFont("Helvetica", "B", 14)
    pdf.Cell(0, 10, "Summary", "", 1, "L", false, "")
    pdf.SetFont("Helvetica", "", 11)
    pdf.MultiCell(0, 6,
        "This report was generated on demand and streamed directly to the "+
        "HTTP response without buffering the complete PDF in memory.",
        "", "J", false)
    return pdf, nil
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Query().Get("title")
    if title == "" {
        title = "Monthly Report"
    }
    pdf, err := buildReport(title)
    if err != nil {
        http.Error(w, fmt.Sprintf("build error: %v", err), 500)
        return
    }

    // Optional: pre-calculate size for Content-Length
    // n, _ := pdf.WriteTo(io.Discard)
    // w.Header().Set("Content-Length", fmt.Sprintf("%d", n))

    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Set("Content-Disposition",
        fmt.Sprintf(`attachment; filename="%s.pdf"`, title))

    if _, err := pdf.WriteTo(w); err != nil {
        // Headers already flushed — log only
        log.Printf("streaming error: %v", err)
    }
}

func main() {
    http.HandleFunc("/report", reportHandler)
    log.Println("Listening on :8080  →  GET /report?title=Q3+Results")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```
