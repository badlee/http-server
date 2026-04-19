package metainfo

import (
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New()
	if m.PDFVersion != "1.7" {
		t.Fatalf("default version: %q", m.PDFVersion)
	}
	if m.Producer == "" {
		t.Fatal("producer should not be empty")
	}
	if m.CreationDate.IsZero() {
		t.Fatal("creation date should be set")
	}
}

func TestSetters(t *testing.T) {
	m := New()
	m.SetTitle("My Title").
		SetAuthor("Author Name").
		SetSubject("Subject").
		SetKeywords("key1 key2").
		SetCreator("MyApp")

	if m.Title != "My Title" {
		t.Fatalf("title: %q", m.Title)
	}
	if m.Author != "Author Name" {
		t.Fatalf("author: %q", m.Author)
	}
	if m.Subject != "Subject" {
		t.Fatalf("subject: %q", m.Subject)
	}
	if m.Keywords != "key1 key2" {
		t.Fatalf("keywords: %q", m.Keywords)
	}
	if m.Creator != "MyApp" {
		t.Fatalf("creator: %q", m.Creator)
	}
}

func TestSetPDFVersion(t *testing.T) {
	m := New()
	for _, v := range []string{"1.0", "1.3", "1.4", "1.5", "1.6", "1.7", "2.0"} {
		if _, err := m.SetPDFVersion(v); err != nil {
			t.Errorf("SetPDFVersion(%q): %v", v, err)
		}
		if m.PDFVersion != v {
			t.Errorf("version not set: %q", m.PDFVersion)
		}
	}
}

func TestSetPDFVersionInvalid(t *testing.T) {
	m := New()
	if _, err := m.SetPDFVersion("3.0"); err == nil {
		t.Fatal("expected error for version 3.0")
	}
	if _, err := m.SetPDFVersion(""); err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestSetSRGB(t *testing.T) {
	m := New()
	m.SetSRGB(true)
	if !m.SRGB {
		t.Fatal("SRGB should be true")
	}
	m.SetSRGB(false)
	if m.SRGB {
		t.Fatal("SRGB should be false")
	}
}

func TestSetCustomXMP(t *testing.T) {
	m := New()
	m.SetCustomXMP("mykey", "<custom/>")
	if m.CustomXMP["mykey"] != "<custom/>" {
		t.Fatalf("custom XMP: %v", m.CustomXMP)
	}
}

func TestSetViewerPreferences(t *testing.T) {
	m := New()
	m.SetViewerPreferences(ViewerPreferences{
		HideToolbar: true,
		NumCopies:   3,
	})
	if m.ViewerPrefs == nil {
		t.Fatal("ViewerPrefs should not be nil")
	}
	if !m.ViewerPrefs.HideToolbar {
		t.Fatal("HideToolbar should be true")
	}
	if m.ViewerPrefs.NumCopies != 3 {
		t.Fatalf("NumCopies: %d", m.ViewerPrefs.NumCopies)
	}
}

func TestInfoDictEntries(t *testing.T) {
	m := New()
	m.SetTitle("Test Doc").SetAuthor("Author")
	enc := func(s string) string { return s }
	dict := m.InfoDictEntries(enc)
	if !strings.Contains(dict, "/Title") {
		t.Fatal("dict missing /Title")
	}
	if !strings.Contains(dict, "Test Doc") {
		t.Fatal("dict missing title value")
	}
	if !strings.Contains(dict, "/Author") {
		t.Fatal("dict missing /Author")
	}
	if !strings.Contains(dict, "/CreationDate") {
		t.Fatal("dict missing /CreationDate")
	}
}

func TestInfoDictEntriesEmpty(t *testing.T) {
	m := New()
	m.Title = ""
	m.Author = ""
	enc := func(s string) string { return s }
	dict := m.InfoDictEntries(enc)
	if strings.Contains(dict, "/Title") {
		t.Fatal("empty title should not appear in dict")
	}
}

func TestViewerPrefDict(t *testing.T) {
	m := New()
	if m.ViewerPrefDict() != "" {
		t.Fatal("no prefs should produce empty dict")
	}
	m.SetViewerPreferences(ViewerPreferences{
		HideToolbar:  true,
		HideMenubar:  true,
		FitWindow:    true,
		Duplex:       "DuplexFlipLongEdge",
		PrintScaling: "None",
		NumCopies:    2,
	})
	dict := m.ViewerPrefDict()
	if !strings.Contains(dict, "HideToolbar") {
		t.Fatal("missing HideToolbar")
	}
	if !strings.Contains(dict, "Duplex") {
		t.Fatal("missing Duplex")
	}
	if !strings.Contains(dict, "NumCopies") {
		t.Fatal("missing NumCopies")
	}
}

func TestXMPData(t *testing.T) {
	m := New()
	m.SetTitle("XMP Test").SetAuthor("Writer")
	m.CreationDate = time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	xmp := m.XMPData()
	if !strings.Contains(xmp, "<?xpacket") {
		t.Fatal("XMP should start with xpacket")
	}
	if !strings.Contains(xmp, "XMP Test") {
		t.Fatal("XMP should contain title")
	}
	if !strings.Contains(xmp, "Writer") {
		t.Fatal("XMP should contain author")
	}
	if !strings.Contains(xmp, "<?xpacket end") {
		t.Fatal("XMP should end with closing xpacket")
	}
}

func TestXMPDataCustom(t *testing.T) {
	m := New()
	m.SetCustomXMP("custom", `<myns:Field>value</myns:Field>`)
	xmp := m.XMPData()
	if !strings.Contains(xmp, "value") {
		t.Fatal("XMP should contain custom fragment")
	}
}

func TestGetVersion(t *testing.T) {
	m := New()
	v := m.GetVersion()
	if v == "" {
		t.Fatal("version string should not be empty")
	}
}
