// Package base provides base types and helpers shared across all TCPDF modules.
// Ported from tc-lib-pdf Base.php by Nicola Asuni.
package base

import (
	"sync"
)

// ---- Border position constants (Table 22, TCPDF) -------------------------

const (
	// BorderPosDefault: border centered on the cell edge.
	BorderPosDefault float64 = 0
	// BorderPosExternal: border external to the cell edge.
	BorderPosExternal float64 = -0.5
	// BorderPosInternal: border internal to the cell edge.
	BorderPosInternal float64 = 0.5
)

// ---- Cell boundary ------------------------------------------------------

// CellBound holds top/right/bottom/left spacing values.
type CellBound struct {
	T, R, B, L float64
}

// ZeroCellBound is a CellBound with all zeros.
var ZeroCellBound = CellBound{0, 0, 0, 0}

// CellDef holds margin, padding and border-position for a cell.
type CellDef struct {
	Margin    CellBound
	Padding   CellBound
	BorderPos float64
}

// ZeroCell is a CellDef with all zeros.
var ZeroCell = CellDef{
	Margin:    ZeroCellBound,
	Padding:   ZeroCellBound,
	BorderPos: BorderPosDefault,
}

// ---- Text bounding box --------------------------------------------------

// BBox is a text bounding box [llx, lly, urx, ury] in user units.
type BBox struct {
	Llx, Lly, Urx, Ury float64
}

// ---- PDF Object Number (PON) counter ------------------------------------

// PON is a thread-safe PDF object number counter.
type PON struct {
	mu  sync.Mutex
	val int
}

// Next increments and returns the next object number.
func (p *PON) Next() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.val++
	return p.val
}

// Current returns the current object number without incrementing.
func (p *PON) Current() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.val
}

// Set forces the counter to a specific value (used during xref loading).
func (p *PON) Set(v int) {
	p.mu.Lock()
	p.val = v
	p.mu.Unlock()
}

// ---- Unit conversion ----------------------------------------------------

// PointsPerUnit maps unit strings to their scale factors (PDF points per unit).
var PointsPerUnit = map[string]float64{
	"pt": 1.0,
	"mm": 72.0 / 25.4,
	"cm": 72.0 / 2.54,
	"in": 72.0,
	"px": 72.0 / 96.0,
}

// Base holds the document-level state shared across all modules.
type Base struct {
	// Unit and scale
	Unit string
	kUnit float64 // points per user unit

	// Page height in points (for Y-coordinate flipping)
	PageHeight float64

	// RTL mode
	IsRTL bool

	// Default page content enabled
	defaultPageContent bool

	// Zero-width break points
	zeroWidthBreakPoints bool

	// Space detection regexp (stored as a compiled string pattern for goja)
	SpaceRegexp string

	// PON counter (shared)
	PON *PON

	// Default cell
	DefaultCell CellDef

	// Last rendered bounding box
	LastBBox BBox
}

// New creates a new Base with the given unit.
func New(unit string) (*Base, error) {
	k, ok := PointsPerUnit[unit]
	if !ok {
		return nil, &ErrUnknownUnit{Unit: unit}
	}
	return &Base{
		Unit:               unit,
		kUnit:              k,
		defaultPageContent: true,
		SpaceRegexp:        `/[^\S\xa0]/`,
		PON:                &PON{},
		DefaultCell:        ZeroCell,
	}, nil
}

// ToPoints converts a value from user units to PDF points.
func (b *Base) ToPoints(usr float64) float64 {
	return usr * b.kUnit
}

// ToUnit converts a value from PDF points to user units.
func (b *Base) ToUnit(pts float64) float64 {
	return pts / b.kUnit
}

// ToYPoints converts a vertical user value to PDF points (Y-flip).
// In PDF, Y increases upward; TCPDF uses top-down. This handles the flip.
func (b *Base) ToYPoints(usr float64) float64 {
	return b.PageHeight - usr*b.kUnit
}

// ToYUnit converts a vertical PDF points value to user units (Y-flip).
func (b *Base) ToYUnit(pts float64) float64 {
	return (b.PageHeight - pts) / b.kUnit
}

// SetRTL sets the default document direction.
func (b *Base) SetRTL(enabled bool) *Base {
	b.IsRTL = enabled
	return b
}

// EnableDefaultPageContent enables or disables the default page content callback.
func (b *Base) EnableDefaultPageContent(enable bool) {
	b.defaultPageContent = enable
}

// IsDefaultPageContentEnabled returns whether default page content is enabled.
func (b *Base) IsDefaultPageContentEnabled() bool {
	return b.defaultPageContent
}

// EnableZeroWidthBreakPoints enables or disables zero-width break-points.
func (b *Base) EnableZeroWidthBreakPoints(enabled bool) {
	b.zeroWidthBreakPoints = enabled
}

// IsZeroWidthBreakPointsEnabled returns whether zero-width break points are on.
func (b *Base) IsZeroWidthBreakPointsEnabled() bool {
	return b.zeroWidthBreakPoints
}

// SetSpaceRegexp sets the whitespace detection regular expression.
func (b *Base) SetSpaceRegexp(re string) {
	if re == "" {
		b.SpaceRegexp = `/[^\S\xa0]/`
		return
	}
	b.SpaceRegexp = re
}

// ---- Errors -------------------------------------------------------------

// ErrUnknownUnit is returned for an unknown measurement unit.
type ErrUnknownUnit struct {
	Unit string
}

func (e *ErrUnknownUnit) Error() string {
	return "base: unknown unit: " + e.Unit
}
