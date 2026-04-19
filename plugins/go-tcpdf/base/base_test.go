package base

import (
	"math"
	"testing"
)

func TestNew(t *testing.T) {
	b, err := New("mm")
	if err != nil {
		t.Fatal(err)
	}
	if b.Unit != "mm" {
		t.Fatalf("unit: %v", b.Unit)
	}
	if b.PON == nil {
		t.Fatal("PON should not be nil")
	}
}

func TestNewUnknownUnit(t *testing.T) {
	if _, err := New("zz"); err == nil {
		t.Fatal("expected error for unknown unit")
	}
}

func TestToPoints(t *testing.T) {
	b, _ := New("mm")
	// 25.4mm = 72pt (1 inch)
	pts := b.ToPoints(25.4)
	if math.Abs(pts-72) > 1e-4 {
		t.Fatalf("ToPoints(25.4mm): want 72pt got %v", pts)
	}
}

func TestToUnit(t *testing.T) {
	b, _ := New("mm")
	mm := b.ToUnit(72)
	if math.Abs(mm-25.4) > 1e-4 {
		t.Fatalf("ToUnit(72pt): want 25.4mm got %v", mm)
	}
}

func TestToPointsToUnitRoundTrip(t *testing.T) {
	for _, unit := range []string{"mm", "cm", "in", "pt", "px"} {
		b, _ := New(unit)
		original := 42.5
		pts := b.ToPoints(original)
		back := b.ToUnit(pts)
		if math.Abs(back-original) > 1e-9 {
			t.Errorf("unit %s round-trip: want %v got %v", unit, original, back)
		}
	}
}

func TestToYPoints(t *testing.T) {
	b, _ := New("mm")
	b.PageHeight = 841.89 // A4 height in points
	// Y=0 (top of page) should map to pageHeight
	py := b.ToYPoints(0)
	if math.Abs(py-841.89) > 1e-4 {
		t.Fatalf("ToYPoints(0): want 841.89 got %v", py)
	}
	// Y=297mm (bottom of A4) should map to 0
	py = b.ToYPoints(297)
	if math.Abs(py) > 0.1 {
		t.Fatalf("ToYPoints(297mm): want ~0 got %v", py)
	}
}

func TestSetRTL(t *testing.T) {
	b, _ := New("mm")
	b.SetRTL(true)
	if !b.IsRTL {
		t.Fatal("IsRTL should be true")
	}
	b.SetRTL(false)
	if b.IsRTL {
		t.Fatal("IsRTL should be false")
	}
}

func TestDefaultPageContent(t *testing.T) {
	b, _ := New("mm")
	if !b.IsDefaultPageContentEnabled() {
		t.Fatal("default page content should be enabled by default")
	}
	b.EnableDefaultPageContent(false)
	if b.IsDefaultPageContentEnabled() {
		t.Fatal("default page content should be disabled")
	}
}

func TestZeroWidthBreakPoints(t *testing.T) {
	b, _ := New("mm")
	if b.IsZeroWidthBreakPointsEnabled() {
		t.Fatal("should be disabled by default")
	}
	b.EnableZeroWidthBreakPoints(true)
	if !b.IsZeroWidthBreakPointsEnabled() {
		t.Fatal("should be enabled")
	}
}

func TestSetSpaceRegexp(t *testing.T) {
	b, _ := New("mm")
	b.SetSpaceRegexp(`/\s/`)
	if b.SpaceRegexp != `/\s/` {
		t.Fatalf("unexpected: %q", b.SpaceRegexp)
	}
	// Empty string resets to default
	b.SetSpaceRegexp("")
	if b.SpaceRegexp == "" {
		t.Fatal("empty reset should restore default")
	}
}

// ---- PON ----------------------------------------------------------------

func TestPONNext(t *testing.T) {
	p := &PON{}
	n1 := p.Next()
	n2 := p.Next()
	n3 := p.Next()
	if n1 != 1 || n2 != 2 || n3 != 3 {
		t.Fatalf("PON sequence: %d %d %d", n1, n2, n3)
	}
}

func TestPONCurrent(t *testing.T) {
	p := &PON{}
	p.Next()
	p.Next()
	if p.Current() != 2 {
		t.Fatalf("current: want 2 got %d", p.Current())
	}
}

func TestPONSet(t *testing.T) {
	p := &PON{}
	p.Set(100)
	if p.Current() != 100 {
		t.Fatalf("Set: want 100 got %d", p.Current())
	}
	n := p.Next()
	if n != 101 {
		t.Fatalf("Next after Set(100): want 101 got %d", n)
	}
}

func TestPONConcurrency(t *testing.T) {
	p := &PON{}
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			p.Next()
			done <- true
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	if p.Current() != 100 {
		t.Fatalf("after 100 concurrent Next(): want 100 got %d", p.Current())
	}
}

// ---- CellBound / CellDef -----------------------------------------------

func TestZeroCellBound(t *testing.T) {
	cb := ZeroCellBound
	if cb.T != 0 || cb.R != 0 || cb.B != 0 || cb.L != 0 {
		t.Fatal("ZeroCellBound should be all zeros")
	}
}

func TestZeroCell(t *testing.T) {
	c := ZeroCell
	if c.BorderPos != BorderPosDefault {
		t.Fatalf("ZeroCell.BorderPos: want %v got %v", BorderPosDefault, c.BorderPos)
	}
}

func TestBorderPosConstants(t *testing.T) {
	if BorderPosDefault != 0 {
		t.Fatal("BorderPosDefault should be 0")
	}
	if BorderPosExternal >= 0 {
		t.Fatal("BorderPosExternal should be negative")
	}
	if BorderPosInternal <= 0 {
		t.Fatal("BorderPosInternal should be positive")
	}
}

// ---- Error type ---------------------------------------------------------

func TestErrUnknownUnit(t *testing.T) {
	err := &ErrUnknownUnit{Unit: "zz"}
	if err.Error() == "" {
		t.Fatal("ErrUnknownUnit.Error() should not be empty")
	}
	if err.Unit != "zz" {
		t.Fatal("Unit field not preserved")
	}
}
