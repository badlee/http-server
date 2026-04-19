package javascript

import (
	"strings"
	"testing"

	"github.com/tecnickcom/go-tcpdf/base"
)

func newJS() *JavaScript {
	pon := &base.PON{}
	return New(pon)
}

// ---- JavaScript ---------------------------------------------------------

func TestAppendRawJS(t *testing.T) {
	j := newJS()
	j.AppendRawJavaScript("app.alert('hi');")
	j.AppendRawJavaScript("var x = 1;")
	if !strings.Contains(j.RawJS(), "app.alert") {
		t.Fatal("RawJS should contain appended script")
	}
	if !strings.Contains(j.RawJS(), "var x") {
		t.Fatal("RawJS should contain second script")
	}
}

func TestAddRawJSObj(t *testing.T) {
	j := newJS()
	id := j.AddRawJavaScriptObj("var y = 2;", false)
	if id <= 0 {
		t.Fatalf("JS object ID should be positive: %d", id)
	}
	if len(j.JSObjects()) != 1 {
		t.Fatalf("expected 1 JS object, got %d", len(j.JSObjects()))
	}
}

// ---- Annotations --------------------------------------------------------

func TestSetAnnotation(t *testing.T) {
	j := newJS()
	opts := AnnotOpts{Subtype: "Text", Contents: "Note"}
	id := j.SetAnnotation(0, 10, 20, 50, 30, "Test note", opts)
	if id <= 0 {
		t.Fatalf("annotation ID: %d", id)
	}
	annots := j.Annotations()
	if len(annots) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(annots))
	}
	if annots[0].Opts.Subtype != "Text" {
		t.Fatalf("subtype: %q", annots[0].Opts.Subtype)
	}
}

func TestDefaultAnnotOpts(t *testing.T) {
	j := newJS()
	opts := j.GetDefJSAnnotProp()
	if opts.Subtype != "Text" {
		t.Fatalf("default subtype: %q", opts.Subtype)
	}
}

func TestSetDefAnnotProp(t *testing.T) {
	j := newJS()
	j.SetDefJSAnnotProp(AnnotOpts{Subtype: "Stamp"})
	if j.GetDefJSAnnotProp().Subtype != "Stamp" {
		t.Fatal("SetDefJSAnnotProp not applied")
	}
}

// ---- Links --------------------------------------------------------------

func TestSetLink(t *testing.T) {
	j := newJS()
	id := j.SetLink(0, 10, 20, 80, 10, "https://example.com")
	if id <= 0 {
		t.Fatalf("link ID: %d", id)
	}
	links := j.Links()
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].URL != "https://example.com" {
		t.Fatalf("link URL: %q", links[0].URL)
	}
}

func TestAddInternalLink(t *testing.T) {
	j := newJS()
	dest := j.AddInternalLink(3, 100.0)
	if !strings.HasPrefix(dest, "#") {
		t.Fatalf("internal link should start with '#': %q", dest)
	}
	if !strings.Contains(dest, "3") {
		t.Fatalf("internal link should contain page: %q", dest)
	}
}

// ---- Bookmarks ----------------------------------------------------------

func TestSetBookmark(t *testing.T) {
	j := newJS()
	j.SetBookmark("Chapter 1", "", 0, 1, 0, 50, "", "")
	j.SetBookmark("Section 1.1", "", 1, 1, 0, 100, "B", "")
	bms := j.Bookmarks()
	if len(bms) != 2 {
		t.Fatalf("expected 2 bookmarks, got %d", len(bms))
	}
	if bms[0].Name != "Chapter 1" {
		t.Fatalf("bookmark name: %q", bms[0].Name)
	}
	if bms[1].Level != 1 {
		t.Fatalf("bookmark level: %d", bms[1].Level)
	}
}

// ---- Named destinations -------------------------------------------------

func TestSetNamedDestination(t *testing.T) {
	j := newJS()
	name := j.SetNamedDestination("intro", 0, 0, 50)
	if name != "intro" {
		t.Fatalf("name: %q", name)
	}
	dests := j.NamedDests()
	if len(dests) != 1 {
		t.Fatalf("expected 1 dest, got %d", len(dests))
	}
}

func TestSetNamedDestinationDeduplicate(t *testing.T) {
	j := newJS()
	j.SetNamedDestination("page1", 0, 0, 0)
	j.SetNamedDestination("page1", 1, 0, 0) // duplicate
	if len(j.NamedDests()) != 1 {
		t.Fatal("duplicate named dest should not be added")
	}
}

// ---- XObject templates --------------------------------------------------

func TestNewXObjectTemplate(t *testing.T) {
	j := newJS()
	tid := j.NewXObjectTemplate(200, 30, "")
	if tid == "" {
		t.Fatal("template ID should not be empty")
	}
	if len(j.XObjects()) != 1 {
		t.Fatalf("expected 1 XObject, got %d", len(j.XObjects()))
	}
}

func TestAddXObjectContent(t *testing.T) {
	j := newJS()
	tid := j.NewXObjectTemplate(100, 20, "")
	j.AddXObjectContent(tid, "q 1 0 0 1 10 10 cm Q\n")
	xobj := j.XObjects()[tid]
	if !strings.Contains(xobj.Content, "q") {
		t.Fatal("content not added")
	}
}

func TestExitXObjectTemplate(t *testing.T) {
	j := newJS()
	j.NewXObjectTemplate(100, 20, "")
	j.ExitXObjectTemplate()
	// Should not panic
}

func TestGetXObjectTemplate(t *testing.T) {
	j := newJS()
	tid := j.NewXObjectTemplate(100, 50, "")
	j.XObjects()[tid].ObjNum = 42
	ops := j.GetXObjectTemplate(tid, 10, 10, 100, 50, "T", "L")
	if !strings.Contains(ops, "Do") {
		t.Fatalf("GetXObjectTemplate should contain 'Do': %q", ops)
	}
}

func TestGetXObjectTemplateUnknown(t *testing.T) {
	j := newJS()
	ops := j.GetXObjectTemplate("NOPE", 0, 0, 100, 50, "", "")
	if ops != "" {
		t.Fatal("unknown template should return empty string")
	}
}

// ---- Embedded files -----------------------------------------------------

func TestAddEmbeddedFile(t *testing.T) {
	j := newJS()
	j.AddEmbeddedFile("readme.txt", []byte("hello"), "text/plain", "Data", "")
	files := j.EmbeddedFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 embedded file, got %d", len(files))
	}
	if files[0].Name != "readme.txt" {
		t.Fatalf("name: %q", files[0].Name)
	}
}

func TestAddEmbeddedFileDeduplicate(t *testing.T) {
	j := newJS()
	j.AddEmbeddedFile("file.txt", []byte("a"), "text/plain", "", "")
	j.AddEmbeddedFile("file.txt", []byte("b"), "text/plain", "", "")
	if len(j.EmbeddedFiles()) != 1 {
		t.Fatal("duplicate embedded file should not be added")
	}
}

// ---- Form fields --------------------------------------------------------

func TestAddFFText(t *testing.T) {
	j := newJS()
	id := j.AddFFText("name", 0, 10, 20, 80, 8, DefaultAnnotOpts(), nil)
	if id <= 0 {
		t.Fatalf("form field ID: %d", id)
	}
	fields := j.FormFields()
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Type != "Tx" {
		t.Fatalf("field type: %q", fields[0].Type)
	}
}

func TestAddFFCheckBox(t *testing.T) {
	j := newJS()
	j.AddFFCheckBox("agree", 0, 20, 30, 5, "Yes", true, DefaultAnnotOpts(), nil)
	if len(j.FormFields()) != 1 {
		t.Fatal("expected 1 checkbox field")
	}
}

func TestAddFFComboBox(t *testing.T) {
	j := newJS()
	j.AddFFComboBox("color", 0, 20, 40, 60, 8,
		[]interface{}{"Red", "Green", "Blue"},
		DefaultAnnotOpts(), nil)
	fields := j.FormFields()
	if len(fields) != 1 {
		t.Fatal("expected 1 combobox")
	}
	if fields[0].Type != "Ch" {
		t.Fatalf("combo type: %q", fields[0].Type)
	}
}

// ---- AnnotDict helper ---------------------------------------------------

func TestAnnotDict(t *testing.T) {
	a := &Annotation{
		Opts: AnnotOpts{Subtype: "Text", Contents: "Note"},
		Text: "Hello",
	}
	dict := AnnotDict(a, "10 20 60 50")
	if !strings.Contains(dict, "/Annot") {
		t.Fatal("missing /Annot")
	}
	if !strings.Contains(dict, "Text") {
		t.Fatal("missing Subtype Text")
	}
	if !strings.Contains(dict, "Hello") {
		t.Fatal("missing text content")
	}
}

// ---- PON ordering -------------------------------------------------------

func TestIDsAreIncreasing(t *testing.T) {
	j := newJS()
	id1 := j.SetLink(0, 0, 0, 10, 10, "a")
	id2 := j.SetLink(0, 0, 0, 10, 10, "b")
	id3 := j.SetAnnotation(0, 0, 0, 10, 10, "c", DefaultAnnotOpts())
	if id1 >= id2 || id2 >= id3 {
		t.Fatalf("IDs should be increasing: %d %d %d", id1, id2, id3)
	}
}
