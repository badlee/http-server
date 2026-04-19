// Package font provides PDF font management.
package font

import (
	"errors"
	"math"
	"strings"
	"sync"

	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

type FontType string

const (
	FontTypeCore     FontType = "core"
	FontTypeTrueType FontType = "TrueType"
	FontTypeOpenType FontType = "OpenType"
	FontTypeCIDType0 FontType = "CIDType0"
	FontTypeType1    FontType = "Type1"
)

type FontMetric struct {
	Family         string
	Name           string
	Type           FontType
	Style          string
	Ascent         float64
	Descent        float64
	CapHeight      float64
	XHeight        float64
	UnderlinePos   float64
	UnderlineThick float64
	ItalicAngle    float64
	Leading        float64
	BBox           [4]float64
	Widths         map[rune]float64
	MissingWidth   float64
	Encoding       string
	IsUnicode      bool
	Flags          uint32
	Data           []byte
	Compressed     bool
	UsedGlyphs     map[rune]bool
	ObjNum         int
	DescObjNum     int
	CIDEncoding    string
}

func (m *FontMetric) GlyphWidth(r rune) float64 {
	if w, ok := m.Widths[r]; ok {
		return w
	}
	return m.MissingWidth
}

func (m *FontMetric) StringWidth(s string, size float64) float64 {
	var total float64
	for _, r := range s {
		total += m.GlyphWidth(r)
	}
	return total * size / 1000.0
}

func (m *FontMetric) MarkUsed(r rune) {
	if m.UsedGlyphs == nil {
		m.UsedGlyphs = make(map[rune]bool)
	}
	m.UsedGlyphs[r] = true
}

func (m *FontMetric) MarkStringUsed(s string) {
	for _, r := range s {
		m.MarkUsed(r)
	}
}

// CoreFontMetrics returns metrics for a standard PDF font (copy per call).
func CoreFontMetrics(name string) (*FontMetric, error) {
	m, ok := coreFonts[strings.ToLower(name)]
	if !ok {
		return nil, errors.New("font: unknown core font: " + name)
	}
	cp := m
	cp.Widths = make(map[rune]float64, len(m.Widths))
	for k, v := range m.Widths {
		cp.Widths[k] = v
	}
	return &cp, nil
}

// ---- Stack (thread-safe) ------------------------------------------------

// Stack manages fonts loaded into a PDF document. All methods are safe
// for concurrent use.
type Stack struct {
	mu      sync.RWMutex
	fonts   map[string]*FontMetric
	byName  map[string]*FontMetric
	counter int
	subset  bool
}

func NewStack(subsetFonts bool) *Stack {
	return &Stack{
		fonts:  make(map[string]*FontMetric),
		byName: make(map[string]*FontMetric),
		subset: subsetFonts,
	}
}

func (s *Stack) LoadCore(name string) (string, error) {
	key := strings.ToLower(name)
	s.mu.RLock()
	if _, ok := s.byName[key]; ok {
		for rk, m := range s.fonts {
			if strings.ToLower(m.Name) == key {
				s.mu.RUnlock()
				return rk, nil
			}
		}
	}
	s.mu.RUnlock()

	m, err := CoreFontMetrics(name)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock
	if _, ok := s.byName[key]; ok {
		for rk, fm := range s.fonts {
			if strings.ToLower(fm.Name) == key {
				return rk, nil
			}
		}
	}
	s.counter++
	var rb strings.Builder
	rb.WriteByte('F')
	rb.WriteString(pdfbuf.FmtI(s.counter))
	rk := rb.String()
	s.fonts[rk] = m
	s.byName[key] = m
	return rk, nil
}

func (s *Stack) Add(m *FontMetric) string {
	key := strings.ToLower(m.Name)
	s.mu.RLock()
	if existing, ok := s.byName[key]; ok {
		for rk, em := range s.fonts {
			if em == existing {
				s.mu.RUnlock()
				return rk
			}
		}
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	var rb strings.Builder
	rb.WriteByte('F')
	rb.WriteString(pdfbuf.FmtI(s.counter))
	rk := rb.String()
	s.fonts[rk] = m
	s.byName[key] = m
	return rk
}

func (s *Stack) Get(rk string) (*FontMetric, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.fonts[rk]
	return m, ok
}

// All returns a snapshot copy of the font map (safe for concurrent iteration).
func (s *Stack) All() map[string]*FontMetric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]*FontMetric, len(s.fonts))
	for k, v := range s.fonts {
		cp[k] = v
	}
	return cp
}

func (s *Stack) SubsetEnabled() bool { return s.subset }

func (s *Stack) MarkUsed(rk, text string) {
	if !s.subset {
		return
	}
	s.mu.RLock()
	m, ok := s.fonts[rk]
	s.mu.RUnlock()
	if ok {
		m.MarkStringUsed(text)
	}
}

func (s *Stack) CharWidth(rk string, r rune, size float64) float64 {
	s.mu.RLock()
	m, ok := s.fonts[rk]
	s.mu.RUnlock()
	if !ok {
		return 0
	}
	return m.GlyphWidth(r) * size / 1000.0
}

func (s *Stack) TextWidth(rk, text string, size float64) float64 {
	s.mu.RLock()
	m, ok := s.fonts[rk]
	s.mu.RUnlock()
	if !ok {
		return 0
	}
	return m.StringWidth(text, size)
}

// ---- Line height helpers ------------------------------------------------

func LineHeight(m *FontMetric, size float64) float64 {
	return (m.Ascent - m.Descent) * size / 1000.0
}

func CapHeight(m *FontMetric, size float64) float64 {
	return math.Abs(m.CapHeight) * size / 1000.0
}

// ---- Width tables for core fonts ----------------------------------------

func latinWidths(ws []float64) map[rune]float64 {
	m := make(map[rune]float64, len(ws))
	for i, w := range ws {
		m[rune(0x0020+i)] = w
	}
	return m
}

var coreFonts = map[string]FontMetric{
	"helvetica": {
		Family: "Helvetica", Name: "Helvetica", Type: FontTypeCore,
		Ascent: 718, Descent: -207, CapHeight: 718, XHeight: 523,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: 0,
		BBox: [4]float64{-166, -225, 1000, 931},
		MissingWidth: 278, Widths: helveticaWidths, Flags: 32,
	},
	"helvetica-bold": {
		Family: "Helvetica", Name: "Helvetica-Bold", Type: FontTypeCore, Style: "B",
		Ascent: 718, Descent: -207, CapHeight: 718, XHeight: 532,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: 0,
		BBox: [4]float64{-170, -228, 1003, 962},
		MissingWidth: 278, Widths: helveticaBoldWidths, Flags: 32,
	},
	"helvetica-oblique": {
		Family: "Helvetica", Name: "Helvetica-Oblique", Type: FontTypeCore, Style: "I",
		Ascent: 718, Descent: -207, CapHeight: 718, XHeight: 523,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: -12,
		BBox: [4]float64{-170, -225, 1116, 931},
		MissingWidth: 278, Widths: helveticaWidths, Flags: 96,
	},
	"helvetica-boldoblique": {
		Family: "Helvetica", Name: "Helvetica-BoldOblique", Type: FontTypeCore, Style: "BI",
		Ascent: 718, Descent: -207, CapHeight: 718, XHeight: 532,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: -12,
		BBox: [4]float64{-174, -228, 1114, 962},
		MissingWidth: 278, Widths: helveticaBoldWidths, Flags: 96,
	},
	"times-roman": {
		Family: "Times", Name: "Times-Roman", Type: FontTypeCore,
		Ascent: 683, Descent: -217, CapHeight: 662, XHeight: 450,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: 0,
		BBox: [4]float64{-168, -218, 1000, 898},
		MissingWidth: 250, Widths: timesWidths, Flags: 34,
	},
	"times-bold": {
		Family: "Times", Name: "Times-Bold", Type: FontTypeCore, Style: "B",
		Ascent: 683, Descent: -217, CapHeight: 676, XHeight: 461,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: 0,
		BBox: [4]float64{-168, -218, 1000, 935},
		MissingWidth: 250, Widths: timesBoldWidths, Flags: 34,
	},
	"times-italic": {
		Family: "Times", Name: "Times-Italic", Type: FontTypeCore, Style: "I",
		Ascent: 683, Descent: -217, CapHeight: 653, XHeight: 441,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: -15.5,
		BBox: [4]float64{-169, -217, 1010, 883},
		MissingWidth: 250, Widths: timesItalicWidths, Flags: 98,
	},
	"times-bolditalic": {
		Family: "Times", Name: "Times-BoldItalic", Type: FontTypeCore, Style: "BI",
		Ascent: 683, Descent: -217, CapHeight: 669, XHeight: 462,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: -15,
		BBox: [4]float64{-200, -218, 996, 921},
		MissingWidth: 250, Widths: timesBoldItalicWidths, Flags: 98,
	},
	"courier": {
		Family: "Courier", Name: "Courier", Type: FontTypeCore,
		Ascent: 629, Descent: -157, CapHeight: 562, XHeight: 426,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: 0,
		BBox: [4]float64{-23, -250, 715, 805},
		MissingWidth: 600, Widths: courierWidths, Flags: 35,
	},
	"courier-bold": {
		Family: "Courier", Name: "Courier-Bold", Type: FontTypeCore, Style: "B",
		Ascent: 629, Descent: -157, CapHeight: 562, XHeight: 439,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: 0,
		BBox: [4]float64{-113, -250, 749, 801},
		MissingWidth: 600, Widths: courierWidths, Flags: 35,
	},
	"courier-oblique": {
		Family: "Courier", Name: "Courier-Oblique", Type: FontTypeCore, Style: "I",
		Ascent: 629, Descent: -157, CapHeight: 562, XHeight: 426,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: -12,
		BBox: [4]float64{-27, -250, 849, 805},
		MissingWidth: 600, Widths: courierWidths, Flags: 99,
	},
	"courier-boldoblique": {
		Family: "Courier", Name: "Courier-BoldOblique", Type: FontTypeCore, Style: "BI",
		Ascent: 629, Descent: -157, CapHeight: 562, XHeight: 439,
		UnderlinePos: -100, UnderlineThick: 50, ItalicAngle: -12,
		BBox: [4]float64{-57, -250, 869, 801},
		MissingWidth: 600, Widths: courierWidths, Flags: 99,
	},
	"symbol": {
		Family: "Symbol", Name: "Symbol", Type: FontTypeCore,
		MissingWidth: 250, Widths: symbolWidths, Flags: 4,
	},
	"zapfdingbats": {
		Family: "ZapfDingbats", Name: "ZapfDingbats", Type: FontTypeCore,
		MissingWidth: 278, Widths: zapfWidths, Flags: 4,
	},
}

var helveticaWidths = latinWidths([]float64{
	278, 278, 355, 556, 556, 889, 667, 222, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556,
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556,
	222, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556,
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584,
})
var helveticaBoldWidths = latinWidths([]float64{
	278, 333, 474, 556, 556, 889, 722, 278, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 333, 333, 584, 584, 584, 611,
	975, 722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 333, 278, 333, 584, 556,
	278, 556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611,
	611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500, 389, 280, 389, 584,
})
var timesWidths = latinWidths([]float64{
	250, 333, 408, 500, 500, 833, 778, 333, 333, 333, 500, 564, 250, 333, 250, 278,
	500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 278, 278, 564, 564, 564, 444,
	921, 722, 667, 667, 722, 611, 556, 722, 722, 333, 389, 722, 611, 889, 722, 722,
	556, 722, 667, 556, 611, 722, 722, 944, 722, 722, 611, 333, 278, 333, 469, 500,
	333, 444, 500, 444, 500, 444, 333, 500, 500, 278, 278, 500, 278, 778, 500, 500,
	500, 500, 333, 389, 278, 500, 500, 722, 500, 500, 444, 480, 200, 480, 541,
})
var timesBoldWidths = latinWidths([]float64{
	250, 333, 555, 500, 500, 1000, 833, 333, 333, 333, 500, 570, 250, 333, 250, 278,
	500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 333, 333, 570, 570, 570, 500,
	930, 722, 667, 722, 722, 667, 611, 778, 778, 389, 500, 778, 667, 944, 722, 778,
	611, 778, 722, 556, 667, 722, 722, 1000, 722, 722, 667, 333, 278, 333, 581, 500,
	333, 500, 556, 444, 556, 444, 333, 500, 556, 278, 333, 556, 278, 833, 556, 500,
	556, 556, 444, 389, 333, 556, 500, 722, 500, 500, 444, 394, 220, 394, 520,
})
var timesItalicWidths = latinWidths([]float64{
	250, 333, 420, 500, 500, 833, 778, 333, 333, 333, 500, 675, 250, 333, 250, 278,
	500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 333, 333, 675, 675, 675, 500,
	920, 611, 611, 667, 722, 611, 611, 722, 722, 333, 444, 667, 556, 833, 667, 722,
	611, 722, 611, 500, 556, 722, 611, 833, 611, 556, 556, 389, 278, 389, 422, 500,
	333, 500, 500, 444, 500, 444, 278, 500, 500, 278, 278, 444, 278, 722, 500, 500,
	500, 500, 389, 389, 278, 500, 444, 667, 444, 444, 389, 400, 275, 400, 541,
})
var timesBoldItalicWidths = latinWidths([]float64{
	250, 389, 555, 500, 500, 833, 778, 333, 333, 333, 500, 570, 250, 333, 250, 278,
	500, 500, 500, 500, 500, 500, 500, 500, 500, 500, 333, 333, 570, 570, 570, 500,
	832, 667, 667, 667, 722, 667, 667, 722, 778, 389, 500, 667, 611, 889, 722, 722,
	611, 722, 667, 556, 611, 722, 667, 889, 667, 611, 611, 333, 278, 333, 570, 500,
	333, 500, 500, 444, 500, 444, 333, 500, 556, 278, 278, 500, 278, 778, 556, 500,
	500, 500, 389, 389, 278, 556, 444, 667, 500, 444, 389, 348, 220, 348, 570,
})
var courierWidths = latinWidths(func() []float64 {
	ws := make([]float64, 224)
	for i := range ws {
		ws[i] = 600
	}
	return ws
}())
var symbolWidths = map[rune]float64{' ': 250, '!': 333, '"': 713, '#': 500}
var zapfWidths = map[rune]float64{' ': 278}
