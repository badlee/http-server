package page

import (
	"math"
	"testing"
)

// ---- unit conversion ----------------------------------------------------

func TestToPoints(t *testing.T) {
	tests := []struct {
		unit  Unit
		value float64
		want  float64
	}{
		{UnitPt, 72, 72},
		{UnitIN, 1, 72},
		{UnitMM, 25.4, 72},
		{UnitCM, 2.54, 72},
		{UnitPX, 96, 72},
	}
	for _, tc := range tests {
		got, err := ToPoints(tc.value, tc.unit)
		if err != nil {
			t.Errorf("ToPoints(%v, %v): %v", tc.value, tc.unit, err)
			continue
		}
		if math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("ToPoints(%v, %v) = %v, want %v", tc.value, tc.unit, got, tc.want)
		}
	}
}

func TestToPointsUnknownUnit(t *testing.T) {
	if _, err := ToPoints(1, Unit("xx")); err == nil {
		t.Fatal("expected error for unknown unit")
	}
}

func TestToUnit(t *testing.T) {
	pts, _ := ToPoints(210, UnitMM)
	mm, err := ToUnit(pts, UnitMM)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(mm-210) > 1e-6 {
		t.Fatalf("round-trip: want 210mm got %v", mm)
	}
}

func TestScaleFactor(t *testing.T) {
	k, err := ScaleFactor(UnitMM)
	if err != nil {
		t.Fatal(err)
	}
	expected := 72.0 / 25.4
	if math.Abs(k-expected) > 1e-9 {
		t.Fatalf("mm scale: want %v got %v", expected, k)
	}
}

// ---- page formats --------------------------------------------------------

func TestGetFormatA4(t *testing.T) {
	f, err := GetFormat("A4")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(f.W-595.28) > 0.01 || math.Abs(f.H-841.89) > 0.01 {
		t.Fatalf("A4 unexpected: %+v", f)
	}
}

func TestGetFormatCaseInsensitive(t *testing.T) {
	f1, _ := GetFormat("A4")
	f2, _ := GetFormat("a4")
	if f1 != f2 {
		t.Fatal("case-insensitive GetFormat failed")
	}
}

func TestGetFormatUnknown(t *testing.T) {
	if _, err := GetFormat("XXXXX"); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestGetFormatLetter(t *testing.T) {
	f, err := GetFormat("LETTER")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(f.W-612) > 0.1 || math.Abs(f.H-792) > 0.1 {
		t.Fatalf("LETTER: %+v", f)
	}
}

// ---- page box ------------------------------------------------------------

func TestBoxFromFormatPortrait(t *testing.T) {
	b, err := BoxFromFormat("A4", Portrait)
	if err != nil {
		t.Fatal(err)
	}
	if b.Width() < b.Height() {
		// portrait: width < height
	} else if b.Width() == b.Height() {
		t.Fatal("A4 portrait: width == height")
	}
	// width should be the shorter dimension
	if b.Width() > b.Height() {
		t.Fatalf("A4 portrait should have W < H: %v > %v", b.Width(), b.Height())
	}
}

func TestBoxFromFormatLandscape(t *testing.T) {
	b, err := BoxFromFormat("A4", Landscape)
	if err != nil {
		t.Fatal(err)
	}
	if b.Width() < b.Height() {
		t.Fatalf("A4 landscape should have W > H: %v < %v", b.Width(), b.Height())
	}
}

func TestNewBox(t *testing.T) {
	b := NewBox(595.28, 841.89)
	if b.Llx != 0 || b.Lly != 0 {
		t.Fatal("NewBox should have origin at 0,0")
	}
	if math.Abs(b.Width()-595.28) > 1e-6 {
		t.Fatalf("Width: %v", b.Width())
	}
}

// ---- page manager --------------------------------------------------------

func TestPageManagerAdd(t *testing.T) {
	pm, err := New(UnitMM, "A4", Portrait, Margins{10, 10, 10, 10})
	if err != nil {
		t.Fatal(err)
	}
	if pm.Count() != 0 {
		t.Fatal("fresh manager should have 0 pages")
	}
	pg, err := pm.Add("", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pm.Count() != 1 {
		t.Fatal("expected 1 page")
	}
	if pg.Index != 0 {
		t.Fatalf("first page index should be 0, got %d", pg.Index)
	}
}

func TestPageManagerCurrent(t *testing.T) {
	pm, _ := New(UnitMM, "A4", Portrait, Margins{})
	if pm.Current() != nil {
		t.Fatal("current should be nil before any page")
	}
	pg, _ := pm.Add("", "", nil)
	if pm.Current() != pg {
		t.Fatal("Current() should return last page")
	}
}

func TestPageManagerGet(t *testing.T) {
	pm, _ := New(UnitMM, "A4", Portrait, Margins{})
	pm.Add("", "", nil)
	pm.Add("", "", nil)
	pg, err := pm.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if pg.Index != 1 {
		t.Fatalf("expected index 1, got %d", pg.Index)
	}
}

func TestPageManagerGetOutOfRange(t *testing.T) {
	pm, _ := New(UnitMM, "A4", Portrait, Margins{})
	if _, err := pm.Get(0); err == nil {
		t.Fatal("expected error for empty page list")
	}
}

func TestPageDimensions(t *testing.T) {
	pm, _ := New(UnitMM, "A4", Portrait, Margins{10, 10, 10, 10})
	pg, _ := pm.Add("", "", nil)
	w := pg.Width()
	h := pg.Height()
	// A4 in mm: 210 x 297
	if math.Abs(w-210) > 0.1 {
		t.Fatalf("A4 width: want ~210mm got %v", w)
	}
	if math.Abs(h-297) > 0.1 {
		t.Fatalf("A4 height: want ~297mm got %v", h)
	}
}

func TestPageCustomFormat(t *testing.T) {
	pm, _ := New(UnitMM, "A4", Portrait, Margins{})
	pg, err := pm.Add("LETTER", Landscape, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Letter landscape: 279.4 x 215.9 mm
	if pg.Width() < pg.Height() {
		t.Fatalf("Letter landscape should have W > H: %v %v", pg.Width(), pg.Height())
	}
}

func TestPageAll(t *testing.T) {
	pm, _ := New(UnitMM, "A4", Portrait, Margins{})
	pm.Add("", "", nil)
	pm.Add("", "", nil)
	pm.Add("", "", nil)
	all := pm.All()
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}
}

// ---- box ----------------------------------------------------------------

func TestBoxWidthHeight(t *testing.T) {
	b := Box{Llx: 10, Lly: 20, Urx: 110, Ury: 220}
	if b.Width() != 100 || b.Height() != 200 {
		t.Fatalf("Box W/H: %v %v", b.Width(), b.Height())
	}
}
