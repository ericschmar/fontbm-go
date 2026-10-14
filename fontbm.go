package fontbm

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const DefaultChars = " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"

type Options struct {
	FontData     []byte
	FontName     string
	PageName     string
	Chars        string
	PixelSize    float64
	MaxAtlasSize int
	ColorMode    string
}

type Glyph struct {
	ID       rune
	X, Y     int
	Width    int
	Height   int
	XOffset  int
	YOffset  int
	XAdvance int
	Page     int
	Channel  int
}

type Result struct {
	Descriptor string
	PNG        []byte
	Width      int
	Height     int
	Glyphs     []Glyph
}

type renderedGlyph struct {
	glyph Glyph
	mask  *image.Alpha
}

func Generate(opts Options) (*Result, error) {
	if len(opts.FontData) == 0 {
		return nil, fmt.Errorf("font data is empty")
	}
	if opts.PixelSize <= 0 || math.IsNaN(opts.PixelSize) || math.IsInf(opts.PixelSize, 0) {
		return nil, fmt.Errorf("pixel size must be positive and finite")
	}
	if opts.MaxAtlasSize == 0 {
		opts.MaxAtlasSize = 4096
	}
	if opts.MaxAtlasSize < 1 {
		return nil, fmt.Errorf("maximum atlas size must be positive")
	}
	if opts.ColorMode == "" {
		opts.ColorMode = "grayscale"
	}
	if opts.ColorMode != "grayscale" && opts.ColorMode != "rgb" {
		return nil, fmt.Errorf("color mode must be grayscale or rgb")
	}
	if opts.Chars == "" {
		opts.Chars = DefaultChars
	}
	if opts.FontName == "" {
		opts.FontName = "font"
	}
	if opts.PageName == "" {
		opts.PageName = "font_0.png"
	}
	parsed, err := opentype.Parse(opts.FontData)
	if err != nil {
		return nil, fmt.Errorf("parse TTF: %w", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: opts.PixelSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, fmt.Errorf("create font face: %w", err)
	}
	defer face.Close()
	fontData, err := sfnt.Parse(opts.FontData)
	if err != nil {
		return nil, fmt.Errorf("parse TTF character map: %w", err)
	}
	var buf sfnt.Buffer
	seen := make(map[rune]bool)
	var glyphs []renderedGlyph
	for _, r := range opts.Chars {
		if seen[r] {
			continue
		}
		seen[r] = true
		index, err := fontData.GlyphIndex(&buf, r)
		if err != nil || index == 0 {
			return nil, fmt.Errorf("font does not contain %q (U+%04X)", r, r)
		}
		drawRect, mask, maskPoint, advance, ok := face.Glyph(fixed.Point26_6{}, r)
		if !ok {
			return nil, fmt.Errorf("rasterize %q (U+%04X)", r, r)
		}
		alpha, ok := mask.(*image.Alpha)
		if !ok {
			return nil, fmt.Errorf("unexpected raster mask for %q", r)
		}
		var copyMask *image.Alpha
		if drawRect.Dx() > 0 && drawRect.Dy() > 0 {
			copyMask = image.NewAlpha(image.Rect(0, 0, drawRect.Dx(), drawRect.Dy()))
			draw.Draw(copyMask, copyMask.Bounds(), alpha, maskPoint, draw.Src)
		} else {
			copyMask = image.NewAlpha(image.Rectangle{})
		}
		glyphs = append(glyphs, renderedGlyph{glyph: Glyph{ID: r, Width: drawRect.Dx(), Height: drawRect.Dy(), XOffset: drawRect.Min.X, YOffset: drawRect.Min.Y, XAdvance: advance.Round(), Page: 0, Channel: 7}, mask: copyMask})

	}
	metrics := face.Metrics()
	base := metrics.Ascent.Round()
	lineHeight := metrics.Height.Round()
	if lineHeight < 1 {
		lineHeight = 1
	}
	width, height, err := pack(glyphs, opts.MaxAtlasSize)
	if err != nil {
		return nil, err
	}
	var atlas image.Image
	if opts.ColorMode == "grayscale" {
		grayAtlas := image.NewGray(image.Rect(0, 0, width, height))

		atlas = grayAtlas
	} else {
		rgbAtlas := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				rgbAtlas.SetRGBA(x, y, color.RGBA{A: 255})
			}
		}
		atlas = rgbAtlas
	}
	resultGlyphs := make([]Glyph, len(glyphs))
	for i, g := range glyphs {
		resultGlyphs[i] = g.glyph
		if g.glyph.Width == 0 || g.glyph.Height == 0 {
			continue
		}
		for y := 0; y < g.glyph.Height; y++ {
			for x := 0; x < g.glyph.Width; x++ {
				v := g.mask.AlphaAt(x, y).A
				if grayAtlas, ok := atlas.(*image.Gray); ok {
					grayAtlas.SetGray(g.glyph.X+x, g.glyph.Y+y, color.Gray{Y: v})
				} else {
					atlas.(*image.RGBA).SetRGBA(g.glyph.X+x, g.glyph.Y+y, color.RGBA{R: v, G: v, B: v, A: 255})
				}
			}
		}
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, atlas); err != nil {
		return nil, fmt.Errorf("encode PNG atlas: %w", err)
	}
	var descriptor strings.Builder
	fmt.Fprintf(&descriptor, "info face=%q size=%d bold=0 italic=0 charset=\"\" unicode=1 stretchH=100 smooth=1 aa=1 padding=0,0,0,0 spacing=0,0 outline=0\n", opts.FontName, int(math.Round(opts.PixelSize)))
	if opts.ColorMode == "grayscale" {
		fmt.Fprintf(&descriptor, "common lineHeight=%d base=%d scaleW=%d scaleH=%d pages=1 packed=0 alphaChnl=3 redChnl=0 greenChnl=0 blueChnl=0\n", lineHeight, base, width, height)
	} else {
		fmt.Fprintf(&descriptor, "common lineHeight=%d base=%d scaleW=%d scaleH=%d pages=1 packed=0 alphaChnl=4 redChnl=0 greenChnl=0 blueChnl=0\n", lineHeight, base, width, height)
	}
	fmt.Fprintf(&descriptor, "page id=0 file=%q\nchars count=%d\n", opts.PageName, len(resultGlyphs))
	for _, g := range resultGlyphs {
		yOffset := g.YOffset + base
		fmt.Fprintf(&descriptor, "char id=%d x=%d y=%d width=%d height=%d xoffset=%d yoffset=%d xadvance=%d page=0 chnl=7\n", g.ID, g.X, g.Y, g.Width, g.Height, g.XOffset, yOffset, g.XAdvance)
	}
	return &Result{Descriptor: descriptor.String(), PNG: pngData.Bytes(), Width: width, Height: height, Glyphs: resultGlyphs}, nil
}

func pack(glyphs []renderedGlyph, maxSize int) (int, int, error) {
	for _, g := range glyphs {
		if g.glyph.Width > maxSize || g.glyph.Height > maxSize {
			return 0, 0, fmt.Errorf("glyph %q exceeds maximum atlas size %d", g.glyph.ID, maxSize)
		}
	}
	for size := 1; size <= maxSize; {

		x, y, rowHeight := 1, 1, 0
		fits := true
		for i := range glyphs {
			w, h := glyphs[i].glyph.Width+2, glyphs[i].glyph.Height+2
			if w <= 2 && h <= 2 {
				glyphs[i].glyph.X, glyphs[i].glyph.Y = 0, 0
				continue
			}
			if x+w > size {
				x, y, rowHeight = 1, y+rowHeight, 0
			}
			if y+h > size {
				fits = false
				break
			}
			glyphs[i].glyph.X, glyphs[i].glyph.Y = x+1, y+1
			x += w
			if h > rowHeight {
				rowHeight = h
			}
		}
		if fits {
			return size, size, nil
		}
		if size > math.MaxInt/2 {
			break
		}
		size *= 2
	}
	return 0, 0, fmt.Errorf("glyphs do not fit in a single atlas of maximum size %d", maxSize)
}

func ReadChars(data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", fmt.Errorf("character file is not valid UTF-8")
	}
	return string(data), nil
}
