// Package image provides PDF image import (JPEG, PNG).
package image

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tecnickcom/go-tcpdf/pdfbuf"
)

func ColorSpaceName(channels int) string {
	switch channels {
	case 1:
		return "DeviceGray"
	case 4:
		return "DeviceCMYK"
	default:
		return "DeviceRGB"
	}
}

type ImageFormat string

const (
	FormatJPEG ImageFormat = "jpeg"
	FormatPNG  ImageFormat = "png"
	FormatGIF  ImageFormat = "gif"
	FormatRAW  ImageFormat = "raw"
)

type PDFImage struct {
	Key          string
	ObjNum       int
	Width        int
	Height       int
	Channels     int
	BitsPerComp  int
	ColorSpace   string
	Data         []byte
	Filter       string
	DecodeParms  string
	Palette      []byte
	MaskObjNum   int
	ColorKeyMask []int
	HasAlpha     bool
	Format       ImageFormat
	DPI          float64
	VRes         float64
	SrcPath      string
	SrcHash      string
}

// Import manages images for a PDF document. Thread-safe.
type Import struct {
	mu      sync.RWMutex
	images  map[string]*PDFImage
	counter int
}

func NewImport() *Import {
	return &Import{images: make(map[string]*PDFImage)}
}

// LoadFile loads an image from disk and registers it.
func (imp *Import) LoadFile(path string) (string, *PDFImage, error) {
	imp.mu.RLock()
	for k, img := range imp.images {
		if img.SrcPath == path {
			imp.mu.RUnlock()
			return k, img, nil
		}
	}
	imp.mu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, errors.New("image: cannot read " + path + ": " + err.Error())
	}
	return imp.LoadBytes(data, path)
}

// LoadBytes parses raw image bytes and registers the image.
func (imp *Import) LoadBytes(data []byte, hint string) (string, *PDFImage, error) {
	format := detectFormat(data, hint)
	var img *PDFImage
	var err error
	switch format {
	case FormatJPEG:
		img, err = parseJPEG(data)
	case FormatPNG:
		img, err = parsePNG(data)
	default:
		return "", nil, errors.New("image: unsupported format: " + string(format))
	}
	if err != nil {
		return "", nil, err
	}
	img.Format = format
	img.SrcPath = hint

	imp.mu.Lock()
	defer imp.mu.Unlock()
	imp.counter++
	var kb strings.Builder
	kb.WriteByte('I')
	kb.WriteString(pdfbuf.FmtI(imp.counter))
	key := kb.String()
	img.Key = key
	imp.images[key] = img
	return key, img, nil
}

// Get returns an image by key.
func (imp *Import) Get(key string) (*PDFImage, bool) {
	imp.mu.RLock()
	defer imp.mu.RUnlock()
	img, ok := imp.images[key]
	return img, ok
}

// All returns a snapshot copy of all registered images.
func (imp *Import) All() map[string]*PDFImage {
	imp.mu.RLock()
	defer imp.mu.RUnlock()
	cp := make(map[string]*PDFImage, len(imp.images))
	for k, v := range imp.images {
		cp[k] = v
	}
	return cp
}

// ---- JPEG ---------------------------------------------------------------

func parseJPEG(data []byte) (*PDFImage, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("image: JPEG decode config: " + err.Error())
	}
	channels := colorModelChannels(cfg.ColorModel)
	return &PDFImage{
		Width:       cfg.Width,
		Height:      cfg.Height,
		Channels:    channels,
		BitsPerComp: 8,
		ColorSpace:  ColorSpaceName(channels),
		Data:        data,
		Filter:      "DCTDecode",
	}, nil
}

// ---- PNG ----------------------------------------------------------------

func parsePNG(data []byte) (*PDFImage, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("image: PNG decode: " + err.Error())
	}
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	var raw []byte
	var alphaChan []byte
	var channels, bpc int

	switch img := src.(type) {
	case *image.NRGBA:
		channels, bpc = 3, 8
		raw = make([]byte, w*h*3)
		alphaChan = make([]byte, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := img.NRGBAAt(x, y)
				off := (y*w + x) * 3
				raw[off] = c.R; raw[off+1] = c.G; raw[off+2] = c.B
				alphaChan[y*w+x] = c.A
			}
		}
	case *image.RGBA:
		channels, bpc = 3, 8
		raw = make([]byte, w*h*3)
		alphaChan = make([]byte, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := img.RGBAAt(x, y)
				off := (y*w + x) * 3
				raw[off] = c.R; raw[off+1] = c.G; raw[off+2] = c.B
				alphaChan[y*w+x] = c.A
			}
		}
	case *image.Gray:
		channels, bpc = 1, 8
		raw = make([]byte, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				raw[y*w+x] = img.GrayAt(x, y).Y
			}
		}
	default:
		channels, bpc = 3, 8
		raw = make([]byte, w*h*3)
		alphaChan = make([]byte, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r32, g32, b32, a32 := src.At(x, y).RGBA()
				off := (y*w + x) * 3
				raw[off] = byte(r32 >> 8)
				raw[off+1] = byte(g32 >> 8)
				raw[off+2] = byte(b32 >> 8)
				alphaChan[y*w+x] = byte(a32 >> 8)
			}
		}
	}

	compressed, err := zlibCompress(raw)
	if err != nil {
		return nil, err
	}

	// Build DecodeParms without fmt.Sprintf
	var dp strings.Builder
	dp.WriteString("<< /Predictor 1 /Colors ")
	dp.WriteString(pdfbuf.FmtI(channels))
	dp.WriteString(" /BitsPerComponent ")
	dp.WriteString(pdfbuf.FmtI(bpc))
	dp.WriteString(" /Columns ")
	dp.WriteString(pdfbuf.FmtI(w))
	dp.WriteString(" >>")

	pi := &PDFImage{
		Width:       w,
		Height:      h,
		Channels:    channels,
		BitsPerComp: bpc,
		ColorSpace:  ColorSpaceName(channels),
		Data:        compressed,
		Filter:      "FlateDecode",
		DecodeParms: dp.String(),
	}

	if alphaChan != nil {
		hasAlpha := false
		for _, a := range alphaChan {
			if a != 255 {
				hasAlpha = true
				break
			}
		}
		if hasAlpha {
			pi.HasAlpha = true
		}
	}
	return pi, nil
}

// ---- helpers ------------------------------------------------------------

func detectFormat(data []byte, hint string) ImageFormat {
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return FormatJPEG
	}
	if len(data) >= 8 && string(data[1:4]) == "PNG" {
		return FormatPNG
	}
	if len(data) >= 3 && string(data[0:3]) == "GIF" {
		return FormatGIF
	}
	switch strings.ToLower(filepath.Ext(hint)) {
	case ".jpg", ".jpeg":
		return FormatJPEG
	case ".png":
		return FormatPNG
	case ".gif":
		return FormatGIF
	}
	return FormatRAW
}

func zlibCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(w, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func colorModelChannels(m color.Model) int {
	switch m {
	case color.GrayModel, color.Gray16Model:
		return 1
	case color.CMYKModel:
		return 4
	default:
		return 3
	}
}

type pngChunkReader struct{ r io.Reader }

func (r *pngChunkReader) ReadChunk() (typ string, data []byte, err error) {
	var length uint32
	if err = binary.Read(r.r, binary.BigEndian, &length); err != nil {
		return
	}
	typBuf := make([]byte, 4)
	if _, err = io.ReadFull(r.r, typBuf); err != nil {
		return
	}
	typ = string(typBuf)
	data = make([]byte, length)
	if _, err = io.ReadFull(r.r, data); err != nil {
		return
	}
	crc := make([]byte, 4)
	_, err = io.ReadFull(r.r, crc)
	return
}
