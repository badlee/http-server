package graph

import (
	"strings"
	"testing"

	"github.com/tecnickcom/go-tcpdf/color"
)

// ---- Matrix -------------------------------------------------------------

func TestIdentity(t *testing.T) {
	m := Identity()
	if m.A != 1 || m.D != 1 || m.B != 0 || m.C != 0 || m.E != 0 || m.F != 0 {
		t.Fatalf("unexpected identity: %+v", m)
	}
}

func TestTranslate(t *testing.T) {
	m := Translate(10, 20)
	if m.E != 10 || m.F != 20 || m.A != 1 || m.D != 1 {
		t.Fatalf("translate: %+v", m)
	}
}

func TestScale(t *testing.T) {
	m := Scale(2, 3)
	if m.A != 2 || m.D != 3 || m.B != 0 || m.C != 0 {
		t.Fatalf("scale: %+v", m)
	}
}

func TestRotateZero(t *testing.T) {
	m := Rotate(0)
	if abs(m.A-1) > 1e-10 || abs(m.D-1) > 1e-10 || abs(m.B) > 1e-10 {
		t.Fatalf("rotate(0): %+v", m)
	}
}

func TestMatrixPDF(t *testing.T) {
	m := Identity()
	pdf := m.PDF()
	if !strings.Contains(pdf, "cm") {
		t.Fatalf("PDF() should contain 'cm': %q", pdf)
	}
}

func TestMatrixMultiply(t *testing.T) {
	t1 := Translate(10, 0)
	t2 := Translate(0, 20)
	result := t1.Multiply(t2)
	if abs(result.E-10) > 1e-9 || abs(result.F-20) > 1e-9 {
		t.Fatalf("multiply: %+v", result)
	}
}

// ---- Draw operators -----------------------------------------------------

func TestNewDraw(t *testing.T) {
	d := NewDraw()
	if d == nil {
		t.Fatal("NewDraw returned nil")
	}
}

func TestSetLineWidth(t *testing.T) {
	d := NewDraw()
	op := d.SetLineWidth(2.5)
	if !strings.Contains(op, "w") {
		t.Fatalf("SetLineWidth: %q", op)
	}
	if !strings.Contains(op, "2.5") {
		t.Fatalf("SetLineWidth should contain width value: %q", op)
	}
}

func TestSetLineCap(t *testing.T) {
	d := NewDraw()
	for _, cap := range []int{0, 1, 2} {
		op := d.SetLineCap(cap)
		if !strings.Contains(op, "J") {
			t.Fatalf("SetLineCap(%d): %q", cap, op)
		}
	}
}

func TestSetLineJoin(t *testing.T) {
	d := NewDraw()
	op := d.SetLineJoin(1)
	if !strings.Contains(op, "j") {
		t.Fatalf("SetLineJoin: %q", op)
	}
}

func TestSetDash(t *testing.T) {
	d := NewDraw()
	op := d.SetDash([]float64{3, 1.5}, 0)
	if !strings.Contains(op, "d") || !strings.Contains(op, "[") {
		t.Fatalf("SetDash: %q", op)
	}
}

func TestSetDashSolid(t *testing.T) {
	d := NewDraw()
	op := d.SetDash(nil, 0)
	if op != "[] 0 d\n" {
		t.Fatalf("solid line: %q", op)
	}
}

func TestSetFillColor(t *testing.T) {
	d := NewDraw()
	op := d.SetFillColor(color.NewRGB(1, 0, 0))
	if !strings.Contains(op, "rg") {
		t.Fatalf("SetFillColor: %q", op)
	}
}

func TestSetStrokeColor(t *testing.T) {
	d := NewDraw()
	op := d.SetStrokeColor(color.Black)
	if !strings.Contains(op, "G") {
		t.Fatalf("SetStrokeColor: %q", op)
	}
}

// ---- Path operators -----------------------------------------------------

func TestMoveTo(t *testing.T) {
	d := NewDraw()
	op := d.MoveTo(10, 20)
	if !strings.Contains(op, "m") || !strings.Contains(op, "10") || !strings.Contains(op, "20") {
		t.Fatalf("MoveTo: %q", op)
	}
}

func TestLineTo(t *testing.T) {
	d := NewDraw()
	op := d.LineTo(30, 40)
	if !strings.Contains(op, "l") {
		t.Fatalf("LineTo: %q", op)
	}
}

func TestCurveTo(t *testing.T) {
	d := NewDraw()
	op := d.CurveTo(1, 2, 3, 4, 5, 6)
	if !strings.Contains(op, "c") {
		t.Fatalf("CurveTo: %q", op)
	}
}

func TestClosePath(t *testing.T) {
	d := NewDraw()
	op := d.ClosePath()
	if op != "h\n" {
		t.Fatalf("ClosePath: %q", op)
	}
}

func TestStroke(t *testing.T) {
	d := NewDraw()
	if d.Stroke() != "S\n" {
		t.Fatal("Stroke should be 'S\\n'")
	}
}

func TestFill(t *testing.T) {
	d := NewDraw()
	if d.Fill() != "f\n" {
		t.Fatal("Fill should be 'f\\n'")
	}
}

func TestFillAndStroke(t *testing.T) {
	d := NewDraw()
	if d.FillAndStroke() != "B\n" {
		t.Fatal("FillAndStroke should be 'B\\n'")
	}
}

// ---- High-level shapes --------------------------------------------------

func TestRect(t *testing.T) {
	d := NewDraw()
	op := d.Rect(10, 20, 100, 50, "FD")
	if !strings.Contains(op, "re") {
		t.Fatalf("Rect missing 're': %q", op)
	}
	if !strings.Contains(op, "B") {
		t.Fatalf("Rect 'FD' should produce 'B': %q", op)
	}
}

func TestRectStyleS(t *testing.T) {
	d := NewDraw()
	op := d.Rect(0, 0, 10, 10, "S")
	if !strings.Contains(op, "S") {
		t.Fatalf("Rect 'S' should contain 'S': %q", op)
	}
}

func TestLine(t *testing.T) {
	d := NewDraw()
	op := d.Line(0, 0, 100, 100)
	if !strings.Contains(op, "m") || !strings.Contains(op, "l") || !strings.Contains(op, "S") {
		t.Fatalf("Line: %q", op)
	}
}

func TestCircle(t *testing.T) {
	d := NewDraw()
	op := d.Circle(50, 50, 30, "FD")
	if !strings.Contains(op, "c") {
		t.Fatalf("Circle should contain bezier curves: %q", op)
	}
}

func TestEllipse(t *testing.T) {
	d := NewDraw()
	op := d.Ellipse(50, 50, 40, 20, 0, "S")
	if !strings.Contains(op, "c") || !strings.Contains(op, "S") {
		t.Fatalf("Ellipse: %q", op)
	}
}

func TestRoundedRect(t *testing.T) {
	d := NewDraw()
	op := d.RoundedRect(10, 10, 100, 50, 5, 5, "FD")
	// Should contain both lines and curves
	if !strings.Contains(op, "l") || !strings.Contains(op, "c") {
		t.Fatalf("RoundedRect should have l and c operators: %q", op)
	}
}

// ---- Graphics state stack -----------------------------------------------

func TestSaveRestoreState(t *testing.T) {
	d := NewDraw()
	save := d.SaveState()
	if save != "q\n" {
		t.Fatalf("SaveState: %q", save)
	}
	restore, err := d.RestoreState()
	if err != nil {
		t.Fatal(err)
	}
	if restore != "Q\n" {
		t.Fatalf("RestoreState: %q", restore)
	}
}

func TestRestoreStateUnderflow(t *testing.T) {
	d := NewDraw()
	if _, err := d.RestoreState(); err == nil {
		t.Fatal("expected underflow error")
	}
}

func TestGStateStackDepth(t *testing.T) {
	stack := NewGStateStack()
	if stack.Depth() != 0 {
		t.Fatal("initial depth should be 0")
	}
	stack.Push(GState{CTM: Identity()})
	stack.Push(GState{CTM: Identity()})
	if stack.Depth() != 2 {
		t.Fatalf("depth after 2 pushes: %d", stack.Depth())
	}
	stack.Pop()
	if stack.Depth() != 1 {
		t.Fatalf("depth after pop: %d", stack.Depth())
	}
}

// ---- Gradient constructors ----------------------------------------------

func TestNewLinearGradient(t *testing.T) {
	stops := []GradientStop{
		{Offset: 0, Color: color.Black},
		{Offset: 1, Color: color.White},
	}
	g := NewLinearGradient(0, 0, 100, 0, stops)
	if g.Type != GradientLinear {
		t.Fatal("expected GradientLinear")
	}
}

func TestNewRadialGradient(t *testing.T) {
	stops := []GradientStop{
		{Offset: 0, Color: color.White},
		{Offset: 1, Color: color.Black},
	}
	g := NewRadialGradient(50, 50, 0, 50, 50, 30, stops)
	if g.Type != GradientRadial {
		t.Fatal("expected GradientRadial")
	}
}

// ---- helpers ------------------------------------------------------------

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
