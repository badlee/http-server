// Package css provides CSS property management for PDF rendering.
// Ported from tc-lib-pdf CSS.php by Nicola Asuni.
package css

import (
	"strconv"
	"strings"

	"github.com/tecnickcom/go-tcpdf/base"
)

// BoxSpacing holds CSS spacing values (margin, padding, border-spacing).
type BoxSpacing struct {
	Top, Right, Bottom, Left float64
}

// CSS manages default CSS layout values.
type CSS struct {
	DefaultMargin        BoxSpacing
	DefaultPadding       BoxSpacing
	DefaultBorderSpacing BoxSpacing // CSS border-spacing (table cell spacing)
}

// New creates a CSS manager with zero defaults.
func New() *CSS {
	return &CSS{}
}

// SetDefaultCSSMargin sets the default CSS margin.
func (c *CSS) SetDefaultCSSMargin(top, right, bottom, left float64) {
	c.DefaultMargin = BoxSpacing{top, right, bottom, left}
}

// SetDefaultCSSPadding sets the default CSS padding.
func (c *CSS) SetDefaultCSSPadding(top, right, bottom, left float64) {
	c.DefaultPadding = BoxSpacing{top, right, bottom, left}
}

// SetDefaultCSSBorderSpacing sets the default CSS border-spacing (vertical, horizontal).
func (c *CSS) SetDefaultCSSBorderSpacing(vert, horiz float64) {
	c.DefaultBorderSpacing = BoxSpacing{vert, horiz, vert, horiz}
}

// ---- CSS value parsing --------------------------------------------------

// ParseBoxValues parses a CSS shorthand like "1px 2px 3px 4px" into TRBL.
// Supports 1, 2, 3, and 4 value syntaxes.
func ParseBoxValues(val string, unit float64) BoxSpacing {
	parts := strings.Fields(val)
	vals := make([]float64, len(parts))
	for i, p := range parts {
		vals[i] = parseLengthValue(p, unit)
	}
	switch len(vals) {
	case 1:
		return BoxSpacing{vals[0], vals[0], vals[0], vals[0]}
	case 2:
		return BoxSpacing{vals[0], vals[1], vals[0], vals[1]}
	case 3:
		return BoxSpacing{vals[0], vals[1], vals[2], vals[1]}
	case 4:
		return BoxSpacing{vals[0], vals[1], vals[2], vals[3]}
	}
	return BoxSpacing{}
}

// parseLengthValue converts a CSS length string to user units.
// unit is the document scale factor (points per user unit).
func parseLengthValue(s string, unit float64) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	switch {
	case s == "0" || s == "auto" || s == "none":
		return 0
	case strings.HasSuffix(s, "px"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 64)
		return v * (72.0 / 96.0) / unit // px → points → user units
	case strings.HasSuffix(s, "pt"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "pt"), 64)
		return v / unit
	case strings.HasSuffix(s, "mm"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "mm"), 64)
		return v * (72.0 / 25.4) / unit
	case strings.HasSuffix(s, "cm"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "cm"), 64)
		return v * (72.0 / 2.54) / unit
	case strings.HasSuffix(s, "in"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "in"), 64)
		return v * 72.0 / unit
	case strings.HasSuffix(s, "em"), strings.HasSuffix(s, "rem"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSuffix(s, "rem"), "em"), 64)
		return v // treat as user units (font size relative — caller resolves)
	case strings.HasSuffix(s, "%"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		return v / 100.0 // percentage — caller resolves against container
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v / unit
}

// ---- CSS property map ---------------------------------------------------

// ParseProperties parses a CSS inline style string into a property map.
// e.g. "color: red; font-size: 12pt" → {"color": "red", "font-size": "12pt"}
func ParseProperties(style string) map[string]string {
	props := make(map[string]string)
	for _, decl := range strings.Split(style, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		idx := strings.IndexByte(decl, ':')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(decl[:idx]))
		val := strings.TrimSpace(decl[idx+1:])
		props[key] = val
	}
	return props
}

// FontWeight returns the CSS font-weight as a boolean (true = bold).
func FontWeight(val string) bool {
	val = strings.ToLower(strings.TrimSpace(val))
	return val == "bold" || val == "bolder" || val == "700" || val == "800" || val == "900"
}

// FontStyle returns the CSS font-style as a boolean (true = italic/oblique).
func FontStyle(val string) bool {
	val = strings.ToLower(strings.TrimSpace(val))
	return val == "italic" || val == "oblique"
}

// TextDecoration parses CSS text-decoration property.
// Returns (underline, linethrough, overline).
func TextDecoration(val string) (underline, linethrough, overline bool) {
	val = strings.ToLower(val)
	underline = strings.Contains(val, "underline")
	linethrough = strings.Contains(val, "line-through")
	overline = strings.Contains(val, "overline")
	return
}

// TextAlign parses CSS text-align into TCPDF halign.
func TextAlign(val string) string {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "left":
		return "L"
	case "right":
		return "R"
	case "center":
		return "C"
	case "justify":
		return "J"
	}
	return ""
}

// VerticalAlign parses CSS vertical-align into TCPDF valign.
func VerticalAlign(val string) string {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "top":
		return "T"
	case "middle":
		return "C"
	case "bottom":
		return "B"
	}
	return "T"
}

// BorderStyle parses a CSS border shorthand like "1px solid #000".
// Returns (width, style, color).
func BorderStyle(val string) (float64, string, string) {
	parts := strings.Fields(val)
	var width float64
	style := "solid"
	clr := ""
	for _, p := range parts {
		p = strings.ToLower(p)
		switch {
		case p == "none" || p == "hidden":
			return 0, "none", ""
		case p == "solid" || p == "dashed" || p == "dotted" || p == "double":
			style = p
		case strings.HasPrefix(p, "#") || strings.HasPrefix(p, "rgb"):
			clr = p
		default:
			if w := parseLengthValue(p, 1.0); w > 0 {
				width = w
			}
		}
	}
	return width, style, clr
}

// ToCellBound converts a BoxSpacing to a base.CellBound.
func ToCellBound(bs BoxSpacing) base.CellBound {
	return base.CellBound{T: bs.Top, R: bs.Right, B: bs.Bottom, L: bs.Left}
}
