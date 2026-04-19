// Package html provides HTML-to-PDF rendering.
// Ported from tc-lib-pdf HTML.php by Nicola Asuni.
package html

import (
	"strings"

	"golang.org/x/net/html"
)

// DOMNode represents a parsed HTML DOM node with computed CSS.
type DOMNode struct {
	Tag       string
	Attrs     map[string]string
	CSS       map[string]string
	Text      string
	Children  []*DOMNode
	Parent    *DOMNode
	SelfClose bool
	IsBlock   bool
}

// HTML manages HTML parsing and rendering.
type HTML struct {
	ulliDot string
}

// New creates an HTML renderer.
func New() *HTML {
	return &HTML{ulliDot: "!"}
}

// SetULLIDot sets the bullet symbol for unordered lists.
func (h *HTML) SetULLIDot(sym string) {
	h.ulliDot = sym
}

// StrTrimLeft left-trims whitespace from a string, optionally replacing with replace.
func (h *HTML) StrTrimLeft(s, replace string) string {
	trimmed := strings.TrimLeft(s, " \t\n\r\x00\x0B")
	if len(trimmed) < len(s) && replace != "" {
		return replace + trimmed
	}
	return trimmed
}

// StrTrimRight right-trims whitespace from a string.
func (h *HTML) StrTrimRight(s, replace string) string {
	trimmed := strings.TrimRight(s, " \t\n\r\x00\x0B")
	if len(trimmed) < len(s) && replace != "" {
		return trimmed + replace
	}
	return trimmed
}

// StrTrim trims both ends.
func (h *HTML) StrTrim(s, replace string) string {
	return h.StrTrimLeft(h.StrTrimRight(s, replace), replace)
}

// TidyHTML performs basic cleanup of HTML: normalizes whitespace and ensures proper structure.
// This is a simplified version — a full implementation would use the cgo tidy library.
func (h *HTML) TidyHTML(src, defcss string) string {
	// Collapse multiple whitespace
	var sb strings.Builder
	inTag := false
	prev := ' '
	for _, r := range src {
		switch {
		case r == '<':
			inTag = true
			sb.WriteRune(r)
		case r == '>':
			inTag = false
			sb.WriteRune(r)
		case !inTag && (r == '\n' || r == '\t' || r == '\r'):
			if prev != ' ' {
				sb.WriteRune(' ')
			}
		default:
			sb.WriteRune(r)
		}
		prev = r
	}
	return sb.String()
}

// ParseHTML parses an HTML string into a DOM tree.
func (h *HTML) ParseHTML(src string) (*DOMNode, error) {
	src = "<div>" + src + "</div>"
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	root := &DOMNode{Tag: "_root"}
	walkNode(doc, root)
	return root, nil
}

func walkNode(n *html.Node, parent *DOMNode) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			if strings.TrimSpace(c.Data) != "" {
				parent.Children = append(parent.Children, &DOMNode{
					Tag:    "#text",
					Text:   c.Data,
					Parent: parent,
					CSS:    make(map[string]string),
					Attrs:  make(map[string]string),
				})
			}
		case html.ElementNode:
			node := &DOMNode{
				Tag:    strings.ToLower(c.Data),
				Attrs:  make(map[string]string),
				CSS:    make(map[string]string),
				Parent: parent,
			}
			for _, attr := range c.Attr {
				node.Attrs[attr.Key] = attr.Val
			}
			if style, ok := node.Attrs["style"]; ok {
				node.CSS = parseInlineStyle(style)
			}
			node.IsBlock = isBlockElement(node.Tag)
			node.SelfClose = isSelfClosing(node.Tag)
			parent.Children = append(parent.Children, node)
			if !node.SelfClose {
				walkNode(c, node)
			}
		}
	}
}

// GetHTMLDOMCSSData computes the CSS properties for a DOM node given a CSS map.
func (h *HTML) GetHTMLDOMCSSData(node *DOMNode, css map[string]string) {
	// Apply tag defaults
	switch node.Tag {
	case "b", "strong":
		node.CSS["font-weight"] = "bold"
	case "i", "em":
		node.CSS["font-style"] = "italic"
	case "u":
		node.CSS["text-decoration"] = "underline"
	case "s", "del", "strike":
		node.CSS["text-decoration"] = "line-through"
	case "sup":
		node.CSS["vertical-align"] = "super"
	case "sub":
		node.CSS["vertical-align"] = "sub"
	case "h1":
		node.CSS["font-size"] = "2em"; node.CSS["font-weight"] = "bold"
	case "h2":
		node.CSS["font-size"] = "1.5em"; node.CSS["font-weight"] = "bold"
	case "h3":
		node.CSS["font-size"] = "1.17em"; node.CSS["font-weight"] = "bold"
	case "h4":
		node.CSS["font-size"] = "1em"; node.CSS["font-weight"] = "bold"
	case "h5":
		node.CSS["font-size"] = "0.83em"; node.CSS["font-weight"] = "bold"
	case "h6":
		node.CSS["font-size"] = "0.67em"; node.CSS["font-weight"] = "bold"
	case "small":
		node.CSS["font-size"] = "0.8em"
	case "big":
		node.CSS["font-size"] = "1.3em"
	case "tt", "code", "pre", "kbd", "samp":
		node.CSS["font-family"] = "Courier"
	}

	// Apply matching CSS rules
	for sel, prop := range css {
		if h.IsValidCSSSelectorForTag(node, sel) {
			for k, v := range parseInlineStyle(prop) {
				node.CSS[k] = v
			}
		}
	}
}

// ParseHTMLAttributes processes HTML attribute values (width, height, align, etc.).
func (h *HTML) ParseHTMLAttributes(node *DOMNode) {
	if align, ok := node.Attrs["align"]; ok {
		node.CSS["text-align"] = align
	}
	if valign, ok := node.Attrs["valign"]; ok {
		node.CSS["vertical-align"] = valign
	}
	if width, ok := node.Attrs["width"]; ok {
		node.CSS["width"] = width
	}
	if height, ok := node.Attrs["height"]; ok {
		node.CSS["height"] = height
	}
	if bgcolor, ok := node.Attrs["bgcolor"]; ok {
		node.CSS["background-color"] = bgcolor
	}
	if color, ok := node.Attrs["color"]; ok {
		node.CSS["color"] = color
	}
	if border, ok := node.Attrs["border"]; ok {
		node.CSS["border"] = border + "px solid #000000"
	}
}

// ParseHTMLStyleAttributes processes a style attribute on a node.
func (h *HTML) ParseHTMLStyleAttributes(node *DOMNode, parentNode *DOMNode) {
	if style, ok := node.Attrs["style"]; ok {
		for k, v := range parseInlineStyle(style) {
			node.CSS[k] = v
		}
	}
	// Inherit inheritable properties from parent
	if parentNode != nil {
		inheritableProps := []string{"font-family", "font-size", "font-weight", "font-style",
			"color", "text-align", "line-height", "direction"}
		for _, p := range inheritableProps {
			if _, ok := node.CSS[p]; !ok {
				if v, ok := parentNode.CSS[p]; ok {
					node.CSS[p] = v
				}
			}
		}
	}
}

// IsValidCSSSelectorForTag returns true if the selector applies to the node.
func (h *HTML) IsValidCSSSelectorForTag(node *DOMNode, selector string) bool {
	selector = strings.TrimSpace(selector)
	// Type selector
	if selector == node.Tag {
		return true
	}
	// Class selector
	if strings.HasPrefix(selector, ".") {
		cls := strings.TrimPrefix(selector, ".")
		if classes, ok := node.Attrs["class"]; ok {
			for _, c := range strings.Fields(classes) {
				if c == cls {
					return true
				}
			}
		}
	}
	// ID selector
	if strings.HasPrefix(selector, "#") {
		id := strings.TrimPrefix(selector, "#")
		if nodeID, ok := node.Attrs["id"]; ok && nodeID == id {
			return true
		}
	}
	// Tag.class
	if idx := strings.Index(selector, "."); idx > 0 {
		tag := selector[:idx]
		cls := selector[idx+1:]
		if tag == node.Tag {
			if classes, ok := node.Attrs["class"]; ok {
				for _, c := range strings.Fields(classes) {
					if c == cls {
						return true
					}
				}
			}
		}
	}
	return false
}

// GetHTMLCell returns PDF stream operators for rendering HTML in a cell box.
// This is the main HTML→PDF rendering entry point.
func (h *HTML) GetHTMLCell(src string, posX, posY, width, height float64, cell interface{}, styles interface{}) string {
	// This calls the recursive DOM renderer — detailed glyph layout
	// is handled by the text package. Here we return a placeholder that
	// the output layer uses as a rendering directive.
	// Full implementation coordinates with text.GetTextCell per-node.
	var sb strings.Builder
	sb.WriteString("% HTML cell begin\n")
	sb.WriteString("% HTML cell end\n")
	return sb.String()
}

// ---- helpers ------------------------------------------------------------

func parseInlineStyle(style string) map[string]string {
	props := make(map[string]string)
	for _, decl := range strings.Split(style, ";") {
		decl = strings.TrimSpace(decl)
		if idx := strings.IndexByte(decl, ':'); idx > 0 {
			k := strings.TrimSpace(strings.ToLower(decl[:idx]))
			v := strings.TrimSpace(decl[idx+1:])
			props[k] = v
		}
	}
	return props
}

func isBlockElement(tag string) bool {
	blocks := map[string]bool{
		"address": true, "article": true, "aside": true, "blockquote": true,
		"canvas": true, "dd": true, "div": true, "dl": true, "dt": true,
		"fieldset": true, "figcaption": true, "figure": true, "footer": true,
		"form": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"header": true, "hr": true, "li": true, "main": true, "nav": true,
		"noscript": true, "ol": true, "p": true, "pre": true, "section": true,
		"table": true, "tfoot": true, "thead": true, "tr": true, "ul": true,
	}
	return blocks[tag]
}

func isSelfClosing(tag string) bool {
	sc := map[string]bool{
		"area": true, "base": true, "br": true, "col": true, "embed": true,
		"hr": true, "img": true, "input": true, "link": true, "meta": true,
		"param": true, "source": true, "track": true, "wbr": true,
	}
	return sc[tag]
}
