// Package javascript provides PDF interactive features: annotations, form fields,
// bookmarks, XObject templates, JavaScript, named destinations, embedded files.
package javascript

import (
	"strings"
	"sync"

	"github.com/tecnickcom/go-tcpdf/base"
	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

// ---- Types ---------------------------------------------------------------

type AnnotOpts struct {
	Subtype  string
	Contents string
	Name     string
	Flags    int
	Color    [3]float64
	Extra    map[string]string
}

func DefaultAnnotOpts() AnnotOpts { return AnnotOpts{Subtype: "Text"} }

type Annotation struct {
	ObjNum              int
	Page                int
	PosX, PosY          float64
	Width, Height       float64
	Text                string
	Opts                AnnotOpts
}

type Bookmark struct {
	Name, Link          string
	Level, Page         int
	PosX, PosY          float64
	FStyle, Color       string
}

type NamedDest struct {
	Name        string
	Page        int
	PosX, PosY  float64
}

type XObjectTemplate struct {
	ObjNum      int
	Width       float64
	Height      float64
	Content     string
	Fonts       map[string]bool
	Images      map[int]bool
	XObjects    map[string]bool
	Gradients   map[int]bool
	ExtGStates  map[int]bool
	SpotColors  map[string]bool
	TranspGroup string
}

type EmbeddedFile struct {
	ObjNum  int
	Name    string
	Content []byte
	MIME    string
	AFRel   string
	Desc    string
}

type FormField struct {
	ObjNum          int
	Name            string
	Page            int
	PosX, PosY      float64
	Width, Height   float64
	Type            string
	Opts            AnnotOpts
	JSProp          map[string]string
}

type jsObject struct {
	ObjNum int
	Script string
	OnLoad bool
}

type link struct {
	ObjNum          int
	Page            int
	PosX, PosY      float64
	Width, Height   float64
	URL             string
}

// JavaScript manages all interactive PDF elements. Thread-safe.
type JavaScript struct {
	mu sync.RWMutex

	rawJS     string
	jsObjects []jsObject

	annotations []*Annotation
	defAnnotOpt AnnotOpts

	links       []link

	bookmarks   []Bookmark

	namedDests  []NamedDest

	xobjects    map[string]*XObjectTemplate

	embeddedFiles  []*EmbeddedFile
	embeddedByName map[string]bool

	formFields []*FormField

	pon *base.PON
}

func New(pon *base.PON) *JavaScript {
	return &JavaScript{
		pon:            pon,
		defAnnotOpt:    DefaultAnnotOpts(),
		xobjects:       make(map[string]*XObjectTemplate),
		embeddedByName: make(map[string]bool),
	}
}

// ---- Raw JavaScript -----------------------------------------------------

func (j *JavaScript) AppendRawJavaScript(script string) {
	j.mu.Lock()
	j.rawJS += script
	j.mu.Unlock()
}

func (j *JavaScript) AddRawJavaScriptObj(script string, onLoad bool) int {
	n := j.pon.Next()
	j.mu.Lock()
	j.jsObjects = append(j.jsObjects, jsObject{ObjNum: n, Script: script, OnLoad: onLoad})
	j.mu.Unlock()
	return n
}

func (j *JavaScript) RawJS() string {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.rawJS
}

func (j *JavaScript) JSObjects() []jsObject {
	j.mu.RLock()
	defer j.mu.RUnlock()
	cp := make([]jsObject, len(j.jsObjects))
	copy(cp, j.jsObjects)
	return cp
}

// ---- Annotations --------------------------------------------------------

func (j *JavaScript) SetDefJSAnnotProp(opts AnnotOpts) {
	j.mu.Lock()
	j.defAnnotOpt = opts
	j.mu.Unlock()
}

func (j *JavaScript) GetDefJSAnnotProp() AnnotOpts {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.defAnnotOpt
}

func (j *JavaScript) SetAnnotation(page int, posX, posY, width, height float64, txt string, opts AnnotOpts) int {
	n := j.pon.Next()
	j.mu.Lock()
	j.annotations = append(j.annotations, &Annotation{
		ObjNum: n, Page: page,
		PosX: posX, PosY: posY, Width: width, Height: height,
		Text: txt, Opts: opts,
	})
	j.mu.Unlock()
	return n
}

func (j *JavaScript) Annotations() []*Annotation {
	j.mu.RLock()
	defer j.mu.RUnlock()
	cp := make([]*Annotation, len(j.annotations))
	copy(cp, j.annotations)
	return cp
}

// ---- Links --------------------------------------------------------------

func (j *JavaScript) SetLink(page int, posX, posY, width, height float64, url string) int {
	n := j.pon.Next()
	j.mu.Lock()
	j.links = append(j.links, link{ObjNum: n, Page: page, PosX: posX, PosY: posY, Width: width, Height: height, URL: url})
	j.mu.Unlock()
	return n
}

func (j *JavaScript) AddInternalLink(page int, posY float64) string {
	var b strings.Builder
	b.WriteByte('#')
	b.WriteString(pdfbuf.FmtI(page))
	b.WriteByte(',')
	b.WriteString(pdfbuf.FmtF(posY))
	return b.String()
}

func (j *JavaScript) Links() []link {
	j.mu.RLock()
	defer j.mu.RUnlock()
	cp := make([]link, len(j.links))
	copy(cp, j.links)
	return cp
}

// ---- Bookmarks ----------------------------------------------------------

func (j *JavaScript) SetBookmark(name, lnk string, level, page int, posX, posY float64, fstyle, clr string) {
	j.mu.Lock()
	j.bookmarks = append(j.bookmarks, Bookmark{Name: name, Link: lnk, Level: level, Page: page, PosX: posX, PosY: posY, FStyle: fstyle, Color: clr})
	j.mu.Unlock()
}

func (j *JavaScript) Bookmarks() []Bookmark {
	j.mu.RLock()
	defer j.mu.RUnlock()
	cp := make([]Bookmark, len(j.bookmarks))
	copy(cp, j.bookmarks)
	return cp
}

// ---- Named destinations -------------------------------------------------

func (j *JavaScript) SetNamedDestination(name string, page int, posX, posY float64) string {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, nd := range j.namedDests {
		if nd.Name == name {
			return name
		}
	}
	j.namedDests = append(j.namedDests, NamedDest{Name: name, Page: page, PosX: posX, PosY: posY})
	return name
}

func (j *JavaScript) NamedDests() []NamedDest {
	j.mu.RLock()
	defer j.mu.RUnlock()
	cp := make([]NamedDest, len(j.namedDests))
	copy(cp, j.namedDests)
	return cp
}

// ---- XObject templates --------------------------------------------------

func (j *JavaScript) NewXObjectTemplate(width, height float64, transpGroup string) string {
	n := j.pon.Next()
	var tb strings.Builder
	tb.WriteString("TPL")
	tb.WriteString(pdfbuf.FmtI(n))
	tid := tb.String()

	xobj := &XObjectTemplate{
		ObjNum: n, Width: width, Height: height, TranspGroup: transpGroup,
		Fonts: make(map[string]bool), Images: make(map[int]bool),
		XObjects: make(map[string]bool), Gradients: make(map[int]bool),
		ExtGStates: make(map[int]bool), SpotColors: make(map[string]bool),
	}
	j.mu.Lock()
	j.xobjects[tid] = xobj
	j.mu.Unlock()
	return tid
}

func (j *JavaScript) ExitXObjectTemplate() {}

func (j *JavaScript) AddXObjectContent(tid, data string) {
	j.mu.Lock()
	if xobj, ok := j.xobjects[tid]; ok {
		xobj.Content += data
	}
	j.mu.Unlock()
}

func (j *JavaScript) AddXObjectFontID(tid, key string) {
	j.mu.Lock()
	if xobj, ok := j.xobjects[tid]; ok {
		xobj.Fonts[key] = true
	}
	j.mu.Unlock()
}

func (j *JavaScript) AddXObjectImageID(tid string, key int) {
	j.mu.Lock()
	if xobj, ok := j.xobjects[tid]; ok {
		xobj.Images[key] = true
	}
	j.mu.Unlock()
}

func (j *JavaScript) AddXObjectXObjectID(tid, key string) {
	j.mu.Lock()
	if xobj, ok := j.xobjects[tid]; ok {
		xobj.XObjects[key] = true
	}
	j.mu.Unlock()
}

func (j *JavaScript) AddXObjectGradientID(tid string, key int) {
	j.mu.Lock()
	if xobj, ok := j.xobjects[tid]; ok {
		xobj.Gradients[key] = true
	}
	j.mu.Unlock()
}

func (j *JavaScript) AddXObjectExtGStateID(tid string, key int) {
	j.mu.Lock()
	if xobj, ok := j.xobjects[tid]; ok {
		xobj.ExtGStates[key] = true
	}
	j.mu.Unlock()
}

func (j *JavaScript) AddXObjectSpotColorID(tid, key string) {
	j.mu.Lock()
	if xobj, ok := j.xobjects[tid]; ok {
		xobj.SpotColors[key] = true
	}
	j.mu.Unlock()
}

func (j *JavaScript) GetXObjectTemplate(tid string, posX, posY, width, height float64, valign, halign string) string {
	j.mu.RLock()
	xobj, ok := j.xobjects[tid]
	j.mu.RUnlock()
	if !ok {
		return ""
	}
	scaleX, scaleY := 1.0, 1.0
	if xobj.Width > 0 && width > 0 {
		scaleX = width / xobj.Width
	}
	if xobj.Height > 0 && height > 0 {
		scaleY = height / xobj.Height
	}
	tx, ty := posX, posY
	switch valign {
	case "C":
		ty += (height - xobj.Height*scaleY) / 2
	case "B":
		ty += height - xobj.Height*scaleY
	}
	switch halign {
	case "C":
		tx += (width - xobj.Width*scaleX) / 2
	case "R":
		tx += width - xobj.Width*scaleX
	}
	var gb strings.Builder
	gb.WriteString("q ")
	gb.WriteString(pdfbuf.FmtF(scaleX)); gb.WriteString(" 0 0 ")
	gb.WriteString(pdfbuf.FmtF(scaleY)); gb.WriteByte(' ')
	gb.WriteString(pdfbuf.FmtF(tx)); gb.WriteByte(' ')
	gb.WriteString(pdfbuf.FmtF(ty))
	gb.WriteString(" cm /XObj"); gb.WriteString(pdfbuf.FmtI(xobj.ObjNum)); gb.WriteString(" Do Q\n")
	return gb.String()
}

func (j *JavaScript) XObjects() map[string]*XObjectTemplate {
	j.mu.RLock()
	defer j.mu.RUnlock()
	cp := make(map[string]*XObjectTemplate, len(j.xobjects))
	for k, v := range j.xobjects {
		cp[k] = v
	}
	return cp
}

// ---- Embedded files -----------------------------------------------------

func (j *JavaScript) AddEmbeddedFile(name string, data []byte, mime, afrel, desc string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.embeddedByName[name] {
		return
	}
	n := j.pon.Next()
	j.embeddedFiles = append(j.embeddedFiles, &EmbeddedFile{ObjNum: n, Name: name, Content: data, MIME: mime, AFRel: afrel, Desc: desc})
	j.embeddedByName[name] = true
}

func (j *JavaScript) EmbeddedFiles() []*EmbeddedFile {
	j.mu.RLock()
	defer j.mu.RUnlock()
	cp := make([]*EmbeddedFile, len(j.embeddedFiles))
	copy(cp, j.embeddedFiles)
	return cp
}

// ---- Form fields --------------------------------------------------------

func (j *JavaScript) AddFFButton(name string, page int, posX, posY, width, height float64, caption string, action interface{}, opts AnnotOpts, jsp map[string]string) int {
	return j.addFormField("Btn", name, page, posX, posY, width, height, opts, jsp)
}

func (j *JavaScript) AddFFCheckBox(name string, page int, posX, posY, width float64, onVal string, checked bool, opts AnnotOpts, jsp map[string]string) int {
	return j.addFormField("Btn", name, page, posX, posY, width, width, opts, jsp)
}

func (j *JavaScript) AddFFRadioButton(name string, page int, posX, posY, width float64, onVal string, checked bool, opts AnnotOpts, jsp map[string]string) int {
	return j.addFormField("Btn", name, page, posX, posY, width, width, opts, jsp)
}

func (j *JavaScript) AddFFText(name string, page int, posX, posY, width, height float64, opts AnnotOpts, jsp map[string]string) int {
	return j.addFormField("Tx", name, page, posX, posY, width, height, opts, jsp)
}

func (j *JavaScript) AddFFComboBox(name string, page int, posX, posY, width, height float64, values []interface{}, opts AnnotOpts, jsp map[string]string) int {
	return j.addFormField("Ch", name, page, posX, posY, width, height, opts, jsp)
}

func (j *JavaScript) AddFFListBox(name string, page int, posX, posY, width, height float64, values []interface{}, opts AnnotOpts, jsp map[string]string) int {
	return j.addFormField("Ch", name, page, posX, posY, width, height, opts, jsp)
}

func (j *JavaScript) AddJSButton(name string, page int, posX, posY, width, height float64, caption, action string, jsp map[string]string) {
	j.AddFFButton(name, page, posX, posY, width, height, caption, action, DefaultAnnotOpts(), jsp)
}

func (j *JavaScript) AddJSCheckBox(name string, page int, posX, posY, width float64, jsp map[string]string) {
	j.AddFFCheckBox(name, page, posX, posY, width, "", false, DefaultAnnotOpts(), jsp)
}

func (j *JavaScript) AddJSComboBox(name string, page int, posX, posY, width, height float64, values []interface{}, jsp map[string]string) {
	j.AddFFComboBox(name, page, posX, posY, width, height, values, DefaultAnnotOpts(), jsp)
}

func (j *JavaScript) AddJSListBox(name string, page int, posX, posY, width, height float64, values []interface{}, jsp map[string]string) {
	j.AddFFListBox(name, page, posX, posY, width, height, values, DefaultAnnotOpts(), jsp)
}

func (j *JavaScript) AddJSRadioButton(name string, page int, posX, posY, width float64, jsp map[string]string) {
	j.AddFFRadioButton(name, page, posX, posY, width, "", false, DefaultAnnotOpts(), jsp)
}

func (j *JavaScript) AddJSText(name string, page int, posX, posY, width, height float64, jsp map[string]string) {
	j.AddFFText(name, page, posX, posY, width, height, DefaultAnnotOpts(), jsp)
}

func (j *JavaScript) FormFields() []*FormField {
	j.mu.RLock()
	defer j.mu.RUnlock()
	cp := make([]*FormField, len(j.formFields))
	copy(cp, j.formFields)
	return cp
}

func (j *JavaScript) addFormField(fieldType, name string, page int, posX, posY, width, height float64, opts AnnotOpts, jsp map[string]string) int {
	n := j.pon.Next()
	j.mu.Lock()
	j.formFields = append(j.formFields, &FormField{ObjNum: n, Name: name, Page: page, PosX: posX, PosY: posY, Width: width, Height: height, Type: fieldType, Opts: opts, JSProp: jsp})
	j.mu.Unlock()
	return n
}

// ---- PDF output helper --------------------------------------------------

// AnnotDict returns the PDF dictionary string for an annotation.
func AnnotDict(a *Annotation, rectStr string) string {
	var sb strings.Builder
	sb.WriteString("<< /Type /Annot /Subtype /")
	sb.WriteString(a.Opts.Subtype)
	sb.WriteString("\n/Rect [")
	sb.WriteString(rectStr)
	sb.WriteString("]\n")
	if a.Text != "" {
		sb.WriteString("/Contents (")
		sb.WriteString(a.Text)
		sb.WriteString(")\n")
	}
	if a.Opts.Name != "" {
		sb.WriteString("/NM (")
		sb.WriteString(a.Opts.Name)
		sb.WriteString(")\n")
	}
	if a.Opts.Flags != 0 {
		sb.WriteString("/F ")
		sb.WriteString(pdfbuf.FmtI(a.Opts.Flags))
		sb.WriteByte('\n')
	}
	for k, v := range a.Opts.Extra {
		sb.WriteByte('/')
		sb.WriteString(k)
		sb.WriteByte(' ')
		sb.WriteString(v)
		sb.WriteByte('\n')
	}
	sb.WriteString(">>")
	return sb.String()
}
