// Package classobjects initializes and wires all sub-objects (dependency injection).
// Ported from tc-lib-pdf ClassObjects.php by Nicola Asuni.
package classobjects

import (
	"github.com/tecnickcom/go-tcpdf/base"
	"github.com/tecnickcom/go-tcpdf/cell"
	"github.com/tecnickcom/go-tcpdf/css"
	"github.com/tecnickcom/go-tcpdf/html"
	"github.com/tecnickcom/go-tcpdf/javascript"
	"github.com/tecnickcom/go-tcpdf/layers"
	"github.com/tecnickcom/go-tcpdf/metainfo"
	"github.com/tecnickcom/go-tcpdf/svg"
	"github.com/tecnickcom/go-tcpdf/text"
	"github.com/tecnickcom/go-tcpdf/toc"
	"github.com/tecnickcom/go-tcpdf/cache"
	"github.com/tecnickcom/go-tcpdf/color"
	"github.com/tecnickcom/go-tcpdf/encrypt"
	"github.com/tecnickcom/go-tcpdf/font"
	"github.com/tecnickcom/go-tcpdf/graph"
	imgpkg "github.com/tecnickcom/go-tcpdf/image"
	"github.com/tecnickcom/go-tcpdf/page"
	uniconv "github.com/tecnickcom/go-tcpdf/unicode"
)

// Config holds constructor-time options.
type Config struct {
	Unit        string
	Format      string
	Orientation page.Orientation
	Margins     page.Margins
	RTL         bool
	Unicode     bool
	Encoding    string
	SubsetFonts bool
	Compress    bool
}

// DefaultConfig returns sensible defaults (A4, mm, portrait, unicode).
func DefaultConfig() Config {
	return Config{
		Unit:        "mm",
		Format:      "A4",
		Orientation: page.Portrait,
		Margins: page.Margins{
			Top: 10, Right: 10, Bottom: 10, Left: 10,
		},
		RTL:         false,
		Unicode:     true,
		Encoding:    "UTF-8",
		SubsetFonts: true,
		Compress:    true,
	}
}

// ClassObjects holds all sub-objects of a TCPDF document.
type ClassObjects struct {
	Base       *base.Base
	Cell       *cell.Cell
	Meta       *metainfo.MetaInfo
	CSS        *css.CSS
	HTML       *html.HTML
	SVG        *svg.SVG
	Text       *text.Text
	JS         *javascript.JavaScript
	Layers     *layers.Layers
	TOC        *toc.TOC
	Cache      *cache.Cache
	Color      *color.SpotRegistry
	Encrypt    *encrypt.Encrypt  // nil until SetEncryption is called
	Fonts      *font.Stack
	Graph      *graph.Draw
	Images     *imgpkg.Import
	Page       *page.Page
	UniConv    *uniconv.Convert
	ICCProfile *color.ICCProfile
}

// InitClassObjects initializes all sub-objects with the given configuration.
func InitClassObjects(cfg Config) (*ClassObjects, error) {
	b, err := base.New(cfg.Unit)
	if err != nil {
		return nil, err
	}
	b.SetRTL(cfg.RTL)

	k, _ := base.PointsPerUnit[cfg.Unit]
	pageManager, err := page.New(
		page.Unit(cfg.Unit),
		cfg.Format,
		cfg.Orientation,
		cfg.Margins,
	)
	if err != nil {
		return nil, err
	}

	fonts := font.NewStack(cfg.SubsetFonts)
	uniCnv := uniconv.NewConvert(cfg.RTL)
	grph := graph.NewDraw()
	imgs := imgpkg.NewImport()
	spots := color.NewSpotRegistry()
	metaObj := metainfo.New()
	cellObj := cell.New()
	cssObj := css.New()
	htmlObj := html.New()
	svgObj := svg.New(k)
	textObj := text.New(fonts, uniCnv)
	jsObj := javascript.New(b.PON)
	layersObj := layers.New(b.PON)
	tocObj := toc.New()
	cacheObj := cache.New()

	return &ClassObjects{
		Base:    b,
		Cell:    cellObj,
		Meta:    metaObj,
		CSS:     cssObj,
		HTML:    htmlObj,
		SVG:     svgObj,
		Text:    textObj,
		JS:      jsObj,
		Layers:  layersObj,
		TOC:     tocObj,
		Cache:   cacheObj,
		Color:   spots,
		Fonts:   fonts,
		Graph:   grph,
		Images:  imgs,
		Page:    pageManager,
		UniConv: uniCnv,
	}, nil
}
