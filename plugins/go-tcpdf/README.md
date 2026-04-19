# go-tcpdf

A faithful 1:1 Go port of [tc-lib-pdf (TCPDF)](https://github.com/tecnickcom/tc-lib-pdf) by Nicola Asuni.

Generates PDF documents from pure Go — no CGO, no external binaries, zero native dependencies.

---

## Features

| Category | Details |
|---|---|
| **Fonts** | 14 standard PDF fonts with full AFM metrics; TrueType/OpenType embedding with optional subsetting |
| **Images** | JPEG, PNG (with alpha/soft-mask) |
| **Vector** | SVG→PDF conversion: path, circle, rect, ellipse, polyline, transform, matrix |
| **HTML** | HTML→PDF: inline styles, CSS selectors, block/inline layout |
| **Colors** | DeviceRGB, DeviceCMYK, DeviceGray, Spot/Separation, ICC profiles (sRGB) |
| **Encryption** | RC4-40, RC4-128, AES-128, AES-256 with full permission flags |
| **Interactive** | Annotations, links, bookmarks/outlines, named destinations |
| **Forms** | Text, checkbox, radio, combo, list, button (AcroForm) |
| **JavaScript** | Document-level and object-level JS embedding |
| **XObjects** | Reusable Form XObject templates |
| **Attachments** | Embedded files with AFRelationship |
| **Layers** | Optional Content Groups (PDF layers) |
| **TOC** | Automatic Table of Contents |
| **Transforms** | Scale, rotate, translate via CTM |
| **Text** | Line wrapping, TeX hyphenation, underline/strikethrough/overline, justification, RTL |
| **Gradients** | Linear and radial, multi-stop |
| **Transparency** | Alpha channel, blend modes (ExtGState) |
| **Metadata** | XMP, PDF/A hooks, viewer preferences |
| **Formats** | A0–A10, B0–B10, C0–C10, Letter, Legal, Ledger, Tabloid, Executive and more |
| **Units** | pt, mm, cm, in, px |
| **Output** | **Streaming `io.Writer` (zero extra RAM)**, in-memory `[]byte`, direct file save |

---

## Installation

```bash
go get github.com/tecnickcom/go-tcpdf
```

**Dependencies** (resolved by `go mod tidy`):

```
golang.org/x/text   >= v0.14.0
golang.org/x/image  >= v0.15.0
golang.org/x/net    >= v0.20.0   # HTML parser
```

---

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    gotcpdf "github.com/tecnickcom/go-tcpdf"
    "github.com/tecnickcom/go-tcpdf/classobjects"
)

func main() {
    pdf, err := gotcpdf.New(classobjects.DefaultConfig()) // A4, mm, portrait
    if err != nil {
        log.Fatal(err)
    }

    pdf.SetTitle("Hello World").SetAuthor("Acme Corp")
    pdf.AddPage()
    pdf.SetFont("Helvetica", "B", 16)
    pdf.Cell(0, 10, "Hello, World!", "1", 1, "C", false, "")

    // Option A — stream directly to HTTP response (zero RAM overhead)
    http.HandleFunc("/pdf", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/pdf")
        pdf.WriteTo(w) // implements io.WriterTo
    })

    // Option B — save to disk (streaming, never holds full PDF in RAM)
    pdf.SavePDF("hello.pdf")

    // Option C — get bytes (convenient for small docs / tests)
    data, _ := pdf.GetOutPDFString()
    _ = data
}
```

---

## Streaming Output

go-tcpdf uses a **two-pass streaming serializer** that writes PDF bytes
directly to any `io.Writer` without buffering the complete document in RAM.

```
Pass 1 ── serialize → io.Discard ── collect xref byte offsets
Pass 2 ── serialize → real writer ── emit bytes with correct offsets
```

Memory overhead is O(objects), not O(file size):

| Overhead | Size |
|---|---|
| xref table | ~8 bytes × number of PDF objects |
| Write buffer | 256 KiB (fixed) |
| **Typical 100-page doc (~300 objects)** | **< 300 KiB extra RAM** |

### Output methods

```go
// ── Streaming (preferred) ──────────────────────────────────────────────

// WriteTo — implements io.WriterTo, returns bytes written.
// Works with any io.Writer: files, pipes, sockets, HTTP responses.
n, err := pdf.WriteTo(w)

// Output — alias for WriteTo that discards the byte count.
err := pdf.Output(w)

// SavePDF — streams directly to a file (calls WriteTo internally).
err := pdf.SavePDF("report.pdf")

// ── In-memory (convenience) ────────────────────────────────────────────

// GetOutPDFString — returns []byte. Uses a bytes.Buffer internally.
// Holds the entire PDF in RAM. Use for small docs or tests only.
data, err := pdf.GetOutPDFString()
```

### HTTP handler example

```go
func pdfHandler(w http.ResponseWriter, r *http.Request) {
    pdf, _ := gotcpdf.NewDefault()
    pdf.SetTitle("Invoice #1042")
    pdf.AddPage()
    pdf.SetFont("Helvetica", "B", 18)
    pdf.Cell(0, 12, "Invoice #1042", "", 1, "L", false, "")
    // ... more content ...

    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Set("Content-Disposition", `attachment; filename="invoice-1042.pdf"`)
    if _, err := pdf.WriteTo(w); err != nil {
        http.Error(w, "PDF generation failed", http.StatusInternalServerError)
    }
}
```

### Pipe example

```go
pr, pw := io.Pipe()

go func() {
    defer pw.Close()
    pdf.WriteTo(pw) // producer
}()

io.Copy(uploader, pr) // consumer — e.g. S3 multipart upload
```

### Size estimation without writing

```go
n, _ := pdf.WriteTo(io.Discard) // dry-run: get size without writing
w.Header().Set("Content-Length", fmt.Sprintf("%d", n))
pdf.WriteTo(w) // second pass writes the real bytes
```

---

## Project Structure

```
go-tcpdf/
├── tcpdf.go                   Public TCPDF type — full API surface
├── go.mod
│
├── classobjects/              Dependency injection / wiring
├── base/                      Unit conversion, PON counter, RTL
├── cell/                      Cell margin/padding/border defaults
├── metainfo/                  Metadata, XMP, viewer preferences
├── output/
│   ├── output.go              Two-pass streaming PDF serializer
│   └── writer.go              countingWriter — tracks byte position
├── javascript/                Annotations, links, bookmarks, forms, XObjects
├── text/                      Text rendering, wrapping, hyphenation
├── css/                       CSS property parsing
├── html/                      HTML → DOM → PDF
├── svg/                       SVG → PDF vector conversion
├── layers/                    Optional Content Groups
├── toc/                       Table of Contents
│
└── internal/
    ├── color/                 RGB/CMYK/Gray/Spot, CSS parsing
    ├── encrypt/               RC4/AES encryption
    ├── font/                  Metrics, 14 core fonts, stack
    ├── graph/                 CTM, paths, gradients, ExtGState
    ├── image/                 JPEG/PNG import
    ├── page/                  Page formats, boxes, units
    ├── unicode/               Bidi, UTF-16BE, TeX hyphenator
    └── cache/                 Thread-safe key-value cache
```

---

## Configuration

```go
cfg := classobjects.Config{
    Unit:        "mm",           // "pt", "mm", "cm", "in", "px"
    Format:      "A4",           // any key from internal/page.PageFormats
    Orientation: page.Portrait,  // page.Portrait or page.Landscape
    Margins: page.Margins{
        Top: 10, Right: 10, Bottom: 10, Left: 10,
    },
    RTL:         false,   // right-to-left document
    Unicode:     true,
    SubsetFonts: true,    // embed only used glyphs
    Compress:    true,    // zlib-compress page streams
}
pdf, err := gotcpdf.New(cfg)
```

---

## License

MIT — same as the original tc-lib-pdf.

Original PHP library copyright © Nicola Asuni – Tecnick.com LTD.
