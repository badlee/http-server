package color

import (
	"math"
	"testing"
)

// ---- constructors --------------------------------------------------------

func TestNewRGB(t *testing.T) {
	c := NewRGB(0.5, 0.25, 1.0)
	if c.Type != ColorTypeRGB {
		t.Fatalf("expected ColorTypeRGB, got %v", c.Type)
	}
	if c.R != 0.5 || c.G != 0.25 || c.B != 1.0 {
		t.Fatalf("unexpected components: %v", c)
	}
	if c.Alpha != 1.0 {
		t.Fatalf("expected Alpha=1, got %v", c.Alpha)
	}
}

func TestNewRGB255(t *testing.T) {
	c := NewRGB255(255, 128, 0)
	if math.Abs(c.R-1.0) > 1e-6 {
		t.Fatalf("R: want 1.0 got %v", c.R)
	}
	if math.Abs(c.G-128.0/255) > 1e-6 {
		t.Fatalf("G: want %v got %v", 128.0/255, c.G)
	}
	if c.B != 0 {
		t.Fatalf("B: want 0 got %v", c.B)
	}
}

func TestNewCMYK(t *testing.T) {
	c := NewCMYK(0.1, 0.2, 0.3, 0.4)
	if c.Type != ColorTypeCMYK {
		t.Fatal("expected ColorTypeCMYK")
	}
}

func TestNewCMYK100(t *testing.T) {
	c := NewCMYK100(10, 20, 30, 40)
	if math.Abs(c.C-0.1) > 1e-9 || math.Abs(c.M-0.2) > 1e-9 {
		t.Fatalf("unexpected CMYK: %+v", c)
	}
}

func TestNewGray(t *testing.T) {
	c := NewGray(0.75)
	if c.Type != ColorTypeGray || c.Gray != 0.75 {
		t.Fatalf("unexpected gray: %+v", c)
	}
}

func TestClamp(t *testing.T) {
	c := NewRGB(-0.5, 2.0, 0.5)
	if c.R != 0 || c.G != 1 || c.B != 0.5 {
		t.Fatalf("clamp failed: %+v", c)
	}
}

func TestWithAlpha(t *testing.T) {
	c := NewRGB(1, 0, 0).WithAlpha(0.3)
	if math.Abs(c.Alpha-0.3) > 1e-9 {
		t.Fatalf("alpha: want 0.3 got %v", c.Alpha)
	}
}

func TestNewSpot(t *testing.T) {
	c := NewSpot("PANTONE 485", 0.8)
	if c.Type != ColorTypeSpot {
		t.Fatal("expected ColorTypeSpot")
	}
	if c.SpotName != "PANTONE 485" || math.Abs(c.SpotTint-0.8) > 1e-9 {
		t.Fatalf("spot: %+v", c)
	}
}

// ---- PDF operators -------------------------------------------------------

func TestFillOperatorRGB(t *testing.T) {
	c := NewRGB(1, 0, 0)
	op := c.FillOperator()
	if op != "1 0 0 rg" {
		t.Fatalf("got %q", op)
	}
}

func TestFillOperatorGray(t *testing.T) {
	op := Black.FillOperator()
	if op != "0 g" {
		t.Fatalf("got %q", op)
	}
	op = White.FillOperator()
	if op != "1 g" {
		t.Fatalf("got %q", op)
	}
}

func TestFillOperatorCMYK(t *testing.T) {
	c := NewCMYK(0, 0.5, 1, 0)
	op := c.FillOperator()
	if op != "0 0.5 1 0 k" {
		t.Fatalf("got %q", op)
	}
}

func TestStrokeOperatorRGB(t *testing.T) {
	c := NewRGB(0, 1, 0)
	op := c.StrokeOperator()
	if op != "0 1 0 RG" {
		t.Fatalf("got %q", op)
	}
}

func TestTransparentOperator(t *testing.T) {
	op := Transparent.FillOperator()
	if op != "" {
		t.Fatalf("expected empty, got %q", op)
	}
}

// ---- CSS parsing ---------------------------------------------------------

func TestParseCSSHex3(t *testing.T) {
	c, err := ParseCSS("#f60")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(c.R-1.0) > 1e-6 || math.Abs(c.G-0.4) > 1e-3 || c.B != 0 {
		t.Fatalf("unexpected: %+v", c)
	}
}

func TestParseCSSHex6(t *testing.T) {
	c, err := ParseCSS("#ff6600")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(c.R-1.0) > 1e-6 {
		t.Fatalf("R: %v", c.R)
	}
}

func TestParseCSSRGB(t *testing.T) {
	c, err := ParseCSS("rgb(255, 128, 0)")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(c.R-1.0) > 1e-6 {
		t.Fatalf("R: %v", c.R)
	}
}

func TestParseCSSRGBA(t *testing.T) {
	c, err := ParseCSS("rgba(255, 0, 0, 0.5)")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(c.Alpha-0.5) > 1e-6 {
		t.Fatalf("alpha: %v", c.Alpha)
	}
}

func TestParseCSSnamed(t *testing.T) {
	for _, name := range []string{"black", "white", "red", "blue", "green"} {
		if _, err := ParseCSS(name); err != nil {
			t.Fatalf("ParseCSS(%q): %v", name, err)
		}
	}
}

func TestParseCSSTransparent(t *testing.T) {
	c, err := ParseCSS("transparent")
	if err != nil {
		t.Fatal(err)
	}
	if c.Type != ColorTypeNone {
		t.Fatalf("expected None, got %v", c.Type)
	}
}

func TestParseCSSUnknown(t *testing.T) {
	if _, err := ParseCSS("notacolor"); err == nil {
		t.Fatal("expected error for unknown color")
	}
}

// ---- Conversions ---------------------------------------------------------

func TestRGBToCMYK(t *testing.T) {
	c, m, y, k := RGBToCMYK(1, 0, 0)
	if c != 0 || m != 1 || y != 1 || k != 0 {
		t.Fatalf("RGB(1,0,0) → CMYK got %v %v %v %v", c, m, y, k)
	}
}

func TestCMYKToRGB(t *testing.T) {
	r, g, b := CMYKToRGB(0, 1, 1, 0)
	if math.Abs(r-1) > 1e-9 || g != 0 || b != 0 {
		t.Fatalf("CMYK(0,1,1,0) → RGB got %v %v %v", r, g, b)
	}
}

func TestRGBToGray(t *testing.T) {
	g := RGBToGray(1, 1, 1)
	if math.Abs(g-1) > 1e-6 {
		t.Fatalf("white gray: %v", g)
	}
	g = RGBToGray(0, 0, 0)
	if g != 0 {
		t.Fatalf("black gray: %v", g)
	}
}

func TestToRGB(t *testing.T) {
	c := NewGray(0.5).ToRGB()
	if c.Type != ColorTypeRGB {
		t.Fatal("expected RGB")
	}
	if math.Abs(c.R-0.5) > 1e-9 || math.Abs(c.G-0.5) > 1e-9 {
		t.Fatalf("gray to RGB: %+v", c)
	}
}

func TestToCMYK(t *testing.T) {
	c := NewRGB(1, 0, 0).ToCMYK()
	if c.Type != ColorTypeCMYK {
		t.Fatal("expected CMYK")
	}
}

func TestHexString(t *testing.T) {
	s := NewRGB255(255, 102, 0).HexString()
	if s != "#ff6600" {
		t.Fatalf("got %q", s)
	}
}

// ---- SpotRegistry --------------------------------------------------------

func TestSpotRegistry(t *testing.T) {
	sr := NewSpotRegistry()
	sr.Add(SpotColorDef{Name: "PANTONE 485", AltSpace: "DeviceRGB", AltValues: []float64{1, 0, 0}})
	sr.Add(SpotColorDef{Name: "PANTONE 286", AltSpace: "DeviceRGB", AltValues: []float64{0, 0, 1}})
	if len(sr.All()) != 2 {
		t.Fatalf("expected 2 spots, got %d", len(sr.All()))
	}
	// Duplicate should not be added
	added := sr.Add(SpotColorDef{Name: "PANTONE 485"})
	if added {
		t.Fatal("duplicate spot should not be added")
	}
	if len(sr.All()) != 2 {
		t.Fatal("count should remain 2")
	}
}
