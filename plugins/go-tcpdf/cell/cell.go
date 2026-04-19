// Package cell provides cell geometry management (margin, padding, border position).
// Ported from tc-lib-pdf Cell.php by Nicola Asuni.
package cell

import "github.com/tecnickcom/go-tcpdf/base"

// Cell manages the default cell parameters.
type Cell struct {
	def base.CellDef
}

// New creates a Cell with defaults.
func New() *Cell {
	return &Cell{def: base.ZeroCell}
}

// SetDefaultCellMargin sets the default cell margin in user units (TRBL).
func (c *Cell) SetDefaultCellMargin(top, right, bottom, left float64) {
	c.def.Margin = base.CellBound{T: top, R: right, B: bottom, L: left}
}

// SetDefaultCellPadding sets the default cell padding in user units (TRBL).
func (c *Cell) SetDefaultCellPadding(top, right, bottom, left float64) {
	c.def.Padding = base.CellBound{T: top, R: right, B: bottom, L: left}
}

// SetDefaultCellBorderPos sets the default border position.
// Use base.BorderPosDefault, base.BorderPosExternal, or base.BorderPosInternal.
func (c *Cell) SetDefaultCellBorderPos(pos float64) {
	c.def.BorderPos = pos
}

// GetDefaultCell returns the current default CellDef.
func (c *Cell) GetDefaultCell() base.CellDef {
	return c.def
}

// Resolve returns the effective CellDef by merging override (may be nil) onto defaults.
func (c *Cell) Resolve(override *base.CellDef) base.CellDef {
	if override == nil {
		return c.def
	}
	return *override
}
