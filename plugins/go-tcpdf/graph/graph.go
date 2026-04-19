// Package graph provides PDF graphics primitives: paths, styles, transformations.
// Ported from tc-lib-pdf-graph (PHP) by Nicola Asuni.
package graph

import (
	"fmt"
	"math"
	"strings"

	"github.com/tecnickcom/go-tcpdf/color"
	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

// LineCap constants (PDF §8.4.3.3)
type LineCap int
const (
	LineCapButt   LineCap = 0
	LineCapRound  LineCap = 1
	LineCapSquare LineCap = 2
)

// LineJoin constants (PDF §8.4.3.4)
type LineJoin int
const (
	LineJoinMiter LineCap = 0
	LineJoinRound LineCap = 1
	LineJoinBevel LineCap = 2
)

// StyleData holds all graphic state parameters for a draw operation.
type StyleData struct {
	LineWidth   float64
	LineCap     int
	LineJoin    int
	MiterLimit  float64
	DashArray   []float64
	DashPhase   float64
	LineColor   color.Color
	FillColor   color.Color
	FillRule    string // "nonzero" or "evenodd"
}

// DefaultStyle returns a default style (black line, no fill).
func DefaultStyle() StyleData {
	return StyleData{
		LineWidth:  0.2,
		LineCap:    0,
		LineJoin:   0,
		MiterLimit: 10,
		LineColor:  color.Black,
		FillColor:  color.Transparent,
		FillRule:   "nonzero",
	}
}

// BorderStyle holds a style for a specific cell border side.
type BorderStyle struct {
	Width float64
	Type  string // "S"=solid, "D"=dashed, "B"=beveled, "I"=inset, "U"=underline
	Color color.Color
}

// ---- CTM (Current Transformation Matrix) --------------------------------

// Matrix is a 2D affine transformation matrix.
//   | a  b  0 |
//   | c  d  0 |
//   | e  f  1 |
type Matrix struct {
	A, B, C, D, E, F float64
}

// Identity returns the identity matrix.
func Identity() Matrix {
	return Matrix{A: 1, B: 0, C: 0, D: 1, E: 0, F: 0}
}

// Translate returns a translation matrix.
func Translate(tx, ty float64) Matrix {
	return Matrix{A: 1, B: 0, C: 0, D: 1, E: tx, F: ty}
}

// Scale returns a scaling matrix.
func Scale(sx, sy float64) Matrix {
	return Matrix{A: sx, B: 0, C: 0, D: sy, E: 0, F: 0}
}

// Rotate returns a rotation matrix (angle in radians).
func Rotate(angle float64) Matrix {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return Matrix{A: c, B: s, C: -s, D: c, E: 0, F: 0}
}

// Multiply multiplies two matrices (m × n).
func (m Matrix) Multiply(n Matrix) Matrix {
	return Matrix{
		A: m.A*n.A + m.B*n.C,
		B: m.A*n.B + m.B*n.D,
		C: m.C*n.A + m.D*n.C,
		D: m.C*n.B + m.D*n.D,
		E: m.E*n.A + m.F*n.C + n.E,
		F: m.E*n.B + m.F*n.D + n.F,
	}
}

// PDF returns the PDF cm operator string: "a b c d e f cm".
func (m Matrix) PDF() string {
	var b pdfbuf.Buf
	b.F(m.A); b.SP(); b.F(m.B); b.SP(); b.F(m.C); b.SP()
	b.F(m.D); b.SP(); b.F(m.E); b.SP(); b.F(m.F); b.S(" cm")
	return b.String()
}

// ---- Graphics state stack -----------------------------------------------

// GState holds a saved graphics state.
type GState struct {
	CTM         Matrix
	Style       StyleData
	TextState   TextState
}

// TextState holds text-specific graphics state.
type TextState struct {
	CharSpacing float64
	WordSpacing float64
	HScale      float64 // horizontal scaling in percent (100 = normal)
	Leading     float64
	Rise        float64
	RenderMode  int
}

// GStateStack manages save/restore of graphics state.
type GStateStack struct {
	stack []GState
}

func NewGStateStack() *GStateStack { return &GStateStack{} }

func (g *GStateStack) Push(s GState) { g.stack = append(g.stack, s) }
func (g *GStateStack) Pop() (GState, bool) {
	if len(g.stack) == 0 {
		return GState{}, false
	}
	top := g.stack[len(g.stack)-1]
	g.stack = g.stack[:len(g.stack)-1]
	return top, true
}
func (g *GStateStack) Depth() int { return len(g.stack) }

// ---- Draw object (stream builder) ---------------------------------------

// Draw builds PDF graphics stream content.
type Draw struct {
	buf   strings.Builder
	style StyleData
	ctm   Matrix
	stack *GStateStack
	// y-flip factor for coordinate conversion (set by output layer)
	PageHeight float64 // current page height in points (for Y-flip)
}

// NewDraw creates a new Draw context.
func NewDraw() *Draw {
	return &Draw{
		style: DefaultStyle(),
		ctm:   Identity(),
		stack: NewGStateStack(),
	}
}

// Reset clears the accumulated stream content.
func (d *Draw) Reset() { d.buf.Reset() }

// String returns the accumulated PDF stream content.
func (d *Draw) String() string { return d.buf.String() }

// ---- State operators ----------------------------------------------------

func (d *Draw) SaveState() string {
	d.stack.Push(GState{CTM: d.ctm, Style: d.style})
	return "q\n"
}

func (d *Draw) RestoreState() (string, error) {
	s, ok := d.stack.Pop()
	if !ok {
		return "", fmt.Errorf("graph: graphics state stack underflow")
	}
	d.ctm = s.CTM
	d.style = s.Style
	return "Q\n", nil
}

func (d *Draw) SetCTM(m Matrix) string {
	d.ctm = d.ctm.Multiply(m)
	return m.PDF() + "\n"
}

// ---- Appearance operators -----------------------------------------------

func (d *Draw) SetLineWidth(w float64) string {
	d.style.LineWidth = w
	return func() string { var b pdfbuf.Buf; b.F(w); b.S(" w\n"); return b.String() }()
}

func (d *Draw) SetLineCap(cap int) string {
	d.style.LineCap = cap
	return func() string { var b pdfbuf.Buf; b.I(cap); b.S(" J\n"); return b.String() }()
}

func (d *Draw) SetLineJoin(join int) string {
	d.style.LineJoin = join
	return func() string { var b pdfbuf.Buf; b.I(join); b.S(" j\n"); return b.String() }()
}

func (d *Draw) SetMiterLimit(limit float64) string {
	d.style.MiterLimit = limit
	return func() string { var b pdfbuf.Buf; b.F(limit); b.S(" M\n"); return b.String() }()
}

func (d *Draw) SetDash(array []float64, phase float64) string {
	d.style.DashArray = array
	d.style.DashPhase = phase
	if len(array) == 0 {
		return "[] 0 d\n"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range array {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmtF(v))
	}
	sb.WriteString("] "); sb.WriteString(fmtF(phase)); sb.WriteString(" d\n")
	return sb.String()
}

func (d *Draw) SetFillColor(c color.Color) string {
	d.style.FillColor = c
	return c.FillOperator() + "\n"
}

func (d *Draw) SetStrokeColor(c color.Color) string {
	d.style.LineColor = c
	return c.StrokeOperator() + "\n"
}

// ApplyStyle emits all style operators for a StyleData.
func (d *Draw) ApplyStyle(s StyleData) string {
	var sb strings.Builder
	sb.WriteString(d.SetLineWidth(s.LineWidth))
	sb.WriteString(d.SetLineCap(s.LineCap))
	sb.WriteString(d.SetLineJoin(s.LineJoin))
	sb.WriteString(d.SetMiterLimit(s.MiterLimit))
	sb.WriteString(d.SetDash(s.DashArray, s.DashPhase))
	sb.WriteString(d.SetStrokeColor(s.LineColor))
	if s.FillColor.Type != color.ColorTypeNone {
		sb.WriteString(d.SetFillColor(s.FillColor))
	}
	return sb.String()
}

// ---- Path operators -----------------------------------------------------

func (d *Draw) MoveTo(x, y float64) string {
	return func() string { var b pdfbuf.Buf; b.F(x); b.SP(); b.F(y); b.S(" m\n"); return b.String() }()
}

func (d *Draw) LineTo(x, y float64) string {
	return func() string { var b pdfbuf.Buf; b.F(x); b.SP(); b.F(y); b.S(" l\n"); return b.String() }()
}

func (d *Draw) CurveTo(x1, y1, x2, y2, x3, y3 float64) string {
	return func() string { var b pdfbuf.Buf; b.F(x1); b.SP(); b.F(y1); b.SP(); b.F(x2); b.SP(); b.F(y2); b.SP(); b.F(x3); b.SP(); b.F(y3); b.S(" c\n"); return b.String() }()
}

func (d *Draw) CurveToV(x2, y2, x3, y3 float64) string {
	return func() string { var b pdfbuf.Buf; b.F(x2); b.SP(); b.F(y2); b.SP(); b.F(x3); b.SP(); b.F(y3); b.S(" v\n"); return b.String() }()
}

func (d *Draw) CurveToY(x1, y1, x3, y3 float64) string {
	return func() string { var b pdfbuf.Buf; b.F(x1); b.SP(); b.F(y1); b.SP(); b.F(x3); b.SP(); b.F(y3); b.S(" y\n"); return b.String() }()
}

func (d *Draw) ClosePath() string { return "h\n" }

func (d *Draw) Stroke() string { return "S\n" }
func (d *Draw) CloseStroke() string { return "s\n" }
func (d *Draw) Fill() string { return "f\n" }
func (d *Draw) FillEvenOdd() string { return "f*\n" }
func (d *Draw) FillAndStroke() string { return "B\n" }
func (d *Draw) FillEvenOddAndStroke() string { return "B*\n" }
func (d *Draw) EndPath() string { return "n\n" }

// ---- High-level shapes --------------------------------------------------

// Rect draws a rectangle at (x, y) with width w and height h.
// paintStyle: "S"=stroke only, "F"=fill only, "FD" or "DF"=fill+stroke, "f*"=fill evenodd
func (d *Draw) Rect(x, y, w, h float64, paintStyle string) string {
	var b pdfbuf.Buf
	b.F(x); b.SP(); b.F(y); b.SP(); b.F(w); b.SP(); b.F(h); b.S(" re\n")
	b.S(d.paintOp(paintStyle))
	return b.String()
}

// Line draws a straight line from (x1,y1) to (x2,y2).
func (d *Draw) Line(x1, y1, x2, y2 float64) string {
	return d.MoveTo(x1, y1) + d.LineTo(x2, y2) + d.Stroke()
}

// Circle draws a circle centred at (cx, cy) with radius r.
func (d *Draw) Circle(cx, cy, r float64, paintStyle string) string {
	return d.Ellipse(cx, cy, r, r, 0, paintStyle)
}

// Ellipse draws an ellipse. rx and ry are horizontal and vertical radii.
// angle is rotation in degrees.
func (d *Draw) Ellipse(cx, cy, rx, ry, angle float64, paintStyle string) string {
	k := 0.5522847498 // 4/3 * tan(pi/8) — Bézier approximation
	var sb strings.Builder

	if angle != 0 {
		rad := angle * math.Pi / 180
		sb.WriteString("q\n")
		m := Rotate(-rad)
		m.E = cx
		m.F = cy
		sb.WriteString(m.PDF() + "\n")
		cx, cy = 0, 0
	}

	sb.WriteString(d.MoveTo(cx+rx, cy))
	sb.WriteString(d.CurveTo(cx+rx, cy+k*ry, cx+k*rx, cy+ry, cx, cy+ry))
	sb.WriteString(d.CurveTo(cx-k*rx, cy+ry, cx-rx, cy+k*ry, cx-rx, cy))
	sb.WriteString(d.CurveTo(cx-rx, cy-k*ry, cx-k*rx, cy-ry, cx, cy-ry))
	sb.WriteString(d.CurveTo(cx+k*rx, cy-ry, cx+rx, cy-k*ry, cx+rx, cy))
	sb.WriteString(d.paintOp(paintStyle))

	if angle != 0 {
		sb.WriteString("Q\n")
	}
	return sb.String()
}

// RoundedRect draws a rectangle with rounded corners.
func (d *Draw) RoundedRect(x, y, w, h, rx, ry float64, paintStyle string) string {
	k := 0.5522847498
	var sb strings.Builder

	sb.WriteString(d.MoveTo(x+rx, y))
	sb.WriteString(d.LineTo(x+w-rx, y))
	sb.WriteString(d.CurveTo(x+w-rx+k*rx, y, x+w, y+ry-k*ry, x+w, y+ry))
	sb.WriteString(d.LineTo(x+w, y+h-ry))
	sb.WriteString(d.CurveTo(x+w, y+h-ry+k*ry, x+w-rx+k*rx, y+h, x+w-rx, y+h))
	sb.WriteString(d.LineTo(x+rx, y+h))
	sb.WriteString(d.CurveTo(x+rx-k*rx, y+h, x, y+h-ry+k*ry, x, y+h-ry))
	sb.WriteString(d.LineTo(x, y+ry))
	sb.WriteString(d.CurveTo(x, y+ry-k*ry, x+rx-k*rx, y, x+rx, y))
	sb.WriteString(d.ClosePath())
	sb.WriteString(d.paintOp(paintStyle))
	return sb.String()
}

// Arrow draws an arrow from (x1,y1) to (x2,y2).
func (d *Draw) Arrow(x1, y1, x2, y2, width, headLength, headWidth float64) string {
	angle := math.Atan2(y2-y1, x2-x1)
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	length := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1))
	shaftEnd := length - headLength

	var sb strings.Builder
	sb.WriteString("q\n")
	m := Matrix{A: cos, B: sin, C: -sin, D: cos, E: x1, F: y1}
	sb.WriteString(m.PDF() + "\n")

	hw := width / 2
	hhw := headWidth / 2
	// Shaft
	sb.WriteString(d.MoveTo(0, -hw))
	sb.WriteString(d.LineTo(shaftEnd, -hw))
	sb.WriteString(d.LineTo(shaftEnd, -hhw))
	sb.WriteString(d.LineTo(length, 0))
	sb.WriteString(d.LineTo(shaftEnd, hhw))
	sb.WriteString(d.LineTo(shaftEnd, hw))
	sb.WriteString(d.LineTo(0, hw))
	sb.WriteString(d.ClosePath())
	sb.WriteString("B\n")
	sb.WriteString("Q\n")
	return sb.String()
}

// ---- Gradient support ---------------------------------------------------

// GradientType selects the gradient type.
type GradientType int
const (
	GradientLinear GradientType = 2
	GradientRadial GradientType = 3
)

// GradientStop is a color stop in a gradient.
type GradientStop struct {
	Offset float64 // [0, 1]
	Color  color.Color
}

// Gradient defines a PDF gradient.
type Gradient struct {
	Type    GradientType
	Stops   []GradientStop
	X1, Y1 float64 // start point (linear) or center (radial)
	X2, Y2 float64 // end point (linear) or focal (radial)
	R1, R2 float64 // radii (radial only)
	ObjNum int
}

// NewLinearGradient creates a linear gradient descriptor.
func NewLinearGradient(x1, y1, x2, y2 float64, stops []GradientStop) Gradient {
	return Gradient{Type: GradientLinear, X1: x1, Y1: y1, X2: x2, Y2: y2, Stops: stops}
}

// NewRadialGradient creates a radial gradient descriptor.
func NewRadialGradient(cx, cy, r1, fx, fy, r2 float64, stops []GradientStop) Gradient {
	return Gradient{Type: GradientRadial, X1: cx, Y1: cy, R1: r1, X2: fx, Y2: fy, R2: r2, Stops: stops}
}

// ---- ExtGState (transparency / blend modes) -----------------------------

// ExtGState holds extended graphics state parameters (PDF §8.4.5).
type ExtGState struct {
	CA   float64 // stroke alpha [0,1]
	Ca   float64 // fill alpha [0,1]
	BM   string  // blend mode, e.g. "Normal", "Multiply"
	ObjNum int
}

// ---- helpers ------------------------------------------------------------

func (d *Draw) paintOp(style string) string {
	switch strings.ToUpper(style) {
	case "S":
		return "S\n"
	case "F", "F1":
		return "f\n"
	case "F*":
		return "f*\n"
	case "DF", "FD", "FDS":
		return "B\n"
	case "DF*", "FD*", "F*D":
		return "B*\n"
	case "CNZ":
		return "W n\n"
	case "CEO":
		return "W* n\n"
	default:
		return "n\n"
	}
}

func fmtF(v float64) string { return pdfbuf.FmtF(v) }
