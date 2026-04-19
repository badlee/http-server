// Package layers provides PDF Optional Content Groups (layers).
package layers

import (
	"strings"
	"sync"

	"github.com/tecnickcom/go-tcpdf/base"
	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

type Layer struct {
	ObjNum  int
	Name    string
	Visible bool
}

// Layers manages document layers. Thread-safe.
type Layers struct {
	mu     sync.RWMutex
	layers []*Layer
	stack  []int
	pon    *base.PON
}

func New(pon *base.PON) *Layers {
	return &Layers{pon: pon}
}

// NewLayer creates and registers a new layer. Returns the layer index.
func (l *Layers) NewLayer(name string, visible bool) int {
	n := l.pon.Next()
	l.mu.Lock()
	idx := len(l.layers)
	l.layers = append(l.layers, &Layer{ObjNum: n, Name: name, Visible: visible})
	l.mu.Unlock()
	return idx
}

// OpenLayer emits the PDF operators to begin a layer.
func (l *Layers) OpenLayer(layerID int) string {
	l.mu.RLock()
	if layerID < 0 || layerID >= len(l.layers) {
		l.mu.RUnlock()
		return ""
	}
	layer := l.layers[layerID]
	l.mu.RUnlock()

	l.mu.Lock()
	l.stack = append(l.stack, layerID)
	l.mu.Unlock()

	var b strings.Builder
	b.WriteString("/OC /OC")
	b.WriteString(pdfbuf.FmtI(layer.ObjNum))
	b.WriteString(" BDC\n")
	return b.String()
}

// CloseLayer emits the PDF operator to end the current layer.
func (l *Layers) CloseLayer() string {
	l.mu.Lock()
	if len(l.stack) > 0 {
		l.stack = l.stack[:len(l.stack)-1]
	}
	l.mu.Unlock()
	return "EMC\n"
}

// All returns a snapshot of all layers.
func (l *Layers) All() []*Layer {
	l.mu.RLock()
	defer l.mu.RUnlock()
	cp := make([]*Layer, len(l.layers))
	copy(cp, l.layers)
	return cp
}

// OCPropertiesDict returns the /OCProperties dictionary for the PDF catalog.
func (l *Layers) OCPropertiesDict() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.layers) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("/OCProperties <<\n/OCGs [")
	for _, layer := range l.layers {
		sb.WriteString(pdfbuf.FmtI(layer.ObjNum))
		sb.WriteString(" 0 R ")
	}
	sb.WriteString("]\n/D << /BaseState /ON /OFF [")
	for _, layer := range l.layers {
		if !layer.Visible {
			sb.WriteString(pdfbuf.FmtI(layer.ObjNum))
			sb.WriteString(" 0 R ")
		}
	}
	sb.WriteString("] >>\n>>\n")
	return sb.String()
}

// OCGDict returns the OCG dictionary for a single layer.
func (l *Layers) OCGDict(layer *Layer) string {
	var b strings.Builder
	b.WriteString("<< /Type /OCG /Name (")
	b.WriteString(layer.Name)
	b.WriteString(") /Usage << /Print << /PrintState /")
	b.WriteString(visState(layer.Visible))
	b.WriteString(" >> >> >>\n")
	return b.String()
}

func visState(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}
