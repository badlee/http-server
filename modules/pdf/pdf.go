package pdf

import (
	"beba/modules"

	"github.com/dop251/goja"
	"github.com/tecnickcom/go-tcpdf"
	"github.com/tecnickcom/go-tcpdf/classobjects"
	"github.com/tecnickcom/go-tcpdf/color"
	"github.com/tecnickcom/go-tcpdf/page"
)

type Module struct{}

func (m *Module) Name() string {
	return "pdf"
}

func (m *Module) Doc() string {
	return "PDF generation module using go-tcpdf"
}

func (m *Module) ToJSObject(vm *goja.Runtime) goja.Value {
	obj := vm.NewObject()
	m.Loader(nil, vm, obj)
	return obj
}

func (m *Module) Loader(_ any, vm *goja.Runtime, moduleObject *goja.Object) {
	// CommonJS support
	module := moduleObject
	if exp := moduleObject.Get("exports"); exp != nil && !goja.IsUndefined(exp) {
		module = exp.ToObject(vm)
	}

	module.Set("TCPDF", m.newTCPDF(vm))
}

func (m *Module) newTCPDF(vm *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		cfg := classobjects.DefaultConfig()
		if len(call.Arguments) > 0 {
			if opts, ok := call.Arguments[0].Export().(map[string]interface{}); ok {
				if v, ok := opts["unit"].(string); ok {
					cfg.Unit = v
				}
				if v, ok := opts["format"].(string); ok {
					cfg.Format = v
				}
				if v, ok := opts["orientation"].(string); ok {
					cfg.Orientation = page.Orientation(v)
				}
				if v, ok := opts["unicode"].(bool); ok {
					cfg.Unicode = v
				}
				if v, ok := opts["encoding"].(string); ok {
					cfg.Encoding = v
				}
				if v, ok := opts["subsetFonts"].(bool); ok {
					cfg.SubsetFonts = v
				}
				if v, ok := opts["compress"].(bool); ok {
					cfg.Compress = v
				}
			}
		}

		p, err := gotcpdf.New(cfg)
		if err != nil {
			panic(vm.NewGoError(err))
		}

		jsPdf := &jsTCPDF{pdf: p, vm: vm}
		return vm.ToValue(jsPdf)
	}
}

type jsTCPDF struct {
	pdf *gotcpdf.TCPDF
	vm  *goja.Runtime
}

func (j *jsTCPDF) SetTitle(title string) *jsTCPDF { j.pdf.SetTitle(title); return j }
func (j *jsTCPDF) SetAuthor(author string) *jsTCPDF { j.pdf.SetAuthor(author); return j }
func (j *jsTCPDF) SetSubject(subject string) *jsTCPDF { j.pdf.SetSubject(subject); return j }
func (j *jsTCPDF) SetKeywords(kw string) *jsTCPDF { j.pdf.SetKeywords(kw); return j }
func (j *jsTCPDF) SetCreator(creator string) *jsTCPDF { j.pdf.SetCreator(creator); return j }

func (j *jsTCPDF) AddPage(call goja.FunctionCall) goja.Value {
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		args[i] = arg.Export()
	}
	if err := j.pdf.AddPage(args...); err != nil {
		panic(j.vm.NewGoError(err))
	}
	return j.vm.ToValue(j)
}

func (j *jsTCPDF) SetFont(family, style string, size float64) {
	if err := j.pdf.SetFont(family, style, size); err != nil {
		panic(j.vm.NewGoError(err))
	}
}

func (j *jsTCPDF) Cell(w, h float64, txt, border string, ln int, align string, fill bool, link string) {
	if err := j.pdf.Cell(w, h, txt, border, ln, align, fill, link); err != nil {
		panic(j.vm.NewGoError(err))
	}
}

func (j *jsTCPDF) MultiCell(w, h float64, txt, border, align string, fill bool) {
	if err := j.pdf.MultiCell(w, h, txt, border, align, fill); err != nil {
		panic(j.vm.NewGoError(err))
	}
}

func (j *jsTCPDF) Write(h float64, txt, link string) {
	if err := j.pdf.Write(h, txt, link); err != nil {
		panic(j.vm.NewGoError(err))
	}
}

func (j *jsTCPDF) Ln(h float64) { j.pdf.Ln(h) }

func (j *jsTCPDF) SetX(x float64) { j.pdf.SetX(x) }
func (j *jsTCPDF) SetY(y float64) { j.pdf.SetY(y) }
func (j *jsTCPDF) SetXY(x, y float64) { j.pdf.SetXY(x, y) }
func (j *jsTCPDF) GetX() float64 { return j.pdf.GetX() }
func (j *jsTCPDF) GetY() float64 { return j.pdf.GetY() }

func (j *jsTCPDF) Image(path string, x, y, w, h float64, imgType, link string) {
	if err := j.pdf.Image(path, x, y, w, h, imgType, link); err != nil {
		panic(j.vm.NewGoError(err))
	}
}

func (j *jsTCPDF) WriteHTML(html string, ln, fill bool) {
	if err := j.pdf.WriteHTML(html, ln, fill); err != nil {
		panic(j.vm.NewGoError(err))
	}
}

func (j *jsTCPDF) SetFillColor(r, g, b float64) {
	j.pdf.SetFillColor(color.NewRGB(r, g, b))
}

func (j *jsTCPDF) SetDrawColor(r, g, b float64) {
	j.pdf.SetDrawColor(color.NewRGB(r, g, b))
}

func (j *jsTCPDF) SetTextColor(r, g, b float64) {
	j.pdf.SetTextColor(color.NewRGB(r, g, b))
}

func (j *jsTCPDF) Rect(x, y, w, h float64, style string) { j.pdf.Rect(x, y, w, h, style) }
func (j *jsTCPDF) Line(x1, y1, x2, y2 float64) { j.pdf.Line(x1, y1, x2, y2) }
func (j *jsTCPDF) Circle(x, y, r float64, style string) { j.pdf.Circle(x, y, r, style) }

func (j *jsTCPDF) GetOutPDFString() goja.Value {
	data, err := j.pdf.GetOutPDFString()
	if err != nil {
		panic(j.vm.NewGoError(err))
	}
	// Return as ArrayBuffer/Uint8Array if possible, or string.
	// For simplicity in this env, we might return a string or an object with bytes.
	return j.vm.ToValue(string(data))
}

func (j *jsTCPDF) SavePDF(path string) {
	if err := j.pdf.SavePDF(path); err != nil {
		panic(j.vm.NewGoError(err))
	}
}

func init() {
	modules.RegisterModule(&Module{})
}
