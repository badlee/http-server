// Package color provides PDF color management: RGB, CMYK, Gray, Spot, ICC.
package color

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

// ColorType identifies the color space.
type ColorType int

const (
	ColorTypeGray  ColorType = iota
	ColorTypeRGB
	ColorTypeCMYK
	ColorTypeSpot
	ColorTypeNone
)

// Color holds a PDF color value.
type Color struct {
	Type               ColorType
	R, G, B            float64
	C, M, Y, K         float64
	Gray               float64
	SpotName           string
	SpotTint           float64
	Alpha              float64
}

var (
	Black       = Color{Type: ColorTypeGray, Gray: 0, Alpha: 1}
	White       = Color{Type: ColorTypeGray, Gray: 1, Alpha: 1}
	Transparent = Color{Type: ColorTypeNone, Alpha: 0}
	Red         = Color{Type: ColorTypeRGB, R: 1, G: 0, B: 0, Alpha: 1}
	Green       = Color{Type: ColorTypeRGB, R: 0, G: 0.5, B: 0, Alpha: 1}
	Blue        = Color{Type: ColorTypeRGB, R: 0, G: 0, B: 1, Alpha: 1}
)

func NewGray(g float64) Color   { return Color{Type: ColorTypeGray, Gray: clamp(g), Alpha: 1} }
func NewRGB(r, g, b float64) Color {
	return Color{Type: ColorTypeRGB, R: clamp(r), G: clamp(g), B: clamp(b), Alpha: 1}
}
func NewRGB255(r, g, b int) Color { return NewRGB(float64(r)/255, float64(g)/255, float64(b)/255) }
func NewCMYK(c, m, y, k float64) Color {
	return Color{Type: ColorTypeCMYK, C: clamp(c), M: clamp(m), Y: clamp(y), K: clamp(k), Alpha: 1}
}
func NewCMYK100(c, m, y, k float64) Color { return NewCMYK(c/100, m/100, y/100, k/100) }
func NewSpot(name string, tint float64) Color {
	return Color{Type: ColorTypeSpot, SpotName: name, SpotTint: clamp(tint), Alpha: 1}
}
func (c Color) WithAlpha(alpha float64) Color { c.Alpha = clamp(alpha); return c }

// ---- PDF operators -------------------------------------------------------

func (c Color) FillOperator() string {
	var b pdfbuf.Buf
	switch c.Type {
	case ColorTypeGray:
		b.F(c.Gray); b.S(" g")
	case ColorTypeRGB:
		b.F(c.R); b.SP(); b.F(c.G); b.SP(); b.F(c.B); b.S(" rg")
	case ColorTypeCMYK:
		b.F(c.C); b.SP(); b.F(c.M); b.SP(); b.F(c.Y); b.SP(); b.F(c.K); b.S(" k")
	case ColorTypeSpot:
		b.S("/CS"); b.S(pdfName(c.SpotName)); b.S(" cs "); b.F(c.SpotTint); b.S(" scn")
	default:
		return ""
	}
	return b.String()
}

func (c Color) StrokeOperator() string {
	var b pdfbuf.Buf
	switch c.Type {
	case ColorTypeGray:
		b.F(c.Gray); b.S(" G")
	case ColorTypeRGB:
		b.F(c.R); b.SP(); b.F(c.G); b.SP(); b.F(c.B); b.S(" RG")
	case ColorTypeCMYK:
		b.F(c.C); b.SP(); b.F(c.M); b.SP(); b.F(c.Y); b.SP(); b.F(c.K); b.S(" K")
	case ColorTypeSpot:
		b.S("/CS"); b.S(pdfName(c.SpotName)); b.S(" CS "); b.F(c.SpotTint); b.S(" SCN")
	default:
		return ""
	}
	return b.String()
}

// ---- CSS parsing ---------------------------------------------------------

func ParseCSS(s string) (Color, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" || s == "transparent" {
		return Transparent, nil
	}
	if strings.HasPrefix(s, "#") {
		return parseHex(s)
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "rgba(") {
		return parseRGBA(s)
	}
	if strings.HasPrefix(low, "rgb(") {
		return parseRGB(s)
	}
	if strings.HasPrefix(low, "cmyk(") {
		return parseCMYK(s)
	}
	if strings.HasPrefix(low, "gray(") || strings.HasPrefix(low, "grey(") {
		return parseGray(s)
	}
	if col, ok := namedColors[low]; ok {
		return col, nil
	}
	return Color{}, errors.New("color: unknown CSS color: " + s)
}

func parseHex(s string) (Color, error) {
	h := strings.TrimPrefix(s, "#")
	switch len(h) {
	case 3:
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	case 6:
	default:
		return Color{}, errors.New("color: invalid hex: " + s)
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return Color{}, err
	}
	return NewRGB255(int(n>>16)&0xFF, int(n>>8)&0xFF, int(n)&0xFF), nil
}

func parseRGB(s string) (Color, error) {
	vals, err := parseArgs(s, "rgb(", ")")
	if err != nil || len(vals) != 3 {
		return Color{}, errors.New("color: invalid rgb(): " + s)
	}
	r, _ := parseComponent(vals[0])
	g, _ := parseComponent(vals[1])
	b, _ := parseComponent(vals[2])
	return NewRGB(r, g, b), nil
}

func parseRGBA(s string) (Color, error) {
	vals, err := parseArgs(s, "rgba(", ")")
	if err != nil || len(vals) != 4 {
		return Color{}, errors.New("color: invalid rgba(): " + s)
	}
	r, _ := parseComponent(vals[0])
	g, _ := parseComponent(vals[1])
	b, _ := parseComponent(vals[2])
	a, _ := strconv.ParseFloat(strings.TrimSpace(vals[3]), 64)
	c := NewRGB(r, g, b)
	c.Alpha = clamp(a)
	return c, nil
}

func parseCMYK(s string) (Color, error) {
	vals, err := parseArgs(s, "cmyk(", ")")
	if err != nil || len(vals) != 4 {
		return Color{}, errors.New("color: invalid cmyk(): " + s)
	}
	c, _ := parseComponent(vals[0])
	m, _ := parseComponent(vals[1])
	y, _ := parseComponent(vals[2])
	k, _ := parseComponent(vals[3])
	return NewCMYK(c, m, y, k), nil
}

func parseGray(s string) (Color, error) {
	s = strings.ToLower(s)
	s = strings.TrimPrefix(s, "gray(")
	s = strings.TrimPrefix(s, "grey(")
	s = strings.TrimSuffix(s, ")")
	g, _ := parseComponent(s)
	return NewGray(g), nil
}

func parseArgs(s, prefix, suffix string) ([]string, error) {
	s = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)), prefix)
	s = strings.TrimSuffix(s, suffix)
	return strings.Split(s, ","), nil
}

func parseComponent(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		return clamp(v / 100), err
	}
	v, err := strconv.ParseFloat(s, 64)
	if v > 1 {
		v /= 255
	}
	return clamp(v), err
}

// ---- Conversions ---------------------------------------------------------

func RGBToCMYK(r, g, b float64) (c, m, y, k float64) {
	k = 1 - math.Max(r, math.Max(g, b))
	if k == 1 {
		return 0, 0, 0, 1
	}
	inv := 1 / (1 - k)
	c = (1 - r - k) * inv
	m = (1 - g - k) * inv
	y = (1 - b - k) * inv
	return
}

func CMYKToRGB(c, m, y, k float64) (r, g, b float64) {
	r = (1 - c) * (1 - k)
	g = (1 - m) * (1 - k)
	b = (1 - y) * (1 - k)
	return
}

func RGBToGray(r, g, b float64) float64 {
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func (c Color) ToRGB() Color {
	switch c.Type {
	case ColorTypeRGB:
		return c
	case ColorTypeGray:
		return NewRGB(c.Gray, c.Gray, c.Gray)
	case ColorTypeCMYK:
		r, g, b := CMYKToRGB(c.C, c.M, c.Y, c.K)
		return NewRGB(r, g, b)
	default:
		return Black
	}
}

func (c Color) ToCMYK() Color {
	switch c.Type {
	case ColorTypeCMYK:
		return c
	case ColorTypeRGB:
		cv, m, y, k := RGBToCMYK(c.R, c.G, c.B)
		return NewCMYK(cv, m, y, k)
	case ColorTypeGray:
		return NewCMYK(0, 0, 0, 1-c.Gray)
	default:
		return NewCMYK(0, 0, 0, 1)
	}
}

func (c Color) HexString() string {
	rgb := c.ToRGB()
	const hex = "0123456789abcdef"
	var b strings.Builder
	b.WriteByte('#')
	for _, v := range [3]float64{rgb.R, rgb.G, rgb.B} {
		n := int(math.Round(v * 255))
		b.WriteByte(hex[(n>>4)&0xF])
		b.WriteByte(hex[n&0xF])
	}
	return b.String()
}

// ---- Spot color registry (thread-safe) ----------------------------------

type SpotColorDef struct {
	Name      string
	AltSpace  string
	AltValues []float64
}

// SpotRegistry is a thread-safe registry of spot colors used in a document.
type SpotRegistry struct {
	mu    sync.RWMutex
	defs  map[string]SpotColorDef
	order []string
}

func NewSpotRegistry() *SpotRegistry {
	return &SpotRegistry{defs: make(map[string]SpotColorDef)}
}

// Add registers a spot color. Returns false if already registered.
func (sr *SpotRegistry) Add(def SpotColorDef) bool {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if _, ok := sr.defs[def.Name]; ok {
		return false
	}
	sr.defs[def.Name] = def
	sr.order = append(sr.order, def.Name)
	return true
}

// All returns spot color definitions in registration order.
func (sr *SpotRegistry) All() []SpotColorDef {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	out := make([]SpotColorDef, 0, len(sr.order))
	for _, name := range sr.order {
		out = append(out, sr.defs[name])
	}
	return out
}

// ---- ICC profile --------------------------------------------------------

type ICCProfile struct {
	Data       []byte
	Components int
}

// ---- helpers ------------------------------------------------------------

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func pdfName(s string) string {
	return strings.ReplaceAll(s, " ", "_")
}

var namedColors = map[string]Color{
	"black": Black, "white": White, "red": Red, "green": Green, "blue": Blue,
	"yellow":  NewRGB(1, 1, 0),
	"cyan":    NewRGB(0, 1, 1),
	"magenta": NewRGB(1, 0, 1),
	"orange":  NewRGB(1, 0.647, 0),
	"purple":  NewRGB(0.502, 0, 0.502),
	"pink":    NewRGB(1, 0.753, 0.796),
	"brown":   NewRGB(0.647, 0.165, 0.165),
	"gray":    NewRGB(0.502, 0.502, 0.502),
	"grey":    NewRGB(0.502, 0.502, 0.502),
	"silver":  NewRGB(0.753, 0.753, 0.753),
	"gold":    NewRGB(1, 0.843, 0),
	"navy":    NewRGB(0, 0, 0.502),
	"teal":    NewRGB(0, 0.502, 0.502),
	"maroon":  NewRGB(0.502, 0, 0),
	"olive":   NewRGB(0.502, 0.502, 0),
	"lime":    NewRGB(0, 1, 0),
	"aqua":    NewRGB(0, 1, 1),
	"fuchsia": NewRGB(1, 0, 1),
}
