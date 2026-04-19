// Package page provides PDF page formats, boxes, unit conversion.
package page

import (
	"errors"
	"sync"
)

type Unit string

const (
	UnitPt Unit = "pt"
	UnitMM Unit = "mm"
	UnitCM Unit = "cm"
	UnitIN Unit = "in"
	UnitPX Unit = "px"
)

var pointsPerUnit = map[Unit]float64{
	UnitPt: 1.0,
	UnitMM: 72.0 / 25.4,
	UnitCM: 72.0 / 2.54,
	UnitIN: 72.0,
	UnitPX: 72.0 / 96.0,
}

func ToPoints(value float64, unit Unit) (float64, error) {
	k, ok := pointsPerUnit[unit]
	if !ok {
		return 0, errors.New("page: unknown unit: " + string(unit))
	}
	return value * k, nil
}

func ToUnit(points float64, unit Unit) (float64, error) {
	k, ok := pointsPerUnit[unit]
	if !ok {
		return 0, errors.New("page: unknown unit: " + string(unit))
	}
	return points / k, nil
}

func ScaleFactor(unit Unit) (float64, error) {
	k, ok := pointsPerUnit[unit]
	if !ok {
		return 0, errors.New("page: unknown unit: " + string(unit))
	}
	return k, nil
}

type Orientation string

const (
	Portrait  Orientation = "P"
	Landscape Orientation = "L"
)

type Box struct {
	Llx, Lly float64
	Urx, Ury float64
}

func (b Box) Width() float64  { return b.Urx - b.Llx }
func (b Box) Height() float64 { return b.Ury - b.Lly }

type PageBoxes struct {
	Media, Crop, Bleed, Trim, Art Box
}

type Margins struct {
	Top, Right, Bottom, Left float64
}

type PageData struct {
	Index       int
	ID          int
	Boxes       PageBoxes
	Orientation Orientation
	Unit        Unit
	ScaleFactor float64
	Margins     Margins
	Content     string
	AnnotRefs   []int
	Layers      []string
	Rotation    int
}

func (p *PageData) Width() float64  { return p.Boxes.Media.Width() / p.ScaleFactor }
func (p *PageData) Height() float64 { return p.Boxes.Media.Height() / p.ScaleFactor }

type FormatSize struct{ W, H float64 }

var PageFormats = map[string]FormatSize{
	"A0": {2383.94, 3370.39}, "A1": {1683.78, 2383.94}, "A2": {1190.55, 1683.78},
	"A3": {841.89, 1190.55}, "A4": {595.28, 841.89}, "A5": {419.53, 595.28},
	"A6": {297.64, 419.53}, "A7": {209.76, 297.64}, "A8": {147.40, 209.76},
	"A9": {104.88, 147.40}, "A10": {73.70, 104.88},
	"B0": {2834.65, 4008.19}, "B1": {2004.09, 2834.65}, "B2": {1417.32, 2004.09},
	"B3": {1000.63, 1417.32}, "B4": {708.66, 1000.63}, "B5": {498.90, 708.66},
	"B6": {354.33, 498.90}, "B7": {249.45, 354.33}, "B8": {175.75, 249.45},
	"B9": {124.72, 175.75}, "B10": {87.87, 124.72},
	"C0": {2599.37, 3676.54}, "C1": {1836.85, 2599.37}, "C2": {1298.27, 1836.85},
	"C3": {918.43, 1298.27}, "C4": {649.13, 918.43}, "C5": {459.21, 649.13},
	"C6": {323.15, 459.21}, "C7": {229.61, 323.15}, "C8": {161.57, 229.61},
	"C9": {113.39, 161.57}, "C10": {79.37, 113.39},
	"LETTER":    {612.00, 792.00}, "LEGAL": {612.00, 1008.00},
	"LEDGER":    {1224.00, 792.00}, "TABLOID": {792.00, 1224.00},
	"EXECUTIVE": {521.86, 756.00}, "FOLIO": {612.00, 936.00},
	"COMMERCIAL11": {396.00, 684.00},
	"B4J": {728.50, 1031.81}, "B5J": {515.91, 728.50},
	"CREDIT": {153.07, 242.65},
	"BROADSHEET": {2038.48, 2976.38}, "BERLINER": {1474.02, 2155.91},
	"TABLOID_NP": {1039.37, 1474.02},
}

func GetFormat(name string) (FormatSize, error) {
	upper := toUpper(name)
	if f, ok := PageFormats[upper]; ok {
		return f, nil
	}
	return FormatSize{}, errors.New("page: unknown format: " + name)
}

func NewBox(w, h float64) Box { return Box{0, 0, w, h} }

func BoxFromFormat(name string, orient Orientation) (Box, error) {
	f, err := GetFormat(name)
	if err != nil {
		return Box{}, err
	}
	w, h := f.W, f.H
	if orient == Landscape && w < h {
		w, h = h, w
	} else if orient == Portrait && w > h {
		w, h = h, w
	}
	return NewBox(w, h), nil
}

// Page manages the list of pages. Thread-safe.
type Page struct {
	mu     sync.RWMutex
	pages  []*PageData
	unit   Unit
	kUnit  float64
	defOri Orientation
	defFmt string
	defMar Margins
}

func New(unit Unit, format string, orient Orientation, margins Margins) (*Page, error) {
	k, err := ScaleFactor(unit)
	if err != nil {
		return nil, err
	}
	return &Page{unit: unit, kUnit: k, defOri: orient, defFmt: format, defMar: margins}, nil
}

func (p *Page) Add(format string, orient Orientation, margins *Margins) (*PageData, error) {
	if format == "" {
		format = p.defFmt
	}
	if orient == "" {
		orient = p.defOri
	}
	if margins == nil {
		margins = &p.defMar
	}
	box, err := BoxFromFormat(format, orient)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	pd := &PageData{
		Index:       len(p.pages),
		Boxes:       PageBoxes{Media: box, Crop: box, Bleed: box, Trim: box, Art: box},
		Orientation: orient,
		Unit:        p.unit,
		ScaleFactor: p.kUnit,
		Margins:     *margins,
	}
	p.pages = append(p.pages, pd)
	return pd, nil
}

func (p *Page) Current() *PageData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.pages) == 0 {
		return nil
	}
	return p.pages[len(p.pages)-1]
}

func (p *Page) Get(index int) (*PageData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if index < 0 || index >= len(p.pages) {
		return nil, errors.New("page: index out of range")
	}
	return p.pages[index], nil
}

func (p *Page) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.pages)
}

func (p *Page) All() []*PageData {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cp := make([]*PageData, len(p.pages))
	copy(cp, p.pages)
	return cp
}

func toUpper(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		b[i] = c
	}
	return string(b)
}
