// Package toc provides automatic Table of Contents generation.
package toc

import (
	"strings"
	"sync"

	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

type TOCEntry struct {
	Level    int
	Title    string
	Page     int
	PosY     float64
	Indent   float64
	FontKey  string
	FontSize float64
	Style    string
}

type TOCStyle struct {
	FontFamily string
	FontSize   float64
	Indent     float64
	LineSpace  float64
	DotLeader  bool
	DotChar    string
}

func DefaultTOCStyle() TOCStyle {
	return TOCStyle{FontFamily: "Helvetica", FontSize: 10, Indent: 5, LineSpace: 2, DotLeader: true, DotChar: "."}
}

// TOC manages table-of-contents entries. Thread-safe.
type TOC struct {
	mu      sync.RWMutex
	entries []TOCEntry
	style   TOCStyle
}

func New() *TOC { return &TOC{style: DefaultTOCStyle()} }

func (t *TOC) SetStyle(s TOCStyle) {
	t.mu.Lock()
	t.style = s
	t.mu.Unlock()
}

func (t *TOC) Style() TOCStyle {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.style
}

func (t *TOC) AddEntry(level int, title string, page int, posY float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries = append(t.entries, TOCEntry{
		Level:    level,
		Title:    title,
		Page:     page,
		PosY:     posY,
		Indent:   float64(level) * t.style.Indent,
		FontKey:  t.style.FontFamily,
		FontSize: t.style.FontSize,
	})
}

// AddTOC returns formatted TOC line strings.
func (t *TOC) AddTOC() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	lines := make([]string, 0, len(t.entries))
	for _, e := range t.entries {
		var lb strings.Builder
		lb.WriteString(strings.Repeat(" ", e.Level*2))
		lb.WriteString(e.Title)
		lb.WriteString(" .... ")
		lb.WriteString(pdfbuf.FmtI(e.Page))
		lines = append(lines, lb.String())
	}
	return lines
}

// Entries returns a snapshot of all entries.
func (t *TOC) Entries() []TOCEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	cp := make([]TOCEntry, len(t.entries))
	copy(cp, t.entries)
	return cp
}

func (t *TOC) Len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}

func (t *TOC) Clear() {
	t.mu.Lock()
	t.entries = nil
	t.mu.Unlock()
}
