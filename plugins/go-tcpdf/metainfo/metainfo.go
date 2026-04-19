// Package metainfo provides PDF document metadata management.
// Ported from tc-lib-pdf MetaInfo.php by Nicola Asuni.
package metainfo

import (
	"fmt"
	"strings"
	"time"

	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

const defaultPDFVersion = "1.7"
const defaultProducer = "go-tcpdf"
const defaultCreator = "go-tcpdf"

// ViewerPreferences controls how the document is displayed.
type ViewerPreferences struct {
	HideToolbar           bool
	HideMenubar           bool
	HideWindowUI          bool
	FitWindow             bool
	CenterWindow          bool
	DisplayDocTitle       bool
	NonFullScreenPageMode string // "UseNone", "UseOutlines", "UseThumbs", "UseOC"
	Direction             string // "L2R" or "R2L"
	ViewArea              string
	ViewClip              string
	PrintArea             string
	PrintClip             string
	PrintScaling          string // "None" or "AppDefault"
	Duplex                string // "Simplex", "DuplexFlipShortEdge", "DuplexFlipLongEdge"
	PickTrayByPDFSize     bool
	PrintPageRange        []int
	NumCopies             int
}

// DisplayMode holds zoom and layout settings.
type DisplayMode struct {
	Zoom   interface{} // int percentage or string: "fullpage", "fullwidth", "real", "default"
	Layout string      // "SinglePage", "OneColumn", "TwoColumnLeft", "TwoColumnRight", "TwoPageLeft", "TwoPageRight"
	Mode   string      // "UseNone", "UseOutlines", "UseThumbs", "FullScreen", "UseOC", "UseAttachments"
}

// MetaInfo holds all PDF document metadata.
type MetaInfo struct {
	Title    string
	Author   string
	Subject  string
	Keywords string
	Creator  string
	Producer string

	CreationDate time.Time
	ModDate      time.Time

	PDFVersion string // e.g. "1.7"
	PDFMode    string // "", "pdfa1", "pdfa2", "pdfa3", "pdfx"

	SRGB bool // embed sRGB ICC profile

	CustomXMP map[string]string // key → additional XMP fragment

	ViewerPrefs *ViewerPreferences
	DisplayMode DisplayMode

	// Language tag (RFC 3066)
	Language string
}

// New creates a MetaInfo with sensible defaults.
func New() *MetaInfo {
	return &MetaInfo{
		Creator:      "go-tcpdf",
		Producer:     "go-tcpdf",
		PDFVersion:   defaultPDFVersion,
		CreationDate: time.Now(),
		ModDate:      time.Now(),
		CustomXMP:    make(map[string]string),
		DisplayMode: DisplayMode{
			Zoom:   "default",
			Layout: "SinglePage",
			Mode:   "UseNone",
		},
	}
}

// SetTitle sets the document title. Returns self for chaining.
func (m *MetaInfo) SetTitle(title string) *MetaInfo {
	m.Title = title
	return m
}

// SetAuthor sets the document author.
func (m *MetaInfo) SetAuthor(author string) *MetaInfo {
	m.Author = author
	return m
}

// SetSubject sets the document subject.
func (m *MetaInfo) SetSubject(subject string) *MetaInfo {
	m.Subject = subject
	return m
}

// SetKeywords sets space-separated keywords.
func (m *MetaInfo) SetKeywords(kw string) *MetaInfo {
	m.Keywords = kw
	return m
}

// SetCreator sets the creator application name.
func (m *MetaInfo) SetCreator(creator string) *MetaInfo {
	m.Creator = creator
	if len(m.Creator) == 0 {
		m.Creator = defaultCreator
	}
	return m
}

// SetProducer sets the producer application name.
func (m *MetaInfo) SetProducer(producer string) *MetaInfo {
	m.Producer = producer
	if len(m.Producer) == 0 {
		m.Producer = defaultProducer
	}
	return m
}

// SetPDFVersion validates and sets the PDF version string.
func (m *MetaInfo) SetPDFVersion(version string) (*MetaInfo, error) {
	valid := map[string]bool{
		"1.0": true, "1.1": true, "1.2": true, "1.3": true,
		"1.4": true, "1.5": true, "1.6": true, "1.7": true,
		"2.0": true,
	}
	if !valid[version] {
		return nil, fmt.Errorf("metainfo: invalid PDF version %q", version)
	}
	m.PDFVersion = version
	return m, nil
}

// SetSRGB enables or disables the sRGB ICC color profile.
func (m *MetaInfo) SetSRGB(enabled bool) *MetaInfo {
	m.SRGB = enabled
	return m
}

// SetCustomXMP sets additional XMP data for a specific key.
// Valid keys: "x:xmpmeta", "x:xmpmeta.rdf:RDF", etc.
func (m *MetaInfo) SetCustomXMP(key, xmp string) *MetaInfo {
	m.CustomXMP[key] = xmp
	return m
}

// SetViewerPreferences sets the viewer preferences dictionary.
func (m *MetaInfo) SetViewerPreferences(vp ViewerPreferences) *MetaInfo {
	m.ViewerPrefs = &vp
	return m
}

// GetVersion returns the go-tcpdf version string.
func (m *MetaInfo) GetVersion() string {
	return "go-tcpdf v1.0.0 (TCPDF port)"
}

// ---- PDF output helpers -------------------------------------------------

// InfoDictEntries returns the /Info dictionary entries as PDF dict fragment.
func (m *MetaInfo) InfoDictEntries(enc func(string) string) string {
	var sb strings.Builder
	write := func(key, val string) {
		if val != "" {
			sb.WriteByte('/')
			sb.WriteString(key)
			sb.WriteString(" (")
			sb.WriteString(enc(val))
			sb.WriteString(")\n")
		}
	}
	write("Title", m.Title)
	write("Author", m.Author)
	write("Subject", m.Subject)
	write("Keywords", m.Keywords)
	write("Creator", m.Creator)
	write("Producer", m.Producer)
	write("PoweredBy", defaultCreator)
	if !m.CreationDate.IsZero() {
		sb.WriteString("/CreationDate (D:")
		sb.WriteString(pdfDate(m.CreationDate))
		sb.WriteString(")\n")
	}
	if !m.ModDate.IsZero() {
		sb.WriteString("/ModDate (D:")
		sb.WriteString(pdfDate(m.ModDate))
		sb.WriteString(")\n")
	}
	return sb.String()
}

// ViewerPrefDict returns the /ViewerPreferences dictionary fragment.
func (m *MetaInfo) ViewerPrefDict() string {
	if m.ViewerPrefs == nil {
		return ""
	}
	vp := m.ViewerPrefs
	var sb strings.Builder
	boolEntry := func(key string, val bool) {
		if val {
			sb.WriteByte('/')
			sb.WriteString(key)
			sb.WriteString(" true\n")
		}
	}
	boolEntry("HideToolbar", vp.HideToolbar)
	boolEntry("HideMenubar", vp.HideMenubar)
	boolEntry("HideWindowUI", vp.HideWindowUI)
	boolEntry("FitWindow", vp.FitWindow)
	boolEntry("CenterWindow", vp.CenterWindow)
	boolEntry("DisplayDocTitle", vp.DisplayDocTitle)
	if vp.NonFullScreenPageMode != "" {
		sb.WriteString("/NonFullScreenPageMode /")
		sb.WriteString(vp.NonFullScreenPageMode)
		sb.WriteByte('\n')
	}
	if vp.Direction != "" {
		sb.WriteString("/Direction /")
		sb.WriteString(vp.Direction)
		sb.WriteByte('\n')
	}
	if vp.PrintScaling != "" {
		sb.WriteString("/PrintScaling /")
		sb.WriteString(vp.PrintScaling)
		sb.WriteByte('\n')
	}
	if vp.Duplex != "" {
		sb.WriteString("/Duplex /")
		sb.WriteString(vp.Duplex)
		sb.WriteByte('\n')
	}
	if vp.NumCopies > 1 {
		sb.WriteString("/NumCopies ")
		sb.WriteString(pdfbuf.FmtI(vp.NumCopies))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// XMPData generates the XMP metadata XML packet.
func (m *MetaInfo) XMPData() string {
	var sb strings.Builder
	sb.WriteString("<?xpacket begin=\"\xef\xbb\xbf\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>\n")
	sb.WriteString("<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">\n")
	sb.WriteString("<rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n")
	sb.WriteString("<rdf:Description rdf:about=\"\"\n")
	sb.WriteString(" xmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n")
	sb.WriteString(" xmlns:xmp=\"http://ns.adobe.com/xap/1.0/\"\n")
	sb.WriteString(" xmlns:pdf=\"http://ns.adobe.com/pdf/1.3/\">\n")

	if m.Title != "" {
		sb.WriteString("<dc:title><rdf:Alt><rdf:li xml:lang=\"x-default\">")
		sb.WriteString(xmlEscape(m.Title))
		sb.WriteString("</rdf:li></rdf:Alt></dc:title>\n")
	}
	if m.Author != "" {
		sb.WriteString("<dc:creator><rdf:Seq><rdf:li>")
		sb.WriteString(xmlEscape(m.Author))
		sb.WriteString("</rdf:li></rdf:Seq></dc:creator>\n")
	}
	if m.Subject != "" {
		sb.WriteString("<dc:description><rdf:Alt><rdf:li xml:lang=\"x-default\">")
		sb.WriteString(xmlEscape(m.Subject))
		sb.WriteString("</rdf:li></rdf:Alt></dc:description>\n")
	}
	if m.Keywords != "" {
		sb.WriteString("<pdf:Keywords>")
		sb.WriteString(xmlEscape(m.Keywords))
		sb.WriteString("</pdf:Keywords>\n")
	}
	sb.WriteString("<xmp:CreateDate>")
	sb.WriteString(m.CreationDate.Format(time.RFC3339))
	sb.WriteString("</xmp:CreateDate>\n")
	sb.WriteString("<xmp:ModifyDate>")
	sb.WriteString(m.ModDate.Format(time.RFC3339))
	sb.WriteString("</xmp:ModifyDate>\n")
	sb.WriteString("<xmp:CreatorTool>")
	sb.WriteString(xmlEscape(m.Creator))
	sb.WriteString("</xmp:CreatorTool>\n")

	// Custom XMP fragments
	for _, frag := range m.CustomXMP {
		sb.WriteString(frag)
	}

	sb.WriteString("</rdf:Description>\n</rdf:RDF>\n</x:xmpmeta>\n")
	sb.WriteString("<?xpacket end=\"w\"?>")
	return sb.String()
}

// ---- helpers ------------------------------------------------------------

func pdfDate(t time.Time) string {
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	offsetHours := offset / 3600
	offsetMins := (offset % 3600) / 60
	var b strings.Builder
	b.WriteString(t.Format("20060102150405"))
	b.WriteString(sign)
	writePad2 := func(n int) {
		if n < 10 {
			b.WriteByte('0')
		}
		b.WriteString(pdfbuf.FmtI(n))
	}
	writePad2(offsetHours)
	b.WriteByte('\'')
	writePad2(offsetMins)
	b.WriteByte('\'')
	return b.String()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
