package pdf

import (
	"beba/processor"
	"strings"
	"testing"
)

func TestPDFModule_Registration(t *testing.T) {
	vm := processor.NewEmpty()
	_, err := vm.RunString(`require('pdf')`)
	if err != nil {
		t.Fatalf("Failed to require('pdf'): %v", err)
	}
}

func TestPDFModule_Constructor_Default(t *testing.T) {
	vm := processor.NewEmpty()
	_, err := vm.RunString(`
		const pdf = require('pdf');
		const doc = pdf.TCPDF(); 
	`)
	if err != nil {
		t.Fatalf("Failed to create TCPDF with default args: %v", err)
	}
}

func TestPDFModule_Constructor_Args(t *testing.T) {
	vm := processor.NewEmpty()
	_, err := vm.RunString(`
		const pdf = require('pdf');
		const doc = pdf.TCPDF({
			title: "Custom Title",
			author: "Custom Author",
			orientation: "L",
			format: "A5",
			unit: "in"
		});
	`)
	if err != nil {
		t.Fatalf("Failed to create TCPDF with custom args: %v", err)
	}
}

func TestPDFModule_Methods(t *testing.T) {
	vm := processor.NewEmpty()
	_, err := vm.RunString(`
		const pdf = require('pdf');
		const doc = pdf.TCPDF();
		doc.AddPage();
		doc.SetFont("helvetica", "B", 16);
		doc.Cell(40, 10, "Hello World!");
		doc.WriteHTML("<b>HTML content</b>", true, false);
	`)
	if err != nil {
		t.Fatalf("Failed to call PDF methods from JS: %v", err)
	}
}

func TestPDFModule_Output(t *testing.T) {
	vm := processor.NewEmpty()
	val, err := vm.RunString(`
		const pdf = require('pdf');
		const doc = pdf.TCPDF();
		doc.AddPage();
		doc.Cell(0, 10, "Test Output");
		doc.GetOutPDFString();
	`)
	if err != nil {
		t.Fatalf("Failed to get PDF output: %v", err)
	}
	out := val.String()
	if !strings.HasPrefix(out, "%PDF-") {
		t.Errorf("Expected PDF prefix %%PDF-, got %q", out[:10])
	}
}
