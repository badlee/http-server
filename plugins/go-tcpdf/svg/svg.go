// Package svg provides SVG-to-PDF rendering.
// Ported from tc-lib-pdf SVG.php by Nicola Asuni.
package svg

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"

	"github.com/tecnickcom/go-tcpdf/color"
	"github.com/tecnickcom/go-tcpdf/graph"
	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

// SVGObject holds a parsed and converted SVG ready for PDF embedding.
type SVGObject struct {
	ObjNum int
	Width  float64 // in user units
	Height float64
	PDF    string  // PDF stream content
}

// SVG manages SVG conversion.
type SVG struct {
	objects []*SVGObject
	counter int
	kUnit   float64
}

// New creates an SVG converter.
func New(kUnit float64) *SVG {
	return &SVG{kUnit: kUnit}
}

// AddSVG parses SVG data and returns a new SVGObject ID.
// img can be raw SVG XML or a file path (detection via '<svg' prefix).
func (s *SVG) AddSVG(img string, posX, posY, width, height, pageHeight float64) (int, error) {
	pdf, svgW, svgH, err := convertSVGToPDF(img, s.kUnit, pageHeight)
	if err != nil {
		return -1, err
	}
	// Scale to requested size
	if width == 0 && height == 0 {
		width = svgW
		height = svgH
	} else if width == 0 {
		width = svgW * height / svgH
	} else if height == 0 {
		height = svgH * width / svgW
	}

	s.counter++
	obj := &SVGObject{
		ObjNum: s.counter, // real obj num assigned by output layer
		Width:  width,
		Height: height,
		PDF:    pdf,
	}
	s.objects = append(s.objects, obj)
	return s.counter, nil
}

// GetSetSVG returns the PDF operators to place an SVG object.
func (s *SVG) GetSetSVG(soid int) string {
	for _, o := range s.objects {
		if o.ObjNum == soid {
			var _sb strings.Builder; _sb.WriteString("/XObjSVG"); _sb.WriteString(pdfbuf.FmtI(soid)); _sb.WriteString(" Do\n"); return _sb.String()
		}
	}
	return ""
}

// All returns all SVG objects.
func (s *SVG) All() []*SVGObject { return s.objects }

// ---- SVG → PDF conversion -----------------------------------------------

// svgElem is the root SVG element.
type svgElem struct {
	XMLName  xml.Name   `xml:"svg"`
	ViewBox  string     `xml:"viewBox,attr"`
	Width    string     `xml:"width,attr"`
	Height   string     `xml:"height,attr"`
	Elements []xml.Token
}

// convertSVGToPDF converts raw SVG XML to a PDF stream.
// Returns (pdf stream, width in user units, height in user units, error).
func convertSVGToPDF(svgData string, kUnit, pageHeight float64) (string, float64, float64, error) {
	dec := xml.NewDecoder(strings.NewReader(svgData))
	var (
		sb          strings.Builder
		svgWidth    float64 = 100
		svgHeight   float64 = 100
		viewBox     [4]float64
		hasViewBox  bool
		currentFill = "black"
		currentStroke = "none"
		currentStrokeWidth = 1.0
		transformStack []string
	)

	parseSVGLength := func(s string) float64 {
		s = strings.TrimSpace(s)
		if strings.HasSuffix(s, "mm") {
			v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "mm"), 64)
			return v * 72 / 25.4 / kUnit
		}
		if strings.HasSuffix(s, "cm") {
			v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "cm"), 64)
			return v * 72 / 2.54 / kUnit
		}
		if strings.HasSuffix(s, "pt") {
			v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "pt"), 64)
			return v / kUnit
		}
		if strings.HasSuffix(s, "px") {
			v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 64)
			return v * 72 / 96 / kUnit
		}
		v, _ := strconv.ParseFloat(s, 64)
		return v / kUnit
	}

	getAttr := func(attrs []xml.Attr, name string) string {
		for _, a := range attrs {
			if a.Name.Local == name {
				return a.Value
			}
		}
		return ""
	}

	applyStyle := func(attrs []xml.Attr) string {
		fill := getAttr(attrs, "fill")
		if fill == "" {
			fill = currentFill
		}
		stroke := getAttr(attrs, "stroke")
		if stroke == "" {
			stroke = currentStroke
		}
		strokeW := getAttr(attrs, "stroke-width")
		sw := currentStrokeWidth
		if strokeW != "" {
			sw, _ = strconv.ParseFloat(strokeW, 64)
		}
		var ops strings.Builder
		// Fill color
		if fill != "none" && fill != "" {
			c, err := color.ParseCSS(fill)
			if err == nil {
				ops.WriteString(c.FillOperator() + "\n")
			}
		}
		// Stroke color
		if stroke != "none" && stroke != "" {
			c, err := color.ParseCSS(stroke)
			if err == nil {
				ops.WriteString(c.StrokeOperator() + "\n")
				ops.WriteString(pdfbuf.FmtF(sw)); ops.WriteString(" w\n")
			}
		}
		paintStyle := "n"
		hasFill := fill != "none" && fill != ""
		hasStroke := stroke != "none" && stroke != ""
		if hasFill && hasStroke {
			paintStyle = "B"
		} else if hasFill {
			paintStyle = "f"
		} else if hasStroke {
			paintStyle = "S"
		}
		return ops.String() + paintStyle
	}
	_ = applyStyle

	d := graph.NewDraw()
	_ = d
	_ = transformStack

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "svg":
				wStr := getAttr(t.Attr, "width")
				hStr := getAttr(t.Attr, "height")
				vbStr := getAttr(t.Attr, "viewBox")
				if wStr != "" {
					svgWidth = parseSVGLength(wStr)
				}
				if hStr != "" {
					svgHeight = parseSVGLength(hStr)
				}
				if vbStr != "" {
					parts := strings.Fields(strings.ReplaceAll(vbStr, ",", " "))
					if len(parts) == 4 {
						for i, p := range parts {
							viewBox[i], _ = strconv.ParseFloat(p, 64)
						}
						hasViewBox = true
					}
				}
				_ = hasViewBox
				_ = viewBox
				sb.WriteString("q\n") // save state

			case "rect":
				x := parseSVGLength(getAttr(t.Attr, "x"))
				y := parseSVGLength(getAttr(t.Attr, "y"))
				w := parseSVGLength(getAttr(t.Attr, "width"))
				h := parseSVGLength(getAttr(t.Attr, "height"))
				fill := getAttr(t.Attr, "fill")
				stroke := getAttr(t.Attr, "stroke")
				paintStyle := svgPaintStyle(fill, stroke)
				if fill != "none" && fill != "" {
					if c, e := color.ParseCSS(fill); e == nil {
						sb.WriteString(c.FillOperator() + "\n")
					}
				}
				if stroke != "none" && stroke != "" {
					if c, e := color.ParseCSS(stroke); e == nil {
						sb.WriteString(c.StrokeOperator() + "\n")
						if sw := getAttr(t.Attr, "stroke-width"); sw != "" {
							if swv, e2 := strconv.ParseFloat(sw, 64); e2 == nil {
								sb.WriteString(fmtF(swv/kUnit) + " w\n")
							}
						}
					}
				}
					sb.WriteString(pdfbuf.FmtF(x)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(w)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(h)); sb.WriteString(" re ")
					sb.WriteString(paintStyle); sb.WriteByte('\n')

			case "circle":
				cx := parseSVGLength(getAttr(t.Attr, "cx"))
				cy := parseSVGLength(getAttr(t.Attr, "cy"))
				r := parseSVGLength(getAttr(t.Attr, "r"))
				fill := getAttr(t.Attr, "fill")
				stroke := getAttr(t.Attr, "stroke")
				paintStyle := svgPaintStyle(fill, stroke)
				if fill != "none" && fill != "" {
					if c, e := color.ParseCSS(fill); e == nil {
						sb.WriteString(c.FillOperator() + "\n")
					}
				}
				if stroke != "none" && stroke != "" {
					if c, e := color.ParseCSS(stroke); e == nil {
						sb.WriteString(c.StrokeOperator() + "\n")
					}
				}
				sb.WriteString(svgCirclePDF(cx, cy, r, paintStyle))

			case "ellipse":
				cx := parseSVGLength(getAttr(t.Attr, "cx"))
				cy := parseSVGLength(getAttr(t.Attr, "cy"))
				rx := parseSVGLength(getAttr(t.Attr, "rx"))
				ry := parseSVGLength(getAttr(t.Attr, "ry"))
				fill := getAttr(t.Attr, "fill")
				paintStyle := svgPaintStyle(fill, getAttr(t.Attr, "stroke"))
				if fill != "none" && fill != "" {
					if c, e := color.ParseCSS(fill); e == nil {
						sb.WriteString(c.FillOperator() + "\n")
					}
				}
				sb.WriteString(svgEllipsePDF(cx, cy, rx, ry, paintStyle))

			case "line":
				x1 := parseSVGLength(getAttr(t.Attr, "x1"))
				y1 := parseSVGLength(getAttr(t.Attr, "y1"))
				x2 := parseSVGLength(getAttr(t.Attr, "x2"))
				y2 := parseSVGLength(getAttr(t.Attr, "y2"))
				if stroke := getAttr(t.Attr, "stroke"); stroke != "" && stroke != "none" {
					if c, e := color.ParseCSS(stroke); e == nil {
						sb.WriteString(c.StrokeOperator() + "\n")
					}
				}
					sb.WriteString(pdfbuf.FmtF(x1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y1)); sb.WriteString(" m ")
					sb.WriteString(pdfbuf.FmtF(x2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y2)); sb.WriteString(" l S\n")

			case "polyline", "polygon":
				pointsStr := getAttr(t.Attr, "points")
				pts := parsePoints(pointsStr, parseSVGLength)
				if len(pts) >= 2 {
					fill := getAttr(t.Attr, "fill")
					stroke := getAttr(t.Attr, "stroke")
					paintStyle := svgPaintStyle(fill, stroke)
					if fill != "none" && fill != "" {
						if c, e := color.ParseCSS(fill); e == nil {
							sb.WriteString(c.FillOperator() + "\n")
						}
					}
					if stroke != "none" && stroke != "" {
						if c, e := color.ParseCSS(stroke); e == nil {
							sb.WriteString(c.StrokeOperator() + "\n")
						}
					}
					sb.WriteString(pdfbuf.FmtF(pts[0][0])); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(pts[0][1])); sb.WriteString(" m\n")
					for _, p := range pts[1:] {
						sb.WriteString(pdfbuf.FmtF(p[0])); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(p[1])); sb.WriteString(" l\n")
					}
					if t.Name.Local == "polygon" {
						sb.WriteString("h\n")
					}
					sb.WriteString(paintStyle + "\n")
				}

			case "path":
				pathD := getAttr(t.Attr, "d")
				fill := getAttr(t.Attr, "fill")
				stroke := getAttr(t.Attr, "stroke")
				paintStyle := svgPaintStyle(fill, stroke)
				if fill != "none" && fill != "" {
					if c, e := color.ParseCSS(fill); e == nil {
						sb.WriteString(c.FillOperator() + "\n")
					}
				}
				if stroke != "none" && stroke != "" {
					if c, e := color.ParseCSS(stroke); e == nil {
						sb.WriteString(c.StrokeOperator() + "\n")
					}
				}
				sb.WriteString(svgPathToPDF(pathD, parseSVGLength))
				sb.WriteString(paintStyle + "\n")

			case "g":
				sb.WriteString("q\n")
				if transform := getAttr(t.Attr, "transform"); transform != "" {
					sb.WriteString(svgTransformToPDF(transform, kUnit))
				}
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "svg":
				sb.WriteString("Q\n")
			case "g":
				sb.WriteString("Q\n")
			}
		}
	}

	return sb.String(), svgWidth, svgHeight, nil
}

// ---- SVG path converter -------------------------------------------------

func svgPathToPDF(d string, scaleLen func(string) float64) string {
	var sb strings.Builder
	tokens := tokenizeSVGPath(d)
	if len(tokens) == 0 {
		return ""
	}
	i := 0
	var cx, cy, lx, ly float64 // current point, last control point

	nextFloat := func() float64 {
		if i < len(tokens) {
			v, _ := strconv.ParseFloat(tokens[i], 64)
			i++
			return v / 1 // SVG units, no scale in path data
		}
		return 0
	}

	for i < len(tokens) {
		cmd := tokens[i]
		i++
		switch cmd {
		case "M":
			cx, cy = nextFloat(), nextFloat()
			lx, ly = cx, cy
			sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" m\n")
		case "m":
			dx, dy := nextFloat(), nextFloat()
			cx, cy = cx+dx, cy+dy
			lx, ly = cx, cy
			sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" m\n")
		case "L":
			for i < len(tokens) && !isSVGCmd(tokens[i]) {
				cx, cy = nextFloat(), nextFloat()
				lx, ly = cx, cy
				sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" l\n")
			}
		case "l":
			for i < len(tokens) && !isSVGCmd(tokens[i]) {
				dx, dy := nextFloat(), nextFloat()
				cx, cy = cx+dx, cy+dy
				lx, ly = cx, cy
				sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" l\n")
			}
		case "H":
			for i < len(tokens) && !isSVGCmd(tokens[i]) {
				cx = nextFloat()
				lx, ly = cx, cy
				sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" l\n")
			}
		case "h":
			for i < len(tokens) && !isSVGCmd(tokens[i]) {
				dx := nextFloat()
				cx += dx
				lx, ly = cx, cy
				sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" l\n")
			}
		case "V":
			for i < len(tokens) && !isSVGCmd(tokens[i]) {
				cy = nextFloat()
				lx, ly = cx, cy
				sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" l\n")
			}
		case "v":
			for i < len(tokens) && !isSVGCmd(tokens[i]) {
				dy := nextFloat()
				cy += dy
				lx, ly = cx, cy
				sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" l\n")
			}
		case "C":
			for i+5 < len(tokens)+1 && !isSVGCmd(tokens[i]) {
				x1, y1 := nextFloat(), nextFloat()
				x2, y2 := nextFloat(), nextFloat()
				x, y := nextFloat(), nextFloat()
					sb.WriteString(pdfbuf.FmtF(x1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(x2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(x)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y)); sb.WriteString(" c\n")
				lx, ly = x2, y2
				cx, cy = x, y
			}
		case "c":
			for i < len(tokens) && !isSVGCmd(tokens[i]) {
				dx1, dy1 := nextFloat(), nextFloat()
				dx2, dy2 := nextFloat(), nextFloat()
				dx, dy := nextFloat(), nextFloat()
				x1, y1 := cx+dx1, cy+dy1
				x2, y2 := cx+dx2, cy+dy2
				x, y := cx+dx, cy+dy
					sb.WriteString(pdfbuf.FmtF(x1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(x2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(x)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y)); sb.WriteString(" c\n")
				lx, ly = x2, y2
				cx, cy = x, y
			}
		case "S":
			for i+3 < len(tokens)+1 && !isSVGCmd(tokens[i]) {
				// Reflect last control point
				x1, y1 := 2*cx-lx, 2*cy-ly
				x2, y2 := nextFloat(), nextFloat()
				x, y := nextFloat(), nextFloat()
					sb.WriteString(pdfbuf.FmtF(x1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(x2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(x)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y)); sb.WriteString(" c\n")
				lx, ly = x2, y2
				cx, cy = x, y
			}
		case "Q":
			for i+3 < len(tokens)+1 && !isSVGCmd(tokens[i]) {
				qx, qy := nextFloat(), nextFloat()
				x, y := nextFloat(), nextFloat()
				// Convert quadratic to cubic
				x1 := cx + 2.0/3.0*(qx-cx)
				y1 := cy + 2.0/3.0*(qy-cy)
				x2 := x + 2.0/3.0*(qx-x)
				y2 := y + 2.0/3.0*(qy-y)
					sb.WriteString(pdfbuf.FmtF(x1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y1)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(x2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y2)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(x)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(y)); sb.WriteString(" c\n")
				lx, ly = qx, qy
				cx, cy = x, y
			}
		case "Z", "z":
			sb.WriteString("h\n")
			cx, cy = lx, ly
		}
	}
	_ = lx
	_ = ly
	return sb.String()
}

func tokenizeSVGPath(d string) []string {
	var tokens []string
	var cur strings.Builder
	for _, r := range d + " " {
		if isSVGCmdRune(r) {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			tokens = append(tokens, string(r))
		} else if r == ',' || r == ' ' || r == '\t' || r == '\n' {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		} else {
			cur.WriteRune(r)
		}
	}
	return tokens
}

func isSVGCmdRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isSVGCmd(s string) bool {
	return len(s) == 1 && isSVGCmdRune(rune(s[0]))
}

func parsePoints(pts string, scaleLen func(string) float64) [][2]float64 {
	pts = strings.ReplaceAll(pts, ",", " ")
	parts := strings.Fields(pts)
	var result [][2]float64
	for i := 0; i+1 < len(parts); i += 2 {
		x, _ := strconv.ParseFloat(parts[i], 64)
		y, _ := strconv.ParseFloat(parts[i+1], 64)
		result = append(result, [2]float64{x, y})
	}
	return result
}

func svgPaintStyle(fill, stroke string) string {
	hasFill := fill != "none" && fill != ""
	hasStroke := stroke != "none" && stroke != ""
	if hasFill && hasStroke {
		return "B"
	} else if hasStroke {
		return "S"
	} else if hasFill {
		return "f"
	}
	return "n"
}

func svgCirclePDF(cx, cy, r float64, paintStyle string) string {
	return svgEllipsePDF(cx, cy, r, r, paintStyle)
}

func svgEllipsePDF(cx, cy, rx, ry float64, paintStyle string) string {
	k := 0.5522847498
	var sb strings.Builder
	sb.WriteString(pdfbuf.FmtF(cx+rx)); sb.WriteByte(' '); sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" m\n")
					sb.WriteString(pdfbuf.FmtF(cx+rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy+k*ry)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cx+k*rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy+ry)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy+ry)); sb.WriteString(" c\n")
					sb.WriteString(pdfbuf.FmtF(cx-k*rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy+ry)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cx-rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy+k*ry)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cx-rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" c\n")
					sb.WriteString(pdfbuf.FmtF(cx-rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy-k*ry)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cx-k*rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy-ry)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy-ry)); sb.WriteString(" c\n")
					sb.WriteString(pdfbuf.FmtF(cx+k*rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy-ry)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cx+rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy-k*ry)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cx+rx)); sb.WriteByte(' ')
					sb.WriteString(pdfbuf.FmtF(cy)); sb.WriteString(" c\n")
	sb.WriteString(paintStyle + "\n")
	return sb.String()
}

func svgTransformToPDF(transform string, kUnit float64) string {
	transform = strings.TrimSpace(transform)
	if strings.HasPrefix(transform, "matrix(") {
		inner := strings.TrimSuffix(strings.TrimPrefix(transform, "matrix("), ")")
		parts := strings.Fields(strings.ReplaceAll(inner, ",", " "))
		if len(parts) == 6 {
			vals := make([]float64, 6)
			for i, p := range parts {
				vals[i], _ = strconv.ParseFloat(p, 64)
			}
			var tb strings.Builder
			tb.WriteString(pdfbuf.FmtF(vals[0])); tb.WriteByte(' ')
			tb.WriteString(pdfbuf.FmtF(vals[1])); tb.WriteByte(' ')
			tb.WriteString(pdfbuf.FmtF(vals[2])); tb.WriteByte(' ')
			tb.WriteString(pdfbuf.FmtF(vals[3])); tb.WriteByte(' ')
			tb.WriteString(pdfbuf.FmtF(vals[4]/kUnit)); tb.WriteByte(' ')
			tb.WriteString(pdfbuf.FmtF(vals[5]/kUnit)); tb.WriteString(" cm\n")
			return tb.String()
		}
	}
	if strings.HasPrefix(transform, "translate(") {
		inner := strings.TrimSuffix(strings.TrimPrefix(transform, "translate("), ")")
		parts := strings.Fields(strings.ReplaceAll(inner, ",", " "))
		if len(parts) >= 1 {
			tx, _ := strconv.ParseFloat(parts[0], 64)
			ty := 0.0
			if len(parts) >= 2 {
				ty, _ = strconv.ParseFloat(parts[1], 64)
			}
			var tb strings.Builder
			tb.WriteString("1 0 0 1 ")
			tb.WriteString(pdfbuf.FmtF(tx/kUnit)); tb.WriteByte(' ')
			tb.WriteString(pdfbuf.FmtF(ty/kUnit)); tb.WriteString(" cm\n")
			return tb.String()
		}
	}
	if strings.HasPrefix(transform, "scale(") {
		inner := strings.TrimSuffix(strings.TrimPrefix(transform, "scale("), ")")
		parts := strings.Fields(strings.ReplaceAll(inner, ",", " "))
		sx, _ := strconv.ParseFloat(parts[0], 64)
		sy := sx
		if len(parts) >= 2 {
			sy, _ = strconv.ParseFloat(parts[1], 64)
		}
		var tb strings.Builder
		tb.WriteString(pdfbuf.FmtF(sx)); tb.WriteString(" 0 0 ")
		tb.WriteString(pdfbuf.FmtF(sy)); tb.WriteString(" 0 0 cm\n")
		return tb.String()
	}
	if strings.HasPrefix(transform, "rotate(") {
		inner := strings.TrimSuffix(strings.TrimPrefix(transform, "rotate("), ")")
		parts := strings.Fields(strings.ReplaceAll(inner, ",", " "))
		angle, _ := strconv.ParseFloat(parts[0], 64)
		rad := angle * math.Pi / 180
		c := math.Cos(rad)
		s := math.Sin(rad)
		var tb strings.Builder
		tb.WriteString(pdfbuf.FmtF(c)); tb.WriteByte(' ')
		tb.WriteString(pdfbuf.FmtF(s)); tb.WriteByte(' ')
		tb.WriteString(pdfbuf.FmtF(-s)); tb.WriteByte(' ')
		tb.WriteString(pdfbuf.FmtF(c)); tb.WriteString(" 0 0 cm\n")
		return tb.String()
	}
	return ""
}


func fmtF(v float64) string { return pdfbuf.FmtF(v) }
