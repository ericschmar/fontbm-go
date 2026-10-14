package fontbm

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
)

func testFont(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/Go-Regular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestGenerate(t *testing.T) {
	result, err := Generate(Options{FontData: testFont(t), FontName: "fixture", PageName: "fixture_0.png", Chars: "Ag 3A", PixelSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Glyphs) != 4 {
		t.Fatalf("got %d glyphs; want 4", len(result.Glyphs))
	}
	if !strings.Contains(result.Descriptor, "chars count=4") || !strings.Contains(result.Descriptor, `page id=0 file="fixture_0.png"`) || !strings.Contains(result.Descriptor, "chnl=7") || !strings.Contains(result.Descriptor, "alphaChnl=3 redChnl=0 greenChnl=0 blueChnl=0") {
		t.Fatalf("unexpected descriptor:\n%s", result.Descriptor)
	}
	img, err := png.Decode(bytes.NewReader(result.PNG))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != result.Width || img.Bounds().Dy() != result.Height {
		t.Fatalf("decoded atlas dimensions %v disagree with result %dx%d", img.Bounds(), result.Width, result.Height)
	}
	for y := 0; y < result.Height; y++ {
		for x := 0; x < result.Width; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a != 0xffff || r != g || g != b {
				t.Fatalf("pixel (%d,%d) is not opaque grayscale: %04x %04x %04x %04x", x, y, r, g, b, a)
			}
		}
	}
	if got := img.At(0, 0); got != (color.Gray{Y: 0}) {
		t.Fatalf("atlas background pixel = %v; want black", got)
	}
	glyph := result.Glyphs[0]
	brightest := uint32(0)
	for y := glyph.Y; y < glyph.Y+glyph.Height; y++ {
		for x := glyph.X; x < glyph.X+glyph.Width; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if r > brightest {
				brightest = r
			}
		}
	}
	if brightest == 0 {
		t.Fatal("glyph has no white or antialiased pixels")
	}
	for _, glyph := range result.Glyphs {
		if glyph.ID == ' ' && (glyph.Width != 0 || glyph.Height != 0 || glyph.XAdvance == 0) {
			t.Fatalf("unexpected space metrics: %+v", glyph)
		}
	}
}

func TestGenerateRGB(t *testing.T) {
	result, err := Generate(Options{FontData: testFont(t), Chars: "Ag", PixelSize: 24, ColorMode: "rgb"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Descriptor, "alphaChnl=4 redChnl=0 greenChnl=0 blueChnl=0") {
		t.Fatalf("unexpected RGB descriptor channels:\n%s", result.Descriptor)
	}
	img, err := png.Decode(bytes.NewReader(result.PNG))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := img.(*image.NRGBA); !ok {
		if _, ok := img.(*image.RGBA); !ok {
			t.Fatalf("got PNG type %T; want RGB/RGBA", img)
		}
	}
	for y := 0; y < result.Height; y++ {
		for x := 0; x < result.Width; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0xffff {
				t.Fatalf("pixel (%d,%d) has non-opaque alpha", x, y)
			}
		}
	}
}

func TestGenerateRejectsInvalidColorMode(t *testing.T) {
	_, err := Generate(Options{FontData: testFont(t), Chars: "A", PixelSize: 12, ColorMode: "cmyk"})
	if err == nil || !strings.Contains(err.Error(), "color mode") {
		t.Fatalf("got error %v; want invalid color mode error", err)
	}
}

func TestGenerateRejectsMissingCharacter(t *testing.T) {
	_, err := Generate(Options{FontData: testFont(t), Chars: "☃", PixelSize: 24})
	if err == nil || !strings.Contains(err.Error(), "does not contain") {
		t.Fatalf("got error %v; want missing-character error", err)
	}
}

func TestGenerateRejectsAtlasOverflow(t *testing.T) {
	_, err := Generate(Options{FontData: testFont(t), Chars: "AB", PixelSize: 24, MaxAtlasSize: 8})
	if err == nil || !strings.Contains(err.Error(), "maximum atlas size") {
		t.Fatalf("got error %v; want atlas overflow", err)
	}
}

func TestGenerateUsesPrintableASCIIByDefault(t *testing.T) {
	result, err := Generate(Options{FontData: testFont(t), PixelSize: 12})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Glyphs) != len([]rune(DefaultChars)) {
		t.Fatalf("got %d glyphs; want %d", len(result.Glyphs), len([]rune(DefaultChars)))
	}
}

func TestReadCharsRequiresUTF8(t *testing.T) {
	if _, err := ReadChars([]byte{0xff}); err == nil {
		t.Fatal("expected invalid UTF-8 error")
	}
}
